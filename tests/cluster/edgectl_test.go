// Package main_test (or at least this particular one) is a go test-compatible
// end-to-end test script for the outbound and intercept features of Edge
// Control.
//
// It expects you to have set up the cluster in the fashion specified in the
// smoke test document.
//
// It uses the edgectl binary in your PATH.
package main_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func showArgs(args []string) {
	fmt.Printf("+ %s\n", strings.Join(args, " "))
}

func showOutput(out string, cmdErr error) {
	if len(out) > 0 {
		fmt.Print(out)

		if out[len(out)-1] != '\n' {
			fmt.Println(" [no newline]")
		}
	}

	if cmdErr != nil {
		fmt.Println("==>", cmdErr)
	}

	fmt.Println()
}

func run(args ...string) error {
	_, err := capture(args...)
	return err
}

func capture(args ...string) (string, error) {
	showArgs(args)
	cmd := exec.Command(args[0], args[1:]...) // #nosec G204
	outBytes, err := cmd.CombinedOutput()
	out := string(outBytes)
	showOutput(out, err)

	return out, err
}

func pollCommand(disposition func(string, error) (bool, error), args ...string) error {
	current_output := "this text is unlikely to match what the command emits"
	repeated := 0

	timeout := 120 * time.Second
	sleepTime := 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		cmd := exec.Command(args[0], args[1:]...) // #nosec G204
		outBytes, err := cmd.CombinedOutput()
		out := string(outBytes)

		if err != nil {
			return err
		}

		if out != current_output {
			if repeated > 0 {
				fmt.Printf("[%d more times...]\n\n", repeated)
			}

			current_output = out
			repeated = 0

			showArgs(args)
			showOutput(out, err)
		} else {
			repeated += 1
		}

		if success, err := disposition(out, err); err != nil {
			return err
		} else if success {
			return nil
		}

		select {
		case <-time.After(sleepTime):
			// Try again
			continue
		case <-ctx.Done():
			// Timed out
			return errors.New("timed out")
		}
	}
}

// pollCommandOutput calls the command repeatedly until its output contains the
// specified string.
func pollCommandOutput(desired string, args ...string) error {
	disposition := func(out string, err error) (bool, error) {
		if err != nil {
			return false, err
		}

		return strings.Contains(out, desired), nil
	}

	return pollCommand(disposition, args...)
}

// pollCommandNotOutput calls the command repeatedly until its output does not
// contain the specified string.
func pollCommandNotOutput(disdained string, args ...string) error {
	disposition := func(out string, err error) (bool, error) {
		if err != nil {
			return false, err
		}

		return !strings.Contains(out, disdained), nil
	}

	return pollCommand(disposition, args...)
}

// pollCommandSuccess calls the command repeatedly until it succeeds.
func pollCommandSuccess(args ...string) error {
	disposition := func(_ string, err error) (bool, error) {
		return err == nil, nil
	}

	return pollCommand(disposition, args...)
}

func subTest(t *testing.T, name string, sub func(*require.Assertions)) {
	formattedSub := func(t *testing.T) {
		defer func() {
			if t.Skipped() {
				fmt.Println()
			}
		}()
		fmt.Println()
		r := require.New(t)
		sub(r)
	}
	if !t.Run(name, formattedSub) {
		t.FailNow()
	}
}

func runServer() *http.Server {
	s := &http.Server{
		Addr: "127.0.0.1:9000",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "local")
		}),
	}
	go s.ListenAndServe()

	return s
}

func TestSmokeOutbound(t *testing.T) {
	r := require.New(t)

	var (
		out string
		err error
	)

	executable := "edgectl"

	namespace := fmt.Sprintf("edgectl-%d", os.Getpid())
	nsArg := fmt.Sprintf("--namespace=%s", namespace)

	subTest(t, "setup", func(r *require.Assertions) {
		r.NoError(run("sudo", "true"), "setup: acquire privileges")
		r.NoError(run("printenv", "KUBECONFIG"), "setup: ensure cluster is set")
		r.NoError(run("sudo", "rm", "-f", "/tmp/edgectl.log"), "setup: remove old log")
		r.NoError(
			run("kubectl", "get", "pod", "does-not-exist", "--ignore-not-found"),
			"setup: check cluster connectivity",
		)
		r.NoError(run("kubectl", "create", "namespace", namespace), "setup: create test namespace")
		r.NoError(
			run("kubectl", nsArg, "create", "deploy", "hello-world", "--image=ark3/hello-world"),
			"setup: create deployment",
		)
		r.NoError(
			run("kubectl", nsArg, "expose", "deploy", "hello-world", "--port=80", "--target-port=8000"),
			"setup: create service",
		)
		r.NoError(
			run("kubectl", nsArg, "get", "svc,deploy", "hello-world"),
			"setup: check svc/deploy",
		)
	})

	defer func() {
		r.NoError(
			run("kubectl", "delete", "namespace", namespace, "--wait=false"),
			"cleanup: delete test namespace",
		)
	}()

	subTest(t, "pre-daemon", func(r *require.Assertions) {
		r.Error(run(executable, "status"), "status with no daemon")
		r.Error(run(executable, "daemon"), "daemon without sudo")
	})

	subTest(t, "launch daemon", func(r *require.Assertions) {
		r.NoError(run("sudo", executable, "daemon"), "launch daemon")
		r.NoError(run(executable, "version"), "version with daemon")
		r.NoError(run(executable, "status"), "status with daemon")
	})

	defer func() { r.NoError(run(executable, "quit"), "cleanup: quit daemon") }()

	subTest(t, "await net overrides", func(r *require.Assertions) {
		r.NoError(pollCommandNotOutput("Network overrides NOT established", executable, "status"))
	})

	subTest(t, "connect", func(r *require.Assertions) {
		r.NoError(run(executable, "connect", "-n", namespace), "connect")
		out, err = capture(executable, "status")
		r.NoError(err, "status connected")
		r.Contains(out, "Context")
	})

	subTest(t, "await bridge", func(r *require.Assertions) {
		r.NoError(pollCommandOutput("Proxy:         ON", executable, "status"))
	})

	subTest(t, "await service", func(r *require.Assertions) {
		r.NoError(pollCommandSuccess(
			"kubectl", nsArg, "run", "curl-from-cluster", "--rm", "-it",
			"--image=pstauffer/curl", "--restart=Never", "--",
			"curl", "--silent", "--output", "/dev/null",
			"http://hello-world."+namespace,
		))
	})

	subTest(t, "check bridge", func(r *require.Assertions) {
		r.NoError(run("curl", "-sv", "hello-world."+namespace), "check bridge")
	})

	subTest(t, "intercept", func(r *require.Assertions) {
		out, err = capture(executable, "status")
		r.NoError(err, "status connected")
		r.NotContains(out, "Unavailable: no traffic manager")

		r.NoError(run("kubectl", "get", "svc,deploy", "echo"))
		r.NoError(run("kubectl", "get", "svc,deploy", "intercepted"))

		out, err = capture(executable, "intercept", "avail")
		r.NoError(err)
		r.Contains(out, "echo")
		r.Contains(out, "intercepted")

		s := runServer()
		defer s.Close()

		doCurl := func(val string) string {
			out, err := capture("curl", "-s", "--fail", "-H", "dev:"+val, "echo.default")
			r.NoError(err)
			return out
		}

		r.Contains(doCurl("moo"), "moo")
		r.Contains(doCurl("arf"), "arf")
		r.Contains(doCurl("baa"), "baa")

		r.NoError(run(executable, "intercept", "list"))

		r.NoError(run(executable, "intercept", "add", "echo", "-m=dev=moo", "-t=localhost:9000", "-n=moo-cept"))
		r.NoError(run(executable, "intercept", "list"))
		r.NoError(pollCommandOutput("Running", "kubectl", "get", "mapping", "moo-cept-mapping"))

		r.Contains(doCurl("moo"), "local")
		r.Contains(doCurl("arf"), "arf")
		r.Contains(doCurl("baa"), "baa")

		r.NoError(run(executable, "intercept", "add", "echo", "-m=dev=arf", "-t=localhost:9000", "-n=arf-cept"))
		r.NoError(run(executable, "intercept", "list"))
		r.NoError(pollCommandOutput("Running", "kubectl", "get", "mapping", "arf-cept-mapping"))

		r.Contains(doCurl("moo"), "local")
		r.Contains(doCurl("arf"), "local")
		r.Contains(doCurl("baa"), "baa")
	})

	subTest(t, "wind down", func(r *require.Assertions) {
		out, err = capture(executable, "status")
		r.NoError(err, "status connected")
		r.Contains(out, "Context")

		r.NoError(run(executable, "disconnect"), "disconnect")

		out, err = capture(executable, "status")
		r.NoError(err, "status disconnected")
		r.Contains(out, "Not connected")

		r.Error(run("curl", "-sv", "hello-world."+namespace), "check disconnected")
	})
}
