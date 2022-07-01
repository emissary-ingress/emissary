package apiext

// This is a separate file from anything else so that people don't see
// logrus being imported and think that it's OK to use logrus instead
// of dlog.  We're just coopting logrus.ParseLevel and the
// logrus.Level type, and not using any of the actual logging
// functionality.  Use dlog!

import (
	//nolint:depguard // So we can turn off buffering if we're not debug logging
	"github.com/sirupsen/logrus"

	"github.com/emissary-ingress/emissary/v3/pkg/busy"
)

func LogLevelIsAtLeastDebug() bool {
	return busy.GetLogLevel() >= logrus.DebugLevel
}
