// Package metriton implements submitting telemetry data to the Metriton database.
//
// Metriton replaced Scout, and was originally going to have its own telemetry API and a
// Scout-compatibility endpoint during the migration.  But now the Scout-compatible API is
// the only thing we use.
//
// See also: The old scout.py package <https://pypi.org/project/scout.py/> /
// <https://github.com/datawire/scout.py>.
//
// Things that are in scout.py, but are intentionally left of this package:
//  - automatically setting the HTTP client user-agent string
//  - an InstallIDFromConfigMap getter
package metriton

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	// The endpoint you should use by default
	DefaultEndpoint = "https://metriton.datawire.io/scout"
	// Use BetaEndpoint for testing purposes without polluting production data
	BetaEndpoint = "https://metriton.datawire.io/beta/scout"
	// ScoutPyEndpoint is the default endpoint used by scout.py; it obeys the
	// SCOUT_HOST and SCOUT_HTTPS environment variables.  I'm not sure when you should
	// use it instead of DefaultEndpoint, but I'm putting it in Go so that I never
	// have to look at scout.py again.
	ScoutPyEndpoint = endpointFromEnv()
)

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

func endpointFromEnv() string {
	host := getenvDefault("SCOUT_HOST", "metriton.datawire.io")
	useHTTPS, _ := strconv.ParseBool(getenvDefault("SCOUT_HTTPS", "1"))
	ret := &url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/scout",
	}
	if useHTTPS {
		ret.Scheme = "https"
	}
	return ret.String()
}

// Reporter is a client to
type Reporter struct {
	// Information about the application submitting telemetry.
	Application string
	Version     string
	// GetInstallID is a function, instead of a fixed 'InstallID' string, in order to
	// facilitate getting it lazily; and possibly updating the BaseMetadata based on
	// the journey to getting the install ID.  See StaticInstallID and
	// InstallIDFromFilesystem.
	GetInstallID func(*Reporter) (string, error)
	// BaseMetadata will be merged in to the data passed to each call to .Report().
	// If the data passed to .Report() and BaseMetadata have a key in common, the
	// value passed as an argument to .Report() wins.
	BaseMetadata map[string]interface{}

	// The HTTP client used to to submit the request; if this is nil, then
	// http.DefaultClient is used.
	Client *http.Client
	// The endpoint URL to submit to; if this is empty, then DefaultEndpoint is used.
	Endpoint string

	mu          sync.Mutex
	initialized bool
	disabled    bool
	installID   string
}

func (r *Reporter) ensureInitialized() error {
	if r.Application == "" {
		return errors.New("Reporter.Application may not be empty")
	}
	if r.Version == "" {
		return errors.New("Reporter.Version may not be empty")
	}
	if r.GetInstallID == nil {
		return errors.New("Reporter.GetInstallID may not be nil")
	}

	if r.initialized {
		return nil
	}

	if r.BaseMetadata == nil {
		r.BaseMetadata = make(map[string]interface{})
	}

	r.disabled = IsDisabledByUser()

	installID, err := r.GetInstallID(r)
	if err != nil {
		return err
	}
	r.installID = installID

	r.initialized = true

	return nil
}

// IsDisabledByUser returns whether telemetry reporting is disabled by the user.
func IsDisabledByUser() bool {
	// From scout.py
	if strings.HasPrefix(os.Getenv("TRAVIS_REPO_SLUG"), "datawire/") {
		return true
	}

	// This mimics the existing ad-hoc Go clients, rather than scout.py; it is a more
	// sensitive trigger than scout.py's __is_disabled() (which only accepts "1",
	// "true", "yes"; case-insensitive).
	return os.Getenv("SCOUT_DISABLE") != ""
}

func (r *Reporter) InstallID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.ensureInitialized()
	return r.installID
}

// Report submits a telemetry report to Metriton.  It is safe to call .Report() from
// different goroutines.  It is NOT safe to mutate the public fields in the Reporter while
// .Report() is being called.
func (r *Reporter) Report(ctx context.Context, metadata map[string]interface{}) (*Response, error) {
	r.mu.Lock()

	if err := r.ensureInitialized(); err != nil {
		r.mu.Unlock()
		return nil, err
	}

	var resp *Response
	var err error

	if r.disabled {
		r.mu.Unlock()
	} else {
		client := r.Client
		if client == nil {
			client = http.DefaultClient
		}

		endpoint := r.Endpoint
		if endpoint == "" {
			endpoint = DefaultEndpoint
		}

		mergedMetadata := make(map[string]interface{}, len(r.BaseMetadata)+len(metadata))
		// FWIW, the resolution of conflicts between 'r.BaseMetadata' and 'metadata'
		// mimics scout.py; I'm not sure whether that aspect of scout.py's API is
		// intentional or incidental.
		for k, v := range r.BaseMetadata {
			mergedMetadata[k] = v
		}
		for k, v := range metadata {
			mergedMetadata[k] = v
		}

		report := Report{
			Application: r.Application,
			InstallID:   r.installID,
			Version:     r.Version,
			Metadata:    mergedMetadata,
		}

		r.mu.Unlock()
		resp, err = report.Send(ctx, client, endpoint)
		if err != nil {
			return nil, err
		}
	}

	if resp == nil {
		// This mimics scout.py
		resp = &Response{
			AppInfo: AppInfo{
				LatestVersion: r.Version,
			},
		}
	}

	if resp != nil && resp.DisableScout {
		r.mu.Lock()
		r.disabled = true
		r.mu.Unlock()
	}

	return resp, nil
}
