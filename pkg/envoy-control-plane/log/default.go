package log

// DefaultLogger is enabled when no consuming clients provide
// a logger to the server/cache subsystem
type DefaultLogger struct {
}

// NewDefaultLogger creates a DefaultLogger which is a no-op to maintain current functionality
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

// Debugf logs a message at level debug on the standard logger.
func (l *DefaultLogger) Debugf(format string, args ...interface{}) {
}

// Infof logs a message at level info on the standard logger.
func (l *DefaultLogger) Infof(format string, args ...interface{}) {
}

// Warnf logs a message at level warn on the standard logger.
func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
}

// Errorf logs a message at level error on the standard logger.
func (l *DefaultLogger) Errorf(format string, args ...interface{}) {
}
