// +build test

package oauth2_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func ensureBrowserInstalled(t *testing.T) {
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
}

// This function is closely coupled with run.js:browserTest().
func browserTest(t *testing.T, timeout time.Duration, expr string) {
	ensureBrowserInstalled(t)

	videoFileName := url.PathEscape(t.Name()) + ".webm"

	os.Remove(filepath.Join("testdata", videoFileName))

	imageStreamR, imageStreamW, err := os.Pipe()
	if err != nil {
		t.Fatal(errors.Wrap(err, "pipe"))
	}

	ffmpegCmd := exec.Command("ffmpeg",
		// input
		"-f", "image2pipe",
		// The Puppeteer docs say that on macOS, creating a
		// frame can take as long as 1/6s (~0.16s / 6fps).  On
		// my Parabola laptop (with X11), I'm seeing ~0.11s
		// (~9fps).  So let's play it safe and ask for 5fps.
		"-r", "5", // FPS
		"-i", "-",
		// output
		videoFileName,
	)
	ffmpegCmd.Dir = "./testdata/"
	ffmpegCmd.Stdin = imageStreamR
	ffmpegCmd.Stdout = os.Stdout
	ffmpegCmd.Stderr = os.Stderr

	jsCmd := exec.Command("node", "--eval", fmt.Sprintf(`
const run = require("./run.js");
const tests = require("./tests.js");

run.browserTest(%d, async (browsertab) => {
	await %s;
});
`, timeout.Milliseconds(), expr))

	jsCmd.Dir = "./testdata/"
	jsCmd.Stdout = os.Stdout
	jsCmd.Stderr = os.Stderr
	jsCmd.ExtraFiles = []*os.File{imageStreamW}

	if err := ffmpegCmd.Start(); err != nil {
		imageStreamR.Close()
		imageStreamW.Close()
		t.Fatal(errors.Wrap(err, "ffmpeg"))
	}
	if err := jsCmd.Start(); err != nil {
		imageStreamR.Close()
		imageStreamW.Close()
		t.Fatal(errors.Wrap(err, "node"))
	}
	imageStreamR.Close()
	imageStreamW.Close()
	jsErr := jsCmd.Wait()
	ffmpegErr := ffmpegCmd.Wait()
	if jsErr != nil {
		if ee, ok := jsErr.(*exec.ExitError); ok && ee.ProcessState.ExitCode() == 77 {
			t.Skip()
		} else {
			t.Error(jsErr)
		}
	}
	if ffmpegErr != nil {
		t.Error(errors.Wrap(ffmpegErr, "ffmpeg"))
	}
}

func TestCanAuthorizeRequests(t *testing.T) {
	t.Parallel()

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
	browserTest(t, 20*time.Second, `tests.chainTest(browsertab, require("./idp_auth0.js"), "Auth0 (/httpbin)")`)
}

func TestCanBeTurnedOffForSpecificPaths(t *testing.T) {
	t.Parallel()
	browserTest(t, 20*time.Second, `tests.disableTest(browsertab, require("./idp_auth0.js"), "Auth0 (/httpbin)")`)
}
