package helm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"

	"github.com/datawire/ambassador/v2/pkg/k8s"
)

const (
	// The default URL for getting the charts listing
	DefaultHelmRepoURL = "https://www.getambassador.io"

	// The default chart name
	DefaultChartName = "ambassador"
)

var (
	// ErrUnknownHelmRepoScheme is unknown helm repo scheme
	ErrUnknownHelmRepoScheme = errors.New("unknown helm repo scheme")

	// ErrNoChartDirFound is no chart directory found
	ErrNoChartDirFound = errors.New("no chart directory found")
)

// TODO: we could replace this with https://github.com/helm/helm/tree/master/pkg/downloader

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// fileIsArchive returns True if the URL points to an archive
func fileIsArchive(u url.URL) bool {
	path := u.Path
	ext := filepath.Ext(path)

	switch ext {
	case ".tar.gz", ".gz", ".zip":
		return true
	default:
		return false
	}
}

// HelmDownloader is a downloader for a remote Helm repo or a file, provided with an URL
type HelmDownloader struct {
	URL     *url.URL
	Version ChartVersionRule

	KubeInfo *k8s.KubeInfo

	// The chart downloaded (the Chart.yaml file as well as the metadata)
	downChartFile string
	downChart     *chart.Metadata
	DownChartDir  string

	// Directory where the chart will be / has been downloaded (the chart will be in a subdirectory inside)
	downDir        string
	downDirCleanup bool

	log *log.Logger
}

// HelmDownloaderConfig specifies options for creating the Helm manager
type HelmDownloaderConfig struct {
	URL      string
	KubeInfo *k8s.KubeInfo
	Version  ChartVersionRule
	Logger   *log.Logger
}

// NewHelmDownloader creates a new charts manager
// The Helm Manager will use the URL provided, and download (lazily) a Chart that
// obeys the Version Rule.
func NewHelmDownloader(options HelmDownloaderConfig) (HelmDownloader, error) {
	// process the URL, using the default URL when not provided
	if options.URL == "" {
		options.URL = DefaultHelmRepoURL
	}
	pu, err := url.Parse(options.URL)
	if err != nil {
		return HelmDownloader{}, err
	}

	if options.Logger == nil {
		options.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	return HelmDownloader{
		URL:      pu,
		KubeInfo: options.KubeInfo,
		Version:  options.Version,
		log:      options.Logger,
	}, nil
}

// GetChart returns the metadata about the Chart that has been downloaded
// from URL with the given constraints (like the Version)
func (lc HelmDownloader) GetChart() *chart.Metadata {
	if lc.downDir == "" {
		panic(fmt.Errorf("must Download() before trying to get the chart"))
	}

	return lc.downChart
}

// GetValues returns the version rules associated with this Helm manager
func (lc HelmDownloader) GetVersionRule() ChartVersionRule {
	return lc.Version
}

// GetReleaseMgr returns a manager for the latest
func (lc *HelmDownloader) Download() error {
	var err error

	// parse the helm repo URL and try to download the helm chart
	switch lc.URL.Scheme {
	case "http", "https":
		if fileIsArchive(*lc.URL) {
			lc.log.Printf("URL points to an archive: downloading")
			if err := lc.downloadChartFile(lc.URL); err != nil {
				return err
			}
		} else {
			lc.log.Printf("URL is a Helm repo: looking for version in repo")
			u, version, err := lc.findInRepo()
			if err != nil {
				return err
			}

			lc.log.Printf("Downloading release %s from %q", version, u)
			if err := lc.downloadChartFile(u); err != nil {
				return err
			}
		}

		lc.log.Printf("Finding chart")
		if err = lc.lookupChart(); err != nil {
			return err
		}

	case "file", "":
		lc.downDir = lc.URL.String()
		lc.log.Printf("Finding chart in %s", lc.URL.String())
		if err = lc.lookupChart(); err != nil {
			return err
		}

	default:
		return fmt.Errorf("%w: scheme %q in %q", ErrUnknownHelmRepoScheme, lc.URL.Scheme, lc.URL.String())
	}

	return nil
}

// FindLatestVersionAvailable returns the latest version available in the
// repository (or file) specified in the `helmRepo`.
func (lc *HelmDownloader) FindLatestVersionAvailable() (string, error) {
	var err error
	var version string

	// parse the helm repo URL and try to download the helm chart
	switch lc.URL.Scheme {
	case "http", "https":
		if fileIsArchive(*lc.URL) {
			lc.log.Printf("URL points to an archive: we must download the file")
			if err := lc.downloadChartFile(lc.URL); err != nil {
				return "", err
			}
			lc.log.Printf("Finding version of Chart in file")
			if err = lc.lookupChart(); err != nil {
				return "", err
			}
			version = lc.downChart.AppVersion
			lc.log.Printf("Latest version in remote file: %q", version)
		} else {
			lc.log.Printf("URL is a Helm repo: looking for latest version in repo")
			_, version, err = lc.findInRepo()
			if err != nil {
				return "", err
			}
			lc.log.Printf("Latest version in repo: %q", version)
		}

	case "file", "":
		lc.downDir = lc.URL.String()
		lc.log.Printf("Finding chart in local file %q", lc.URL.String())
		if err = lc.lookupChart(); err != nil {
			return "", err
		}
		version = lc.downChart.AppVersion
		lc.log.Printf("Latest version in local file: %q", version)

	default:
		return "", fmt.Errorf("%w: scheme %q in %q", ErrUnknownHelmRepoScheme, lc.URL.Scheme, lc.URL.String())
	}

	return version, nil
}

// Cleanup removed all the download directories
func (lc *HelmDownloader) Cleanup() error {
	cleanup := lc.downDirCleanup
	if d := os.Getenv("DEBUG"); d != "" {
		cleanup = false
	}
	if lc.downDir != "" && cleanup {
		lc.log.Printf("Removing downloads directory %q", lc.downDir)
		_ = os.RemoveAll(lc.downDir)
	}
	lc.downDir = ""
	lc.downChartFile = ""
	lc.DownChartDir = ""
	lc.downChart = nil
	return nil
}

// downloadChartFile downloads a Chart archive from a URL
func (lc *HelmDownloader) downloadChartFile(url *url.URL) error {
	// creates/erases the downloads directory, ignoring any error (just in case it does not exist)
	d, err := ioutil.TempDir("", "chart-download")
	if err != nil {
		return err
	}
	lc.downDir = d
	lc.downDirCleanup = true

	filename := filepath.Base(url.Path)

	// generates a random filename in /tmp (but it does not create the file)
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	tempFilename := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", hex.EncodeToString(randBytes), filename))

	lc.log.Printf("Downloading file %q (temp=%q) (dest=%q)", url, tempFilename, lc.downDir)
	if err := downloadFile(tempFilename, url.String()); err != nil {
		return err
	}
	defer func() { _ = os.Remove(tempFilename) }()

	lc.log.Printf("Uncompressing file")
	if err := archiver.Unarchive(tempFilename, lc.downDir); err != nil {
		return err
	}
	lc.log.Printf("File uncompressed")

	return nil
}

func (lc *HelmDownloader) findInRepo() (*url.URL, string, error) {
	chartName := DefaultChartName
	repoURL := lc.URL.String()

	// Download and write the index file to a temporary location
	tempIndexFile, err := ioutil.TempFile("", "tmp-repo-file")
	if err != nil {
		return nil, "", fmt.Errorf("cannot write index file for repository requested")
	}
	defer func() { _ = os.Remove(tempIndexFile.Name()) }()

	home := helmpath.Home(environment.DefaultHelmHome)
	settings := environment.EnvSettings{
		Home: home,
	}

	c := repo.Entry{
		URL: repoURL,
	}
	r, err := repo.NewChartRepository(&c, getter.All(settings))
	if err != nil {
		return nil, "", err
	}
	if err := r.DownloadIndexFile(tempIndexFile.Name()); err != nil {
		return nil, "", fmt.Errorf("looks like %q is not a valid chart repository or cannot be reached: %s", repoURL, err)
	}

	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := repo.LoadIndexFile(tempIndexFile.Name())
	if err != nil {
		return nil, "", err
	}

	versions, ok := repoIndex.Entries[chartName]
	if !ok {
		return nil, "", repo.ErrNoChartName
	}
	if len(versions) == 0 {
		return nil, "", repo.ErrNoChartVersion
	}

	parsedURL := func(u string) (*url.URL, error) {
		absoluteChartURL, err := repo.ResolveReferenceURL(repoURL, u)
		if err != nil {
			return nil, fmt.Errorf("failed to make chart URL absolute: %v", err)
		}

		lc.log.Printf("Chart URL %q", absoluteChartURL)
		pu, err := url.Parse(absoluteChartURL)
		if err != nil {
			return nil, err
		}
		return pu, nil
	}

	//
	// note: when looking for the right chart, there are two versions to consider:
	//
	// - the AppVersion is the version of the software **installed by** the Chart (ie, Ambassador 1.0)
	// - the Version is the version of the Chart (ie, Ambassador Chart 0.6)
	//
	// So there can be multiple Chart Versions for the same `AppVersion`. For example, we updated
	// the Helm Chart several times for AppVersion=1.0 (AES) because there were some changes
	// in the templates, etc... So once we have a valid/latest `AppVersion`, we must get the chart
	// with the highest `Version`.
	//
	var latest *repo.ChartVersion
	for _, curVer := range versions {
		allowed, err := lc.Version.Allowed(curVer.AppVersion)
		if err != nil {
			return nil, "", fmt.Errorf("%w while checking if allowed for %s", err, lc.Version)
		}
		if !allowed {
			lc.log.Printf("Chart not allowed by version constraint: version=%q, required=%q", curVer.AppVersion, lc.Version)
			continue
		}
		if len(curVer.URLs) == 0 {
			return nil, "", fmt.Errorf("no URL found for %s-%s", chartName, lc.Version)
		}

		// no previous `latest` chart: use this one
		if latest == nil {
			latest = curVer
			continue
		}

		// compare the versions: first, the `AppVersion`, and then the `Chart` version
		if moreRecent, err := MoreRecentThan(curVer.AppVersion, latest.AppVersion); err == nil && moreRecent {
			lc.log.Printf("Updating 'latest chart version' to %q", curVer)
			latest = curVer
		} else if equal, err := Equal(curVer.AppVersion, latest.AppVersion); err == nil && equal {
			// if this chart has the same version of Ambassador, then check if it is a more recent Chart
			if moreRecent, err := MoreRecentThan(curVer.Version, latest.Version); err == nil && moreRecent {
				lc.log.Printf("Updating 'latest chart version' to %q", curVer)
				latest = curVer
			}
		}
	}
	if latest != nil {
		u, err := parsedURL(latest.URLs[0])
		return u, latest.AppVersion, err
	}

	return nil, "", fmt.Errorf("no chart version found for %s-%s", chartName, lc.Version)
}

// lookupChart looks for the chart in a directory or subdirectory that can contain a Chart
func (lc *HelmDownloader) lookupChart() error {
	res := ""

	if lc.downDir == "" {
		panic(fmt.Errorf("no downloads directory: must Download() before trying to find the chart"))
	}

	_ = filepath.Walk(lc.downDir, func(path string, info os.FileInfo, err error) error {
		if res != "" {
			return nil
		}
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			lc.log.Printf("Looking for Chart in directory. directory=%s", path)
			if validChart, _ := chartutil.IsChartDir(path); validChart {
				lc.log.Printf("Directory contains a Chart")
				res = path
			}
		}
		return nil
	})

	if res == "" {
		return fmt.Errorf("%w: %q", ErrNoChartDirFound, lc.downDir)
	}

	lc.log.Printf("Chart directory found in %q", res)
	chartFile := filepath.Join(res, "Chart.yaml")

	chart, err := chartutil.LoadChartfile(chartFile)
	if err != nil {
		return fmt.Errorf("%w: could no load chart from %q", ErrNoChartDirFound, chartFile)
	}

	lc.downChart = chart
	lc.DownChartDir = res
	lc.downChartFile = chartFile
	return nil
}
