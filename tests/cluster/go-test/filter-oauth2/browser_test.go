// +build test

package oauth2_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type logWriter struct {
	t    *testing.T
	name string
	buf  []byte
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	now := time.Now()
	for {
		nl := bytes.IndexByte(w.buf, '\n')
		if nl < 0 {
			break
		}
		line := w.buf[:nl]
		w.buf = w.buf[nl+1:]
		w.t.Logf("%v [%s] %q", now, w.name, line)
	}
	return len(p), nil
}

func (w *logWriter) Close() error {
	if len(w.buf) > 0 {
		w.Write([]byte{'\n'})
	}
	return nil
}

var _ io.WriteCloser = &logWriter{}

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
	lw := &logWriter{t: t, name: "npm install"}
	cmd.Stdout = lw
	cmd.Stderr = lw
	err := cmd.Run()
	lw.Close()
	if err != nil {
		t.Fatal(err)
	}
	npmInstalled = true
}

// This function is closely coupled with run.js:browserTest().
func browserTest(t *testing.T, timeout time.Duration, expr string) {
	videoFileName := url.PathEscape(t.Name()) + ".webm"
	os.Remove(filepath.Join("testdata", videoFileName))

	shotdir, err := ioutil.TempDir("", "browserTest.")
	if err != nil {
		t.Fatal(time.Now(), "Bail out!", err)
	}
	defer os.RemoveAll(shotdir)

	// The main "node" invocation //////////////////////////////////////////
	cmd := exec.Command("node", "--eval", fmt.Sprintf(`
const run = require("./run.js");
const tests = require("./tests.js");

run.browserTest(%d, %q, async (browsertab) => {
	console.log("[inner] started");
	await %s;
	console.log("[inner] ran to completion");
});
`, timeout.Milliseconds(), shotdir, expr))
	cmd.Dir = "./testdata/"
	lw := &logWriter{t: t, name: "node"}
	cmd.Stdout = lw
	cmd.Stderr = lw
	t.Log(time.Now(), "starting...")
	nodeErr := cmd.Run()
	t.Log(time.Now(), "...finished")

	// Turn the timestamped screenshots in to a video //////////////////////
	//
	// https://stackoverflow.com/questions/25073292/how-do-i-render-a-video-from-a-list-of-time-stamped-images
	fileinfos, err := ioutil.ReadDir(shotdir)
	if err != nil {
		t.Fatal(time.Now(), "Bail out!", err)
	}
	var timestamps []int
	for _, fileinfo := range fileinfos {
		if !strings.HasSuffix(fileinfo.Name(), ".png") {
			continue
		}
		timestamp, err := strconv.ParseInt(strings.TrimSuffix(fileinfo.Name(), ".png"), 10, 0)
		if err != nil {
			continue
		}
		timestamps = append(timestamps, int(timestamp))
	}
	sort.Ints(timestamps)
	inputTxt, err := os.OpenFile(filepath.Join(shotdir, "input.txt"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(time.Now(), "Bail out!", err)
	}
	for i, timestamp := range timestamps {
		if i > 0 {
			duration := timestamp - timestamps[i-1]
			fmt.Fprintf(inputTxt, "duration %d.%03d\n", duration/1000, duration%1000)
		}
		fmt.Fprintf(inputTxt, "file '%d.png'\n", timestamp)
	}
	inputTxt.Close()
	cmd = exec.Command("ffmpeg",
		// input options
		"-f", "concat", // input format
		"-i", filepath.Join(shotdir, "input.txt"), // input file

		// output options
		videoFileName,
	)
	cmd.Dir = "./testdata/"
	lw = &logWriter{t: t, name: "ffmpeg"}
	cmd.Stdout = lw
	cmd.Stderr = lw
	if err := cmd.Run(); err != nil {
		t.Fatal(time.Now(), "Bail out!", err)
	}

	// Report the result ///////////////////////////////////////////////////
	if nodeErr == nil {
		t.Log("result: pass")
	} else if ee, ok := nodeErr.(*exec.ExitError); ok && ee.ProcessState.ExitCode() == 77 {
		t.Log("result: skip")
		t.Skip()
	} else {
		t.Error("result: fail:", nodeErr)
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
				lw := &logWriter{t: t, name: "node list"}
				cmd.Stderr = lw
				jsonBytes, err := cmd.Output()
				lw.Close()
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
