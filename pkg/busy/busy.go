// Package busy implements a dispatcher for BusyBox-style multi-call binaries.
package busy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/datawire/ambassador/pkg/environment"
	"github.com/datawire/dlib/dlog"
)

type Command = func(ctx context.Context, version string, args ...string) error

var logrusLogger *logrus.Logger

func init() {
	logrusLogger = logrus.New()
	logrusFormatter := &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
	logrusLogger.SetFormatter(logrusFormatter)
	logrusLogger.SetReportCaller(true)
}

func SetLogLevel(lvl logrus.Level) {
	logrusLogger.SetLevel(lvl)
}

func GetLogLevel() logrus.Level {
	return logrusLogger.GetLevel()
}

func Main(binName, humanName string, version string, cmds map[string]Command) {
	name := filepath.Base(os.Args[0])
	if name == binName && len(os.Args) > 1 {
		name = os.Args[1]
		os.Args = os.Args[1:]
	}

	if name != "entrypoint" { // XXX: This is a layer-breaking hack
		environment.EnvironmentSetupEntrypoint()
	}

	logger := dlog.WrapLogrus(logrusLogger).
		WithField("PID", os.Getpid()).
		WithField("CMD", name)
	ctx := dlog.WithLogger(context.Background(), logger)
	dlog.SetFallbackLogger(logger.WithField("oops-i-did-not-pass-context-correctly", true))

	if cmdFn, cmdFnOK := cmds[name]; cmdFnOK {
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
