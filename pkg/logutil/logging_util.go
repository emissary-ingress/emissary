package logutil

import (
	//nolint: depguard
	"github.com/sirupsen/logrus"
)

func ParseLogLevel(str string) (logrus.Level, error) {
	return logrus.ParseLevel(str)
}

func LogrusToKLogLevel(level logrus.Level) int {
	// Well this is disgusting. Logrus and klog use levels going in opposite directions,
	// _and_ klog doesn't export any of the mapping from names to numbers. So we hardcode
	// based on klog 1.0.0: info == 0, warning, error, fatal == 3.

	klogLevel := 2 // ERROR

	if level == logrus.WarnLevel {
		klogLevel = 1
	} else if level >= logrus.InfoLevel { // info or debug
		klogLevel = 0
	}

	return klogLevel
}
