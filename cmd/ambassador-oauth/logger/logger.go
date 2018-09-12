package logger

import (
	"log"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	jlog "github.com/joonix/log"
	"github.com/sirupsen/logrus"
)

var instance *logrus.Logger

// New ..
func New(c *config.Config) *logrus.Logger {
	// Whatever common logger configurations we need, should
	// be placed here.
	if instance == nil {
		instance = logrus.New()

		// Set Kubernetes log formatter.
		instance.Formatter = &jlog.FluentdFormatter{}

		// Set log level.
		if level, err := logrus.ParseLevel(c.Level); err == nil {
			instance.SetLevel(level)
		}

		log.SetOutput(instance.Writer())
	}

	return instance
}
