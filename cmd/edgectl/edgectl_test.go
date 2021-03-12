package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/datawire/ambassador/pkg/dtest"
	"github.com/datawire/ambassador/pkg/dtest/testprocess"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

var kubeconfig string

func TestMain(m *testing.M) {
	testprocess.Dispatch()
	kubeconfig = dtest.Kubeconfig()
	os.Setenv("DTEST_KUBECONFIG", kubeconfig)
	dtest.WithMachineLock(func() {
		os.Exit(m.Run())
	})
}

func newCmd(t testing.TB, args ...string) *dexec.Cmd {
	ctx := dlog.NewTestContext(t, false)
	cmd := dexec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	return cmd
}

func run(t testing.TB, args ...string) error {
	return newCmd(t, args...).Run()
}

func capture(t testing.TB, args ...string) (string, error) {
	output, err := newCmd(t, args...).CombinedOutput()
	return string(output), err
}

// doBuildExecutable calls make in a subprocess running as the user
func doBuildExecutable(t testing.TB) error {
	if !strings.Contains(os.Getenv("MAKEFLAGS"), "--jobserver-auth") {
		alreadySudoed := os.Getuid() == 0 && os.Getenv("SUDO_USER") != ""
		args := []string{"make", "-C", "../..", "bin_" + runtime.GOOS + "_" + runtime.GOARCH + "/edgectl"}
		if alreadySudoed {
			// un-sudo
			args = append([]string{"sudo", "-E", "-u", os.Getenv("SUDO_USER"), "--"}, args...)
		}
		return run(t, args...)
	}
	return nil
}

var executable = "../../bin_" + runtime.GOOS + "_" + runtime.GOARCH + "/edgectl"

func TestSmokeOutbound(t *testing.T) {
	var out string
	var err error

	namespace := fmt.Sprintf("edgectl-%d", os.Getpid())
	nsArg := fmt.Sprintf("--namespace=%s", namespace)

	t.Log("setup")
	require.NoError(t, run(t, "sudo", "true"), "setup: acquire privileges")
	require.NoError(t, run(t, "printenv", "KUBECONFIG"), "setup: ensure cluster is set")
	require.NoError(t, run(t, "sudo", "rm", "-f", "/tmp/edgectl.log"), "setup: remove old log")
	require.NoError(t,
		run(t, "kubectl", "delete", "pod", "teleproxy", "--ignore-not-found", "--wait=true"),
		"setup: check cluster connectivity",
	)
	require.NoError(t, doBuildExecutable(t), "setup: build executable")
	require.NoError(t, run(t, "kubectl", "create", "namespace", namespace), "setup: create test namespace")
	require.NoError(t,
		run(t, "kubectl", nsArg, "create", "deploy", "hello-world", "--image=ark3/hello-world"),
		"setup: create deployment",
	)
	require.NoError(t,
		run(t, "kubectl", nsArg, "expose", "deploy", "hello-world", "--port=80", "--target-port=8000"),
		"setup: create service",
	)
	require.NoError(t,
		run(t, "kubectl", nsArg, "get", "svc,deploy", "hello-world"),
		"setup: check svc/deploy",
	)
	defer func() {
		require.NoError(t,
			run(t, "kubectl", "delete", "namespace", namespace, "--wait=false"),
			"cleanup: delete test namespace",
		)
	}()

	t.Log("pre-daemon")
	require.Error(t, run(t, executable, "status"), "status with no daemon")
	require.Error(t, run(t, executable, "daemon"), "daemon without sudo")

	t.Log("launch daemon")
	if !assert.NoError(t, run(t, "sudo", executable, "daemon"), "launch daemon") {
		logBytes, _ := ioutil.ReadFile("/tmp/edgectl.log")
		for _, line := range strings.Split(string(logBytes), "\n") {
			t.Logf("/tmp/edgectl.log: %q", line)
		}
		t.FailNow()
	}
	require.NoError(t, run(t, executable, "version"), "version with daemon")
	require.NoError(t, run(t, executable, "status"), "status with daemon")
	defer func() { require.NoError(t, run(t, executable, "quit"), "quit daemon") }()

	t.Log("await net overrides")
	func() {
		for i := 0; i < 120; i++ {
			out, _ := capture(t, executable, "status")
			if !strings.Contains(out, "Network overrides NOT established") {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("timed out waiting for net overrides")
	}()

	t.Log("connect")
	require.Error(t, run(t, executable, "connect"), "connect without --legacy")
	require.NoError(t, run(t, executable, "connect", "--legacy", "-n", namespace), "connect with --legacy")
	out, err = capture(t, executable, "status")
	require.NoError(t, err, "status connected")
	if !strings.Contains(out, "Context") {
		t.Fatal("Expected Context in connected status output")
	}
	defer func() {
		require.NoError(t,
			run(t, "kubectl", "delete", "pod", "teleproxy", "--ignore-not-found", "--wait=false"),
			"make next time quicker",
		)
	}()

	t.Log("await bridge")
	func() {
		for i := 0; i < 120; i++ {
			out, _ := capture(t, executable, "status")
			if strings.Contains(out, "Proxy:         ON") {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		_ = run(t, "kubectl", "get", "pod", "teleproxy")
		t.Fatal("timed out waiting for bridge")
	}()

	t.Log("await service")
	func() {
		for i := 0; i < 120; i++ {
			err := run(t,
				// `kubectl` and global args
				"kubectl", nsArg,
				// `kubectl run` and args
				"run", "--rm", "-it", "--image=pstauffer/curl", "--restart=Never", "curl-from-cluster", "--",
				// `curl` and args
				"curl", "--silent", "--output", "/dev/null", "http://hello-world."+namespace,
			)
			if err == nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("timed out waiting for hello-world service")
	}()

	t.Log("check bridge")
	require.NoError(t, run(t, "curl", "-sv", "hello-world."+namespace), "check bridge")

	t.Log("wind down")
	out, err = capture(t, executable, "status")
	require.NoError(t, err, "status connected")
	if !strings.Contains(out, "Context") {
		t.Fatal("Expected Context in connected status output")
	}
	require.NoError(t, run(t, executable, "disconnect"), "disconnect")
	out, err = capture(t, executable, "status")
	require.NoError(t, err, "status disconnected")
	if !strings.Contains(out, "Not connected") {
		t.Fatal("Expected Not connected in status output")
	}
	require.Error(t, run(t, "curl", "-sv", "hello-world."+namespace), "check disconnected")
}
