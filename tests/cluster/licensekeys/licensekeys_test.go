// +build test

package licensekeys_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys"
)

var wd = func() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}()

type Result func(*exec.Cmd) error

var (
	ResultExitZero Result = func(cmd *exec.Cmd) error {
		return cmd.Run()
	}
	ResultRunsFineExitZero Result = func(cmd *exec.Cmd) error {
		outputBytes, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}
		if regexp.MustCompile("[Ll]icense.?[Kk]ey").Match(outputBytes) {
			return errors.Errorf("output metnioned a license key: %q", outputBytes)
		}
		return nil
	}
	ResultRunsFineForAtLeast20Sec Result = func(cmd *exec.Cmd) error {
		// start
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		cmd.Stderr = cmd.Stdout
		if err := cmd.Start(); err != nil {
			return err
		}
		// wait
		output := new(strings.Builder)
		events := make(chan string)
		go func() {
			_, err := io.Copy(output, stdout)
			events <- fmt.Sprintf("i/o complete: %v", err)
		}()
		go func() {
			time.Sleep(20 * time.Second)
			cmd.Process.Kill()
			events <- "timeout complete"
		}()
		go func() {
			cmd.Wait()
			events <- "process complete"
		}()
		start := time.Now()
		a := <-events
		aTime := time.Since(start)
		b := <-events
		bTime := time.Since(start)
		c := <-events
		cTime := time.Since(start)
		// inspect
		if a != "timeout complete" {
			return errors.Errorf("process died early; events: [ %v:%q | %v:%q | %v:%q ]; output: %q", aTime, a, bTime, b, cTime, c, output)
		}
		if regexp.MustCompile("[Ll]icense.?[Kk]ey").MatchString(output.String()) {
			return errors.Errorf("output mentioned a license key: %q", output)
		}
		return nil
	}
	ResultLicenseKeyExitNonZero Result = func(cmd *exec.Cmd) error {
		outputBytes, err := cmd.CombinedOutput()
		if err == nil {
			return errors.New("expected an ExitError, but got no error")
		}
		ee, ok := err.(*exec.ExitError)
		if !ok {
			return errors.Errorf("expected an ExitError, but got an error of type %T: %v", err, err)
		}
		// Don't accept exit codes < 0 (even though they're
		// "NonZero"), because they mean that it didn't really
		// exit.
		if ee.ProcessState.ExitCode() <= 0 {
			return errors.Errorf("expected exit code > 0, got %d: %v", ee.ProcessState.ExitCode(), ee)
		}
		if !regexp.MustCompile("[Ll]icense.?[Kk]ey").Match(outputBytes) {
			return errors.Errorf("output did not mention a license key: %q", outputBytes)
		}
		return nil
	}
)

type TestCase struct {
	Args             []string
	Env              map[string]string
	RequiredFeatures []licensekeys.Feature
	SuccessResult    Result
	FailureResult    Result
}

func (tc TestCase) Name() string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.Join(tc.Args, " "), wd, ""), "/", "_")
}

func (tc TestCase) Run(key string, discriminator Result) error {
	// amb-sidecar needs something to connect to
	mockedRedis, err := newRedisMock()
	if err != nil {
		return err
	}
	defer mockedRedis.Shutdown()

	// app-sidecar needs ./temp/ and ./data/ directories, and
	// amb-sidecar needs a tempdir.
	tmpdir, err := ioutil.TempDir("", "lickey-test.") // NB: Don't include anything matching "[Ll]icense.?[Kk]ey", in case it appears in the output.
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	if err := os.Mkdir(filepath.Join(tmpdir, "temp"), 0777); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(tmpdir, "data"), 0777); err != nil {
		return err
	}

	cmd := exec.Command(tc.Args[0], tc.Args[1:]...)
	cmd.Dir = tmpdir
	cmd.Env = append(os.Environ(),
		"AMBASSADOR_ID="+tmpdir, // as good an identifier as any
		"AMBASSADOR_LICENSE_KEY="+key,
		// amb-sidecar
		"REDIS_SOCKET_TYPE="+mockedRedis.Network(),
		"REDIS_URL="+mockedRedis.Address(),
		"RLS_RUNTIME_DIR="+filepath.Join(tmpdir, "amb", "config"),
	)
	for k, v := range tc.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return discriminator(cmd)
}

func TestLicenseCheck(t *testing.T) {
	t.Parallel()

	testCases := []TestCase{
		// "help" commands should exit 0, but are allowed to mention "--license-key"
		{
			Args:          []string{"apictl"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "--help"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "help"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "rls"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "rls", "Validate", "--help"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "traffic"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "traffic", "initialize", "--help"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "traffic", "inject", "--help"},
			SuccessResult: ResultExitZero,
		},
		{
			Args:          []string{"apictl", "traffic", "intercept", "--help"},
			SuccessResult: ResultExitZero,
		},
		// commands that don't require a license key
		{
			Args:          []string{"apictl", "version"},
			SuccessResult: ResultRunsFineExitZero,
		},

		{
			Args:          []string{"apictl", "--version"},
			SuccessResult: ResultRunsFineExitZero,
		},
		// commands that do require a license key
		{
			Args: []string{"apictl", "rls", "Validate", "--offline", filepath.Join(wd, "testdata/ratelimits.yaml")},
			RequiredFeatures: []licensekeys.Feature{
				licensekeys.FeatureRateLimit,
			},
			SuccessResult: ResultRunsFineExitZero,
			FailureResult: ResultLicenseKeyExitNonZero,
		},
		/* Actually does things in the cluster, so it doesn't
		   work great to run a bunch of these in parallel.
		   Plus, even then it's slow and takes *minutes*.
		{
			Args: []string{"apictl", "traffic", "initialize"},
			RequiredFeatures: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
			},
			SuccessResult: ResultRunsFineExitZero,
			FailureResult: ResultLicenseKeyExitNonZero,
		},*/
		{
			Args: []string{"apictl", "traffic", "inject", "--deployment=qotm", "--port=5000", filepath.Join(wd, "testdata/deployment.yaml")},
			RequiredFeatures: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
			},
			SuccessResult: ResultRunsFineExitZero,
			FailureResult: ResultLicenseKeyExitNonZero,
		},
		/* Actually does things in the cluster, so we'd have
		   to provision a bunch of stuff in the cluster first,
		   and it doesn't seem like it's worth the effort for
		   this test.
		{
			Args: []string{"apictl", "traffic", "intercept"},
			RequiredFeatures: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
			},
			SuccessResult: ResultRunsFineExitZero,
			FailureResult: ResultLicenseKeyExitNonZero,
		},*/
		// daemons
		{
			Args: []string{"amb-sidecar"},
			Env: map[string]string{
				"APRO_HTTP_PORT": "0",
			},
			RequiredFeatures: nil, // it can run, but with features disabled
			SuccessResult:    ResultRunsFineForAtLeast20Sec,
			FailureResult:    ResultLicenseKeyExitNonZero,
		},
		{
			Args: []string{"traffic-proxy"},
			Env: map[string]string{
				"APRO_HTTP_PORT": "0",
			},
			RequiredFeatures: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
			},
			SuccessResult: ResultRunsFineForAtLeast20Sec,
			FailureResult: ResultLicenseKeyExitNonZero,
		},
		{
			Args: []string{"app-sidecar"},
			Env: map[string]string{
				"APPNAME": "myapp",
			},
			RequiredFeatures: []licensekeys.Feature{
				licensekeys.FeatureTraffic,
			},
			SuccessResult: ResultRunsFineForAtLeast20Sec,
			FailureResult: ResultLicenseKeyExitNonZero,
		},
	}
	invalidKeys := map[string]string{
		"empty":         "",
		"malformed":     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"bad signature": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8bogon",
		"v0 expired":    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6MTU2NjIzMjU1MX0.ihYqQ9w_vIUtm_dl1FCH7oAMDFsSitr1yCiGhjYCTdc",
		"v1 expired":    "eyJhbGciOiJQUzUxMiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlX2tleV92ZXJzaW9uIjoidjEiLCJjdXN0b21lcl9pZCI6ImRldiIsImVuYWJsZWRfZmVhdHVyZXMiOlsiZmlsdGVyIiwicmF0ZWxpbWl0IiwidHJhZmZpYyIsImRldnBvcnRhbCJdLCJleHAiOjE1NjYyMjY3MzcsImlhdCI6MTU2NjIyNjczNywibmJmIjoxNTY2MjI2NzM3fQ.IZezOL2ocqXuWRsOu545wh62esJxRht85wqbpwD8weWSeu9-K7benJKEV5t1xpUqP2OGzBjXO4KNagb8kDu1NA8rqVr87VvcsSFDvM0emCg6vREZqcLcMy65olo-HaNtDi5TFq4eQvQw3UdbsqCixOhbCFReeG7XdqTuEzbCbKmx8dLutjQKzTrILYWzCF_sGXhue-OcQGo1NbZS5X1DLypu2vPqFSdnGb47dMY2N4MewKwUMsrs8SOeFVnNsU9jEMCea1BsMPJsOycZAuoYsVVOa17KeAGTpR2zzH0w5TbXeOTnsyk0WChj206AjS5rBpgf3byyhcgRgv7wzrQ5SQ",
	}
	validKeys := func() map[string]string {
		ret := map[string]string{
			"v0": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8",
		}
		for _, combo := range combinations(licensekeys.ListKnownFeatures()) {
			cmd := exec.Command("apictl-key", "create", "--id=dev", "--expiration=1", "--features="+strings.Join(combo, ","))
			keyBytes, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			ret[strings.Join(combo, ",")] = strings.TrimSpace(string(keyBytes))
		}
		return ret
	}()

	for _, testCase := range testCases {
		testCase := testCase // capture loop variable
		t.Run(testCase.Name(), func(t *testing.T) {
			t.Parallel()

			successKeys := make(map[string]string)
			failureKeys := make(map[string]string)
			if testCase.FailureResult == nil {
				for keyName, key := range invalidKeys {
					successKeys[keyName] = key
				}
				for keyName, key := range validKeys {
					successKeys[keyName] = key
				}
			} else {
				for keyName, key := range invalidKeys {
					failureKeys[keyName] = key
				}
				for keyName, key := range validKeys {
					hasFeatures, err := keyHasFeatures(key, testCase.RequiredFeatures)
					if err != nil {
						t.Fatalf("parse valid license key %q: %v: %q", keyName, err, key)
					}
					if hasFeatures {
						successKeys[keyName] = key
					} else {
						failureKeys[keyName] = key
					}
				}
			}
			if len(successKeys) == 0 {
				t.Fatal("there doesn't seem to be a license key that would make the program happy; something is probably wrong with the tests")
			}

			for keyName, key := range successKeys {
				key := key // capture loop variable
				t.Run(keyName, func(t *testing.T) {
					t.Parallel()

					if err := testCase.Run(key, testCase.SuccessResult); err != nil {
						t.Error(err)
					}
				})
			}
			for keyName, key := range failureKeys {
				key := key // capture loop variable
				t.Run(keyName, func(t *testing.T) {
					t.Parallel()

					if err := testCase.Run(key, testCase.FailureResult); err != nil {
						t.Error(err)
					}
				})
			}
		})
	}
}

func keyHasFeatures(key string, features []licensekeys.Feature) (bool, error) {
	claims, err := licensekeys.ParseKey(key)
	if err != nil {
		return false, err
	}
	for _, feature := range features {
		if err := claims.RequireFeature(feature); err != nil {
			return false, nil
		}
	}
	return true, nil
}

func combinations(bucket []string) [][]string {
	var ret [][]string
	switch len(bucket) {
	case 0:
		ret = [][]string{
			{}, // this is 1 combination, not 0
		}
	default:
		without := combinations(bucket[1:])
		ret = append(ret, without...)
		for _, partial := range without {
			combo := append([]string{bucket[0]}, partial...)
			ret = append(ret, combo)
		}
	}
	return ret
}

type redisMock struct {
	sock net.Listener
	wg   sync.WaitGroup
}

func newRedisMock() (*redisMock, error) {
	sock, err := net.Listen("tcp", ":0") // #nosec G102
	if err != nil {
		return nil, err
	}
	ret := &redisMock{
		sock: sock,
	}
	ret.wg.Add(1)
	go ret.accepter()
	return ret, nil
}

func (m *redisMock) Network() string { return m.sock.Addr().Network() }
func (m *redisMock) Address() string { return m.sock.Addr().String() }

func (m *redisMock) accepter() {
	defer m.wg.Done()
	for {
		conn, err := m.sock.Accept()
		if err != nil {
			return
		}
		m.wg.Add(1)
		go m.connectionHandler(conn)
	}
}

func (m *redisMock) connectionHandler(conn net.Conn) {
	defer m.wg.Done()
	time.Sleep(time.Second)
	conn.Close()
}

func (m *redisMock) Shutdown() {
	m.sock.Close()
	m.wg.Wait()
}
