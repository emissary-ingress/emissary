package logger

import (
	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/sirupsen/logrus"
)

// NewLogger ...
func NewLogger(c *config.Config) *logrus.Logger {
	// LOGGER
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000",
		FullTimestamp:   true,
	}

	if c.Quiet {
		logger.Level = logrus.ErrorLevel
	} else {
		logger.Level = logrus.InfoLevel
	}

	return logger
}
