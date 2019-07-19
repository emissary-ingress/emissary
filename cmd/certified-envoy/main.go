//go:generate make -C ../.. cmd/certified-envoy/envoy.go

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/datawire/apro/lib/licensekeys"
)

func withEnvoyBinary(fn func(filename string) error) error {
	envoyFile, err := ioutil.TempFile("", "envoy.")
	defer os.Remove(envoyFile.Name())
	if err != nil {
		return err
	}
	if err := envoyFile.Chmod(0200); err != nil { // write only
		envoyFile.Close()
		return err
	}
	if _, err := io.WriteString(envoyFile, envoyBytes); err != nil {
		envoyFile.Close()
		return err
	}
	if err = envoyFile.Chmod(0100); err != nil { // execute only
		envoyFile.Close()
		return err
	}
	envoyFile.Close()

	return fn(envoyFile.Name())
}

func execEnvoy(args ...string) {
	// OK, we don't really syscall.Exec() Envoy, because we need
	// to be able to remove the binary after it starts.  So wait
	// for the process, and exit() as if we did become Envoy.

	var cmd *exec.Cmd
	err := withEnvoyBinary(func(envoyFileName string) error {
		cmd = &exec.Cmd{
			Path:   envoyFileName,
			Args:   append([]string{os.Args[0]}, args...),
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		return cmd.Start()
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
		os.Exit(126) // POSIX says 126 = not executable
	}

	if waitErr := cmd.Wait(); waitErr != nil {
		status := waitErr.(*exec.ExitError).Sys().(syscall.WaitStatus)
		switch {
		case status.Exited():
			os.Exit(status.ExitStatus())
		case status.Signaled():
			// POSIX shells use 128+SIGNAL for the exit
			// code when the process is killed by a
			// signal.
			os.Exit(128 + int(status.Signal()))
		default:
			panic("should not happen")
		}
	}
	os.Exit(0)
}

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func cmdVersion() {
	envoyVersion := "envoy  version: unknown"
	_ = withEnvoyBinary(func(envoyFileName string) error {
		cmd := &exec.Cmd{
			Path: envoyFileName,
			Args: []string{"envoy", "--version"},
		}
		bs, err := cmd.Output()
		if err != nil {
			return err
		}
		envoyVersion = strings.TrimSpace(string(bs))
		return nil
	})
	fmt.Println(envoyVersion)
	fmt.Println("ambassador-core  version:", Version)
	os.Exit(0)
}

func cmdHelp() {
	fmt.Printf("Usage: AMBASSADOR_LICENSE_KEY=<...> %s <ARGS...>\n", os.Args[0])
	fmt.Printf("   or:                              %s -h|--help\n", os.Args[0])
	fmt.Printf("   or:                              %s --version\n", os.Args[0])
	fmt.Println("Datawire's certified build of Envoy Proxy.")
	fmt.Println("")
	fmt.Println("The AMBASSADOR_LICENSE_KEY environment variable must be set in order to run normally.")
	fmt.Println("You may request a license key from <https://www.getambassador.io/pro>.")
	fmt.Println("")
	fmt.Println("Following is the usage assuming that AMBASSADOR_LICENSE_KEY is set properly:")
	fmt.Println("")
	execEnvoy("--help")
}

func main() {
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--version":
			cmdVersion()
		case "-h", "--help":
			cmdHelp()
		}
	}

	claims, err := licensekeys.ParseKey(os.Getenv("AMBASSADOR_LICENSE_KEY"))

	go func() {
		if err := licensekeys.PhoneHome(claims, "certified-envoy", Version); err != nil {
			fmt.Fprintf(os.Stderr, "%s: metriton error: %v\n", os.Args[0], err)
		}
	}()

	if err != nil {
		if os.Getenv("AMBASSADOR_LICENSE_KEY") == "" {
			fmt.Fprintf(os.Stderr, "%s: AMBASSADOR_LICENSE_KEY is not set\n", os.Args[0])
		} else {
			fmt.Fprintf(os.Stderr, "%s: license key error: %v", os.Args[0], err)
		}
		os.Exit(1)
	}

	execEnvoy(os.Args[1:]...)
}
