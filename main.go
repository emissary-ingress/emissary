package main

import (
	"fmt"
	"net/http"
	"os"
	"plugin"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

func usage() {
	fmt.Printf("Usage: %s TCP_ADDR PATH/TO/PLUGIN.so\n", os.Args[0])
	fmt.Printf("   or: %s <-h|--help>\n", os.Args[0])
	fmt.Printf("Run an Ambassador Pro middleware plugin locally, for plugin development\n")
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

	// this should match how amb-sidecar is compiled
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && runtime.Version() == "go1.11.4" {
		err := mainNative(os.Args[1], os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %v", os.Args[0], err)
			os.Exit(1)
		}
	} else {
		err := mainDocker(os.Args[1], os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %v", os.Args[0], err)
			os.Exit(1)
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

	return http.ListenAndServe(":8080", http.HandlerFunc(pluginMain))
}

func mainDocker(socketName, pluginFilepath string) error {
	panic("not implemented")
}
