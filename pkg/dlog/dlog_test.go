package dlog_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

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

type testLogEntry struct {
	level   dlog.LogLevel
	fields  map[string]interface{}
	message string
}

type testLog struct {
	entries []testLogEntry
}

type testLogger struct {
	log    *testLog
	fields map[string]interface{}
}

func (l testLogger) Helper() {}
func (l testLogger) WithField(key string, value interface{}) dlog.Logger {
	ret := testLogger{
		log:    l.log,
		fields: make(map[string]interface{}, len(l.fields)+1),
	}
	for k, v := range l.fields {
		ret.fields[k] = v
	}
	ret.fields[key] = value
	return ret
}
func (l testLogger) StdLogger(dlog.LogLevel) *log.Logger {
	panic("not implemented")
}
func (l testLogger) Log(lvl dlog.LogLevel, msg string) {
	entry := testLogEntry{
		level:   lvl,
		message: msg,
		fields:  make(map[string]interface{}, len(l.fields)),
	}
	for k, v := range l.fields {
		entry.fields[k] = v
	}
	l.log.entries = append(l.log.entries, entry)
}

func TestFormating(t *testing.T) {
	funcs := []func(context.Context, ...interface{}){
		func(ctx context.Context, args ...interface{}) { dlog.Log(ctx, dlog.LogLevelInfo, args...) },
		dlog.Error,
		dlog.Warn,
		dlog.Info,
		dlog.Debug,
		dlog.Trace,
		dlog.Print,
		dlog.Warning,
	}
	funcsf := []func(context.Context, string, ...interface{}){
		func(ctx context.Context, fmt string, args ...interface{}) {
			dlog.Logf(ctx, dlog.LogLevelInfo, fmt, args...)
		},
		dlog.Errorf,
		dlog.Warnf,
		dlog.Infof,
		dlog.Debugf,
		dlog.Tracef,
		dlog.Printf,
		dlog.Warningf,
	}
	funcsln := []func(context.Context, ...interface{}){
		func(ctx context.Context, args ...interface{}) { dlog.Logln(ctx, dlog.LogLevelInfo, args...) },
		dlog.Errorln,
		dlog.Warnln,
		dlog.Infoln,
		dlog.Debugln,
		dlog.Traceln,
		dlog.Println,
		dlog.Warningln,
	}

	var log testLog
	ctx := dlog.WithLogger(context.Background(), testLogger{log: &log})

	testcases := []struct {
		Funcs    interface{}
		Args     []interface{}
		Expected string
	}{
		// tc 1
		{Funcs: funcs, Args: []interface{}{ctx, "foo %s", "bar"}, Expected: "foo %sbar"},
		{Funcs: funcsf, Args: []interface{}{ctx, "foo %s", "bar"}, Expected: "foo bar"},
		{Funcs: funcsln, Args: []interface{}{ctx, "foo %s", "bar"}, Expected: "foo %s bar"},
		// tc 2
		{Funcs: funcs, Args: []interface{}{ctx, "foo\n"}, Expected: "foo\n"},
		{Funcs: funcsf, Args: []interface{}{ctx, "foo\n"}, Expected: "foo\n"},
		{Funcs: funcsln, Args: []interface{}{ctx, "foo\n"}, Expected: "foo\n"},
	}
	cnt := 0
	for i, tc := range testcases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			funcsValue := reflect.ValueOf(tc.Funcs)
			fnvalues := make([]reflect.Value, funcsValue.Len())
			for i := 0; i < funcsValue.Len(); i++ {
				fnvalues[i] = funcsValue.Index(i)
			}
			argvalues := make([]reflect.Value, 0, len(tc.Args))
			for _, v := range tc.Args {
				argvalues = append(argvalues, reflect.ValueOf(v))
			}
			for j, fn := range fnvalues {
				t.Run(fmt.Sprint(j), func(t *testing.T) {
					if !assert.Len(t, log.entries, cnt) {
						return
					}
					fn.Call(argvalues)
					if !assert.Len(t, log.entries, cnt+1) {
						return
					}
					assert.Equal(t, tc.Expected, log.entries[cnt].message)
					cnt++
				})
			}
		})
	}
}
