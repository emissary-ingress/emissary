// Ambassador combines the various Golang binaries used in the Ambassador
// container, dispatching on os.Args[0] like BusyBox.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	metriton "github.com/datawire/apro/lib/metriton"

	aes_plugin_runner "github.com/datawire/apro/cmd/aes-plugin-runner-native"
	amb_sidecar "github.com/datawire/apro/cmd/amb-sidecar"
	app_sidecar "github.com/datawire/apro/cmd/app-sidecar"
	traffic_manager "github.com/datawire/apro/cmd/traffic-proxy"

	"github.com/datawire/ambassador/cmd/ambex"
	"github.com/datawire/ambassador/cmd/kubestatus"
	"github.com/datawire/ambassador/cmd/watt"
)

var Version = "(unknown version)"

func main() {
	metriton.Reporter.Version = Version

	ambex.Version = Version
	// kubestatus.Version = Version // Does not exist
	watt.Version = Version

	aes_plugin_runner.Version = Version
	app_sidecar.Version = Version
	amb_sidecar.Version = Version

	name := filepath.Base(os.Args[0])
	if name == "ambassador" && len(os.Args) > 1 {
		name = os.Args[1]
		os.Args = os.Args[1:]
	}

	switch name {
	case "ambex":
		ambex.Main()
	case "watt":
		watt.Main()
	case "kubestatus":
		kubestatus.Main()
	case "aes-plugin-runner":
		aes_plugin_runner.Main()
	case "amb-sidecar":
		amb_sidecar.Main()
	case "app-sidecar":
		app_sidecar.Main()
	case "traffic-manager":
		traffic_manager.Main()
	default:
		fmt.Println("The Ambassador main program is a multi-call binary that combines various")
		fmt.Println("support programs into one executable.")
		fmt.Println()
		fmt.Println("Usage: ambassador <PROGRAM> [arguments]...")
		fmt.Println("   or: <PROGRAM> [arguments]...")
		fmt.Println()
		fmt.Println("Available programs: ambex kubestatus watt")
		fmt.Println("                    aes-plugin-runner amb-sidecar app-sidecar traffic-manager")
		fmt.Println()
		fmt.Printf("Unknown name %q\n", name)
	}
}
