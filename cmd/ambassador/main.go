// Ambassador combines the various Golang binaries used in the Ambassador
// container, dispatching on os.Args[0] like BusyBox. Note that the
// capabilities_wrapper binary is _not_ included here. That one has special
// permissions magic applied to it that is not appropriate for these other
// binaries.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/datawire/ambassador/cmd/ambex"
	"github.com/datawire/ambassador/cmd/kubestatus"
	"github.com/datawire/ambassador/cmd/watt"
)

func main() {
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
	default:
		fmt.Println("The Ambassador main program is a multi-call binary that combines various")
		fmt.Println("support programs into one executable.")
		fmt.Println()
		fmt.Println("Usage: ambassador [program] [arguments]...")
		fmt.Println("   or: [program] [arguments]...")
		fmt.Println()
		fmt.Println("Available programs: ambex kubestatus watt")
		fmt.Println()
		fmt.Printf("Unknown name %q\n", name)
	}
}
