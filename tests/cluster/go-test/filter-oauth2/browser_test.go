// +build test

package oauth2_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
)

var npmLock sync.Mutex
var npmInstalled bool = false

func ensureNPMInstalled(t *testing.T) {
	npmLock.Lock()
	defer npmLock.Unlock()
	if npmInstalled {
		return
	}
	cmd := exec.Command("npm", "install")
	cmd.Dir = "./testdata/"
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	npmInstalled = true
}

// This function is closely coupled with run.js:browserTest().
func browserTest(t *testing.T, timeout time.Duration, expr string) {
	// NB: Use log.Println instead of t.Log because timestamps
	log.Println("starting...")

	videoFileName := url.PathEscape(t.Name()) + ".webm"
	os.Remove(filepath.Join("testdata", videoFileName))

	imageStreamR, imageStreamW, err := os.Pipe()
	if err != nil {
		t.Fatal(errors.Wrap(err, "pipe"))
	}
	wgStarted := new(sync.WaitGroup)
	wgStarted.Add(2)
	wgFinished := new(sync.WaitGroup)
	wgFinished.Add(2)
	var ffmpegErr, nodeErr error
	go func() {
		// The Puppeteer docs say that on macOS, creating a
		// frame can take as long as 1/6s (~0.16s / 6fps).  On
		// my Parabola laptop (with X11), I'm seeing ~0.11s
		// (~9fps).  So let's play it safe and ask for 5fps.
		cmd := exec.Command("ffmpeg",
			// input options
			"-f", "image2pipe", // input format
			"-r", "5", // fps
			"-i", "-", // input file

			// output options
			videoFileName,
		)
		cmd.Dir = "./testdata/"
		cmd.Stdin = imageStreamR
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		ffmpegErr = cmd.Start()
		log.Println("...ffmpeg started")
		wgStarted.Done()
		if ffmpegErr != nil {
			ffmpegErr = cmd.Wait()
		}
		log.Println("...ffmpeg finished")
		wgFinished.Done()
	}()
	go func() {
		cmd := exec.Command("node", "--eval", fmt.Sprintf(`
const run = require("./run.js");
const tests = require("./tests.js");

run.browserTest(%d, async (browsertab) => {
	await %s;
});
`, timeout.Milliseconds(), expr))

		cmd.Dir = "./testdata/"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.ExtraFiles = []*os.File{imageStreamW}

		nodeErr = cmd.Start()
		log.Println("...node started")
		wgStarted.Done()
		if nodeErr == nil {
			nodeErr = cmd.Wait()
		}
		log.Println("...node finished")
		wgFinished.Done()
	}()
	wgStarted.Wait()
	imageStreamR.Close()
	imageStreamW.Close()
	log.Println("... started")

	wgFinished.Wait()
	log.Println("... finished")

	log.Println("ffmpegErr", ffmpegErr)
	log.Println("nodeErr", nodeErr)

	if nodeErr != nil {
		if ee, ok := nodeErr.(*exec.ExitError); ok && ee.ProcessState.ExitCode() == 77 {
			t.Skip()
		} else {
			t.Error("nodeErr", nodeErr)
		}
	}
	if ffmpegErr != nil {
		t.Error("ffmpegErr", ffmpegErr)
	}
}

func TestCanAuthorizeRequests(t *testing.T) {
	t.Parallel()
	ensureNPMInstalled(t)

	fileInfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		fileInfo := fileInfo // capture loop variable
		if strings.HasPrefix(fileInfo.Name(), "idp_") && strings.HasSuffix(fileInfo.Name(), ".js") {
			t.Run(fileInfo.Name(), func(t *testing.T) {
				t.Parallel()

				cmd := exec.Command("node", "--print", fmt.Sprintf("JSON.stringify(require(%q).testcases)", "./"+fileInfo.Name()))
				cmd.Dir = "./testdata/"
				cmd.Stderr = os.Stderr
				jsonBytes, err := cmd.Output()
				if err != nil {
					t.Fatal(err)
				}
				var jsonData map[string]interface{}
				if err = json.Unmarshal(jsonBytes, &jsonData); err != nil {
					t.Fatal(err)
				}

				for casename := range jsonData {
					casename := casename // capture loop variable
					t.Run(casename, func(t *testing.T) {
						t.Parallel()
						browserTest(t, 20*time.Second, fmt.Sprintf(`tests.standardTest(browsertab, require("./%s"), "%s")`, fileInfo.Name(), casename))
					})
				}
			})
		}
	}
}

func TestCanBeChainedWithOtherFilters(t *testing.T) {
	t.Parallel()
	ensureNPMInstalled(t)

	t.Run("run", func(t *testing.T) {
		t.Parallel()
		browserTest(t, 20*time.Second, `tests.chainTest(browsertab, require("./idp_auth0.js"), "Auth0 (/httpbin)")`)
	})
}

func TestCanBeTurnedOffForSpecificPaths(t *testing.T) {
	t.Parallel()
	ensureNPMInstalled(t)

	t.Run("run", func(t *testing.T) {
		t.Parallel()
		browserTest(t, 20*time.Second, `tests.disableTest(browsertab, require("./idp_auth0.js"), "Auth0 (/httpbin)")`)
	})
}
