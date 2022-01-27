// We have this separate cheat.go file so that people working with the other files don't mistakenly
// believe that it's OK to use logrus directly.

package ambex

import (
	//nolint:depguard // We need to be able to pass these as arguments to pkg/busy.
	"github.com/sirupsen/logrus"
)

const (
	logrusDebugLevel = logrus.DebugLevel
	logrusInfoLevel  = logrus.InfoLevel
)
