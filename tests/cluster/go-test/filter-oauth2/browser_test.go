// +build test

package oauth2_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mediocregopher/radix.v2/redis"

	"github.com/datawire/apro/lib/testutil"
)

const (
	// So we want to give the test enough time to do its stuff,
	// and be forgiving of external servers being slow.  But we
	// also want all of the TestCanAuthorizeRequests sub-tests to
	// finish within `go test`'s default 10m timeout.  The timeout
	// below is the timeout for the actual test portion; it's
	// another 10-14s for cleanup and ffmpeg to happen after that.
	// Currently there are 14 sub-tests, so the upper bound for
	// this is around (600s/14)-14s = 28s.  Let's make it
	// shorter--fail fast.
	usualTimeout = 15 * time.Second
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
	t.Log(time.Now(), "...finished:", nodeErr)

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
	ensureNPMInstalled(t)

	fileInfos, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		fileInfo := fileInfo // capture loop variable
		if strings.HasPrefix(fileInfo.Name(), "idp_") && strings.HasSuffix(fileInfo.Name(), ".js") {
			t.Run(fileInfo.Name(), func(t *testing.T) {
				if fileInfo.Name() == "idp_google.js" {
					// XFail (Flynn) Need to beat on Google a beat more in the Multidomain world.
					t.SkipNow()
				}

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
						browserTest(t, usualTimeout, fmt.Sprintf(`tests.standardTest(browsertab, require("./%s"), "%s")`, fileInfo.Name(), casename))
					})
				}
			})
		}
	}
}

func TestCanBeChainedWithOtherFilters(t *testing.T) {
	ensureNPMInstalled(t)

	t.Run("run", func(t *testing.T) {
		browserTest(t, usualTimeout, `tests.chainTest(browsertab, require("./idp_auth0.js"), "Auth0 (/oauth2-auth0-nojwt-and-plugin-and-whitelist)")`)
	})
}

func TestCanBeTurnedOffForSpecificPaths(t *testing.T) {
	ensureNPMInstalled(t)

	t.Run("run", func(t *testing.T) {
		browserTest(t, usualTimeout, `tests.disableTest(browsertab, require("./idp_auth0.js"), "Auth0 (/oauth2-auth0-nojwt-and-plugin-and-whitelist)")`)
	})
}

func TestCanUseComplexJWTValidation(t *testing.T) {
	ensureNPMInstalled(t)

	t.Run("run", func(t *testing.T) {
		assert := &testutil.Assert{T: t}

		// step 1: get the session ID
		sessionID, xsrfToken := func() (string, string) {
			dirname, err := ioutil.TempDir("", "TestCanUseComplexJWTValidation.")
			assert.NotError(err)
			defer os.RemoveAll(dirname)

			sessionFilename := filepath.Join(dirname, "session-id.txt")
			xsrfFilename := filepath.Join(dirname, "xsrf-token.txt")

			browserTest(t, usualTimeout, fmt.Sprintf(`tests.writeSessionID(browsertab, require("./idp_auth0.js"), "Auth0 (/oauth2-auth0-complexjwt)", %q, %q)`, sessionFilename, xsrfFilename))
			assert.Bool(!t.Failed())

			sessionID, err := ioutil.ReadFile(sessionFilename)
			assert.NotError(err)

			xsrfToken, err := ioutil.ReadFile(xsrfFilename)
			assert.NotError(err)

			return string(sessionID), string(xsrfToken)
		}()

		// step 2: connect to Redis so that we can directly manipulate the Access Token
		redisClient, err := redis.Dial("tcp", "ambassador-redis.ambassador.svc.cluster.local:6379")
		if err != nil {
			t.Fatal(err)
		}
		getSessionData := func() interface{} {
			sessionDataBytes, err := redisClient.Cmd("GET", "session:"+sessionID).Bytes()
			assert.NotError(err)
			var sessionData interface{}
			assert.NotError(json.Unmarshal(sessionDataBytes, &sessionData))
			return sessionData
		}
		setSessionData := func(sessionData interface{}) {
			sessionDataBytes, err := json.Marshal(sessionData)
			assert.NotError(err)
			assert.NotError(redisClient.Cmd("SET", "session:"+sessionID, sessionDataBytes).Err)
		}

		// step 3: set up an HTTP client so we don't have to keep going through the headless browser
		curlJSON := func(urlStr string) (int, interface{}) {
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
				Transport: &http.Transport{
					// #nosec G402
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}

			req := &http.Request{
				Method: http.MethodGet,
				URL:    urlMust(url.Parse(urlStr)),
				Header: make(http.Header),
			}
			req.AddCookie(&http.Cookie{Name: "ambassador_session.oauth2-auth0-complexjwt.default", Value: sessionID})
			req.AddCookie(&http.Cookie{Name: "ambassador_xsrf.oauth2-auth0-complexjwt.default", Value: xsrfToken})

			resp, err := client.Do(req)
			assert.NotError(err)
			defer resp.Body.Close()

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			assert.NotError(err)
			var bodyData interface{}
			assert.NotError(json.Unmarshal(bodyBytes, &bodyData))
			t.Logf("=> HTTP %v : %v", resp.StatusCode, bodyData)
			return resp.StatusCode, bodyData
		}

		// step 4: validate that the session+access token work
		sessionData := getSessionData()
		func() {
			accessToken := sessionData.(map[string]interface{})["CurrentAccessToken"].(map[string]interface{})["AccessToken"].(string)
			respCode, respBody := curlJSON("https://ambassador.ambassador.svc.cluster.local/oauth2-auth0-complexjwt/headers")
			assert.IntEQ(http.StatusOK, respCode)
			authorization := respBody.(map[string]interface{})["headers"].(map[string]interface{})["Authorization"].(string)
			assert.StrEQ("Bearer "+accessToken, authorization)
			test := respBody.(map[string]interface{})["headers"].(map[string]interface{})["X-Test-Header"].(string)
			assert.StrEQ("yeppers", test)
		}()

		// step 5: validate that we can spoof an valid access token
		func() {
			accessToken, err := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
				"iss": "https://ambassador-oauth-e2e.auth0.com/",
				"sub": "auth0|5bbd4a9c5e09334d778a8b89",
				"aud": []string{
					"urn:datawire:ambassador:testapi",
					"https://ambassador-oauth-e2e.auth0.com/userinfo",
				},
				"iat":   1575420683,
				"exp":   1675507083,
				"azp":   "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
				"scope": "openid",
			}).SignedString(jwt.UnsafeAllowNoneSignatureType)
			assert.NotError(err)
			sessionData.(map[string]interface{})["CurrentAccessToken"].(map[string]interface{})["AccessToken"] = accessToken
			setSessionData(sessionData)
			respCode, respBody := curlJSON("https://ambassador.ambassador.svc.cluster.local/oauth2-auth0-complexjwt/headers")
			assert.IntEQ(http.StatusOK, respCode)
			authorization := respBody.(map[string]interface{})["headers"].(map[string]interface{})["Authorization"].(string)
			assert.StrEQ("Bearer "+accessToken, authorization)
			test := respBody.(map[string]interface{})["headers"].(map[string]interface{})["X-Test-Header"].(string)
			assert.StrEQ("yeppers", test)
		}()

		// step 6: validate that the JWT Filter is being strict about aud; spoof an invalid access token
		func() {
			accessToken, err := jwt.NewWithClaims(jwt.GetSigningMethod("none"), jwt.MapClaims{
				"iss": "https://ambassador-oauth-e2e.auth0.com/",
				"sub": "auth0|5bbd4a9c5e09334d778a8b89",
				"aud": []string{
					"urn:datawire:ambassador:testapi-bogus",
					"https://ambassador-oauth-e2e.auth0.com/userinfo",
				},
				"iat":   1575420683,
				"exp":   1675507083,
				"azp":   "DOzF9q7U2OrvB7QniW9ikczS1onJgyiC",
				"scope": "openid",
			}).SignedString(jwt.UnsafeAllowNoneSignatureType)
			assert.NotError(err)
			sessionData.(map[string]interface{})["CurrentAccessToken"].(map[string]interface{})["AccessToken"] = accessToken
			setSessionData(sessionData)
			respCode, _ := curlJSON("https://ambassador.ambassador.svc.cluster.local/oauth2-auth0-complexjwt/headers")
			assert.IntEQ(http.StatusForbidden, respCode)
		}()
	})
}

func TestWorksWithMSOffice(t *testing.T) {
	ensureNPMInstalled(t)

	// https://github.com/datawire/apro/issues/999
	t.Run("run", func(t *testing.T) {
		assert := &testutil.Assert{T: t}

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				// #nosec G402
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		resp, err := client.Get("https://ambassador.ambassador.svc.cluster.local/azure/httpbin/headers")
		assert.NotError(err)
		assert.HTTPResponseStatusEQ(resp, http.StatusSeeOther)
		u, err := resp.Location()
		assert.NotError(err)

		browserTest(t, usualTimeout, fmt.Sprintf(`tests.msofficeTest(browsertab, require("./idp_azure.js"), "Azure AD", "%s")`, u))
	})
}
