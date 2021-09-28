package busy

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoggingTextFormatterDefault(t *testing.T) {
	os.Unsetenv("AMBASSADOR_JSON_LOGGING")
	testInit()

	fm, isTextFormatter := logrusLogger.Formatter.(*logrus.TextFormatter)
	if !assert.True(t, isTextFormatter) {
		return
	}
	assert.Equal(t, "2006-01-02 15:04:05.0000", fm.TimestampFormat)
	assert.True(t, fm.FullTimestamp)
}

func TestLoggingJsonFormatter(t *testing.T) {
	os.Setenv("AMBASSADOR_JSON_LOGGING", "true")
	testInit()

	fm, isJSONFormatter := logrusLogger.Formatter.(*logrus.JSONFormatter)
	if !assert.True(t, isJSONFormatter) {
		return
	}
	assert.Equal(t, "2006-01-02 15:04:05.0000", fm.TimestampFormat)
}
