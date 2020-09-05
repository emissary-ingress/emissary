// Package busy implements a dispatcher for BusyBox-style multi-call binaries.
package busy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/datawire/ambassador/pkg/environment"
)

type Command = func(ctx context.Context, version string, args ...string) error

func Main(binName, humanName string, version string, cmds map[string]Command) {
	name := filepath.Base(os.Args[0])
	if name == binName && len(os.Args) > 1 {
		name = os.Args[1]
		os.Args = os.Args[1:]
	}

	if name != "entrypoint" { // XXX: This is a layer-breaking hack
		environment.EnvironmentSetupEntrypoint()
	}

	if cmdFn, cmdFnOK := cmds[name]; cmdFnOK {
		ctx := context.Background()
		if err := cmdFn(ctx, version, os.Args[1:]...); err != nil {
			dlog.Errorf(ctx, "shut down with error error: %v", err)
			os.Exit(1)
		}
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
