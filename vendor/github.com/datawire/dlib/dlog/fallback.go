package dlog

import (
	"os"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

// DefaultFieldSort is the field sorter that is used by the default fallback logger.  It mostly
// mimics the logrus defaults.  This function may go away if we change the default fallback logger.
func DefaultFieldSort(fieldNames []string) {
	sort.Slice(fieldNames, func(i, j int) bool {
		// The order that log fields are printed in; fields not explicitly in this list are
		// at position "0".  This is similar to the order of the default behavior of
		// logrus.TextFormatter, except that:
		//
		//  - it also includes special ordering for the dexec fields
		//  - it also includes special ordering for the dgroup fields
		//  - it puts the caller information after any unknown fields, rather than before
		orders := map[string]int{
			logrus.FieldKeyTime:        -9,
			logrus.FieldKeyLevel:       -8,
			"THREAD":                   -7, // dgroup
			"dexec.pid":                -6,
			"dexec.stream":             -5,
			"dexec.data":               -4,
			"dexec.err":                -3,
			logrus.FieldKeyMsg:         -2,
			logrus.FieldKeyLogrusError: -1,
			logrus.FieldKeyFunc:        1,
			logrus.FieldKeyFile:        2,
		}
		iOrd := orders[fieldNames[i]]
		jOrd := orders[fieldNames[j]]
		if iOrd != jOrd {
			return iOrd < jOrd
		}
		return fieldNames[i] < fieldNames[j]
	})
}

var (
	// This mimics logrus.New(), but with a .Formatter.SortingFunc that makes dexec look nicer.
	fallbackLogger Logger = WrapLogrus(&logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			SortingFunc: DefaultFieldSort,
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	})
	fallbackLoggerMu sync.RWMutex
)

func getFallbackLogger() Logger {
	fallbackLoggerMu.RLock()
	defer fallbackLoggerMu.RUnlock()
	return fallbackLogger
}

// SetFallbackLogger sets the Logger that is returned for a context
// that doesn't have a Logger associated with it.  A nil fallback
// Logger will cause dlog calls on a context without a Logger to
// panic, which would be good for detecting places where contexts are
// not passed around correctly.  However, the default fallback logger
// is Logrus and will behave reasonably; in order to make using dlog a
// safe "no brainer".
func SetFallbackLogger(l Logger) {
	fallbackLoggerMu.Lock()
	defer fallbackLoggerMu.Unlock()
	fallbackLogger = l
}
