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
		// This matches the default behavior of logrus.TextFormatter, except that it also
		// includes "dexec.XXX" (in addition to the usual logrus.FieldXXX fields) in the
		// fixed-ordering.
		orders := map[string]int{
			logrus.FieldKeyTime:        -10,
			logrus.FieldKeyLevel:       -9,
			"dexec.pid":                -8,
			"dexec.stream":             -7,
			"dexec.data":               -6,
			"dexec.err":                -5,
			logrus.FieldKeyMsg:         -4,
			logrus.FieldKeyLogrusError: -3,
			logrus.FieldKeyFunc:        -2,
			logrus.FieldKeyFile:        -1,
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
