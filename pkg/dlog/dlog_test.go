package dlog_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/datawire/ambassador/pkg/dlog"
)

var logPos struct {
	File string
	Line int
}

// doLog() logs "grep for this" and sets logPos to exactly where it
// logged from.
func doLog(ctx context.Context) {
	_, file, line, _ := runtime.Caller(0)
	logPos.File, logPos.Line = file, line+2
	dlog.Infof(ctx, "grep for this")
}

var testLoggers = map[string]func(*testing.T) context.Context{
	"logrus": func(_ *testing.T) context.Context {
		logger := logrus.New()
		logger.SetReportCaller(true)
		return dlog.WithLogger(context.Background(), dlog.WrapLogrus(logger))
	},
	"testing": func(t *testing.T) context.Context {
		return dlog.WithLogger(context.Background(), dlog.WrapTB(t, false))
	},
}

func TestCaller(t *testing.T) {
	t.Parallel()

	doLog(dlog.WithLogger(context.Background(), dlog.WrapTB(t, false))) // initialize logPos
	expectedPos := fmt.Sprintf("%s:%d", filepath.Base(logPos.File), logPos.Line)
	t.Logf("expected pos = %q", expectedPos)

	for testname := range testLoggers {
		testname := testname
		t.Run(testname, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.v", "-test.run=TestHelperProcess", "--", testname)
			cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Error(err)
			}
			var logline string
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "grep for this") {
					logline = line
					break
				}
			}
			if logline == "" {
				t.Fatal("did not get any log output")
			}
			t.Logf("logline=%q", logline)
			if !strings.Contains(logline, expectedPos) {
				t.Errorf("it does not appear that the log reported itself as coming from %q",
					expectedPos)
			}
		})
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "expected exactly 1 argument, got %d\n", len(args))
		os.Exit(2)
	}

	ctx := testLoggers[args[0]](t)
	doLog(ctx)
}
