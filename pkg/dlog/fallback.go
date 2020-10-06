package dlog

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	fallbackLogger   Logger = WrapLogrus(logrus.New())
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
