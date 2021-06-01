package busy

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLoggingTextFormatterDefault(t *testing.T) {
	os.Unsetenv("AMBASSADOR_JSON_LOGGING")
	Init()

	formatter := GetLogrusFormatter()
	fm, isTextFormatter := formatter.(*logrus.TextFormatter)
	assert.True(t, isTextFormatter)
	assert.Equal(t, "2006-01-02 15:04:05", fm.TimestampFormat)
	assert.True(t, fm.FullTimestamp)
}

func TestLoggingJsonFormatter(t *testing.T) {
	os.Setenv("AMBASSADOR_JSON_LOGGING", "true")
	Init()

	formatter := GetLogrusFormatter()
	fm, isJSONFormatter := formatter.(*logrus.JSONFormatter)
	assert.True(t, isJSONFormatter)
	assert.Equal(t, "2006-01-02 15:04:05", fm.TimestampFormat)
}
