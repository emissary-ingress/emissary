// Package busy implements a dispatcher for BusyBox-style multi-call binaries.
package busy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/datawire/ambassador/pkg/environment"
)

func Main(binName, humanName string, cmds map[string]func()) {
	name := filepath.Base(os.Args[0])
	if name == binName && len(os.Args) > 1 {
		name = os.Args[1]
		os.Args = os.Args[1:]
	}

	if name != "entrypoint" { // XXX: This is a layer-breaking hack
		environment.EnvironmentSetupEntrypoint()
	}

	if cmdFn, cmdFnOK := cmds[name]; cmdFnOK {
		cmdFn()
	} else {
		fmt.Printf("The %s main program is a multi-call binary that combines various\n", humanName)
		fmt.Println("support programs into one executable.")
		fmt.Println()
		fmt.Printf("Usage: %s <PROGRAM> [arguments]...\n", binName)
		fmt.Println("   or: <PROGRAM> [arguments]...")
		fmt.Println()
		cmdnames := make([]string, 0, len(cmds))
		for cmdname := range cmds {
			cmdnames = append(cmdnames, cmdname)
		}
		sort.Strings(cmdnames)
		fmt.Println("Available programs:", cmdnames)
		fmt.Println()
		fmt.Printf("Unknown program %q\n", name)
		// POSIX says the shell should set $?=127 for "command
		// not found", so non-shell programs that just run a
		// command for you (including busybox) tend to mimic
		// that and use exit code 127 to indicate "command not
		// found".
		os.Exit(127)
	}
}
