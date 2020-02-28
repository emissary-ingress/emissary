package aes_plugin_runner

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

var mainNative func(socketName, pluginFilepath string) error = nil

// Version is inserted at build using --ldflags -X
var Version = "(unknown version)"

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] TCP_ADDR PATH/TO/PLUGIN.so\n", os.Args[0])
	fmt.Printf("   or: %s <-h|--help>\n", os.Args[0])
	fmt.Printf("   or: %s --version\n", os.Args[0])
	fmt.Printf("Run an Ambassador Edge Stack Filter plugin as an Ambassador AuthService, for plugin development\n")
	fmt.Printf("\n")
	fmt.Printf("OPTIONS:\n")
	fmt.Printf("  --docker   Force the use of Docker, for increased realism\n")
	if mainNative == nil {
		fmt.Printf("             (no-op; this build of aes-plugin-runner always uses Docker)\n")
	}
	fmt.Printf("\n")
	fmt.Printf("Example:\n")
	fmt.Printf("    %s :8080 ./myplugin.so\n", os.Args[0])
}

func errusage(msg string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], msg)
	fmt.Fprintf(os.Stderr, "Try '%s --help' for more information.\n", os.Args[0])
	os.Exit(2)
}

func Main() {
	argparser := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	argparser.Usage = func() {}
	flagVersion := argparser.Bool("version", false, "")
	flagDocker := argparser.Bool("docker", mainNative == nil, "")

	if err := argparser.Parse(os.Args[1:]); err != nil {
		if err == pflag.ErrHelp {
			usage()
			return
		}
		errusage(err.Error())
	}
	if *flagVersion {
		t := template.New("top")
		template.Must(t.Parse(`{{.Name}} version {{.Version}} ({{.GoVersion}} {{.GOOS}}/{{.GOARCH}})
Copyright 2019 Datawire. All rights reserved.

Information about open source code used in this executable is found at
<https://s3.amazonaws.com/datawire-static-files/{{.Name}}/{{.Version}}/{{.GOOS}}/{{.GOARCH}}/{{.Name}}.opensource.tar.gz>.

Information about open source code used in the Docker image used by
'--docker' is found in the '/{{.Name}}.opensource.tar.gz' file in
the 'quay.io/datawire/aes:{{.Name}}-{{.Version}}'
Docker image.
`))
		t.Execute(os.Stdout, map[string]string{
			"Name":      "aes-plugin-runner",
			"Version":   Version,
			"GoVersion": runtime.Version(),
			"GOOS":      runtime.GOOS,
			"GOARCH":    runtime.GOARCH,
		})
		return
	}
	if argparser.NArg() != 2 {
		errusage(fmt.Sprintf("expected exactly 2 arguments, but got %d", argparser.NArg()))
	}
	if !strings.HasSuffix(argparser.Arg(1), ".so") {
		errusage(fmt.Sprintf("plugin file path does not end with '.so': %s", argparser.Arg(1)))
	}
	_, portName, err := net.SplitHostPort(argparser.Arg(0))
	if err != nil {
		errusage(fmt.Sprintf("invalid TCP address: %v", err))
	}
	_, err = net.LookupPort("tcp", portName)
	if err != nil {
		errusage(fmt.Sprintf("invalid TCP port: %q", portName))
	}

	fmt.Fprintf(os.Stderr, " > aes-plugin-runner %s (%s %s/%s)\n", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if !*flagDocker && mainNative != nil {
		fmt.Fprintf(os.Stderr, " > running natively\n")
		err := mainNative(argparser.Arg(0), argparser.Arg(1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %v\n", os.Args[0], err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, " > running in Docker\n")
		err := mainDocker(argparser.Arg(0), argparser.Arg(1))
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

	aes_image := os.Getenv("AES_IMAGE")
	if aes_image == "" {
		aes_image = "quay.io/datawire/aes:" + Version
	}

	pluginFileDir := filepath.Dir(pluginFilepath)
	cmd := exec.Command("docker", "run", "--rm", "-it",
		"--volume="+pluginFileDir+":"+pluginFileDir+":ro",
		"--publish="+net.JoinHostPort(host, strconv.Itoa(portNumber))+":"+strconv.Itoa(portNumber),
		"--entrypoint=/ambassador/aes-plugin-runner",
		aes_image,
		fmt.Sprintf(":%d", portNumber), pluginFilepath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	eargs := make([]string, len(cmd.Args))
	for i := range cmd.Args {
		eargs[i] = "'" + cmd.Args[i] + "'"
	}
	fmt.Fprintf(os.Stderr, " $ %s\n", strings.Join(eargs, " "))

	return cmd.Run()
}
