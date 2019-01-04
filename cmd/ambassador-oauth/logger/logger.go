package logger

import (
	"log"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	"github.com/sirupsen/logrus"
)

var instance *logrus.Logger

// New returns singleton instance of logger.
func New(c *config.Config) *logrus.Logger {
	// Whatever common logger configurations we need, should
	// be placed here.
	if instance == nil {
		instance = logrus.New()

		// Sets custom formatter.
		customFormatter := new(logrus.TextFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		instance.SetFormatter(customFormatter)

		customFormatter.FullTimestamp = true
		// Sets log level.
		if level, err := logrus.ParseLevel(c.LogLevel); err == nil {
			instance.SetLevel(level)
		}

		log.SetOutput(instance.Writer())
	}

	return instance
}
