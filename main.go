package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

// What compiler version amb-sidecar was compiled with
const AProGoVersion = "1.11.4"

func usage() {
	fmt.Printf("Usage: %s TCP_ADDR PATH/TO/PLUGIN.so\n", os.Args[0])
	fmt.Printf("   or: %s <-h|--help>\n", os.Args[0])
	fmt.Printf("Run an Ambassador Pro middleware plugin as an Ambassador AuthService, for plugin development\n")
	fmt.Printf("\n")
	fmt.Printf("Example:\n")
	fmt.Printf("    %s :8080 ./myplugin.so\n", os.Args[0])
}

func errusage(msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], msg)
	fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", os.Args[0])
	os.Exit(2)
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			usage()
			os.Exit(0)
		}
	}
	if len(os.Args) != 3 {
		errusage(fmt.Sprintf("expected exactly 2 arguments, but got %d", len(os.Args)-1))
	}
	if !strings.HasSuffix(os.Args[2], ".so") {
		errusage(fmt.Sprintf("plugin file path does not end with '.so': %s", os.Args[2]))
	}
	_, portName, err := net.SplitHostPort(os.Args[1])
	if err != nil {
		errusage(fmt.Sprintf("invalid TCP address: %v", err))
	}
	_, err = net.LookupPort("tcp", portName)
	if err != nil {
		errusage(fmt.Sprintf("invalid TCP port: %q", portName))
	}

	fmt.Fprintf(os.Stderr, " > apro-plugin-runner %s/%s/%s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	fmt.Fprintf(os.Stderr, " > apro amb-sidecar   %s/%s/%s\n", "linux", "amd64", "go"+AProGoVersion)

	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && runtime.Version() == "go"+AProGoVersion {
		fmt.Fprintf(os.Stderr, " > GOOS/GOARCH/GOVERSION match, running natively\n")
		err := mainNative(os.Args[1], os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, " > GOOS/GOARCH/GOVERSION do not match, running in Docker\n")
		err := mainDocker(os.Args[1], os.Args[2])
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				ws := ee.ProcessState.Sys().(syscall.WaitStatus)
				switch {
				case ws.Exited():
					os.Exit(ws.ExitStatus())
				case ws.Signaled():
					os.Exit(128 + int(ws.Signal()))
				}
			} else {
				fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
				os.Exit(255)
			}
		}
	}
}

func mainNative(socketName, pluginFilepath string) error {
	pluginHandle, err := plugin.Open(pluginFilepath)
	if err != nil {
		return errors.Wrap(err, "load plugin file")
	}

	pluginInterface, err := pluginHandle.Lookup("PluginMain")
	if err != nil {
		return errors.Wrap(err, "invalid plugin file")
	}

	pluginMain, ok := pluginInterface.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return errors.New("invalid plugin file: PluginMain has the wrong type signature")
	}

	return http.ListenAndServe(socketName, http.HandlerFunc(pluginMain))
}

func mainDocker(socketName, pluginFilepath string) error {
	host, portName, _ := net.SplitHostPort(socketName)
	if host != "" {
		return errors.New("unfortunately, it is not valid to specify a host part of the TCP address when running in Docker, you may only specify a ':PORT'")
	}
	portNumber, _ := net.LookupPort("tcp", portName)

	pluginFilepath, err := filepath.Abs(pluginFilepath)
	if err != nil {
		return errors.Wrap(err, "unable to find absolute path of plugin file path")
	}

	pluginFileDir := filepath.Dir(pluginFilepath)
	cmd := exec.Command("docker", "run", "--rm", "-it",
		"--volume="+pluginFileDir+":"+pluginFileDir+":ro",
		"--publish="+net.JoinHostPort(host, strconv.Itoa(portNumber))+":"+strconv.Itoa(portNumber),
		"docker.io/library/golang:"+AProGoVersion,
		"/bin/sh", "-c", "cd /tmp && go mod init example.com/bogus && GO111MODULE=on go get github.com/datawire/apro-plugin-runner && apro-plugin-runner $@", "--", fmt.Sprintf(":%d", portNumber), pluginFilepath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	eargs := make([]string, len(cmd.Args))
	for i := range cmd.Args {
		eargs[i] = "'"+cmd.Args[i]+"'"
	}
	fmt.Fprintf(os.Stderr, " $ %s\n", strings.Join(eargs, " "))

	return cmd.Run()
}
