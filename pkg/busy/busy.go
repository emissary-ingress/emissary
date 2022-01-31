// Package busy implements a dispatcher for BusyBox-style multi-call binaries.
//
// BUG(lukeshu): Global state is bad, but this package has global state in the form of the global
// log level.
package busy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	//nolint:depguard // This is one of the few places where it is approrpiate to not go through
	// to initialize dlog.
	"github.com/sirupsen/logrus"

	"github.com/datawire/dlib/dlog"
)

type Command struct {
	Setup func(ctx context.Context)
	Run   func(ctx context.Context, version string, args ...string) error
}

// logrusLogger is a global (rather than simply being a variable within the Main() function) for the
// sole purpose of allowing the global program-wide log level to be fussed with at runtime.
//
// If you find yourself adding any other access to this blob of global state:
//
//     Stop.  You don't want more global state.  I (LukeShu) promise you there's a better way to do
//     whatever you're attempting, and that adding more global state is not what you really want.
var logrusLogger *logrus.Logger

func init() {
	testInit()
}

// testInit is separate from init() so that it can be explicitly called from the tests.
func testInit() {
	logrusLogger = logrus.New()
	if useJSON, _ := strconv.ParseBool(os.Getenv("AMBASSADOR_JSON_LOGGING")); useJSON {
		logrusLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05.0000",
		})
	} else {
		logrusLogger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05.0000",
			FullTimestamp:   true,
		})
	}
	logrusLogger.SetReportCaller(true)
}

// SetLogLevel sets the global program-wide log level.
//
// BUG(lukeshu): SetLogLevel mutates global state, and global state is bad.
func SetLogLevel(lvl logrus.Level) {
	logrusLogger.SetLevel(lvl)
}

// GetLogLevel gets the global program-wide log level.
//
// BUG(lukeshu): GetLogLevel accesses global state, and global state is bad.
func GetLogLevel() logrus.Level {
	return logrusLogger.GetLevel()
}

// Main should be called from your actual main() function.
func Main(binName, humanName string, version string, cmds map[string]Command) {
	name := filepath.Base(os.Args[0])
	if name == binName && len(os.Args) > 1 {
		name = os.Args[1]
		os.Args = os.Args[1:]
	}

	logger := dlog.WrapLogrus(logrusLogger).
		WithField("PID", os.Getpid()).
		WithField("CMD", name)
	ctx := dlog.WithLogger(context.Background(), logger) // early in Main()
	dlog.SetFallbackLogger(logger.WithField("oops-i-did-not-pass-context-correctly", "THIS IS A BUG"))

	if cmd, cmdOk := cmds[name]; cmdOk {
		cmd.Setup(ctx)
		if err := cmd.Run(ctx, version, os.Args[1:]...); err != nil {
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
