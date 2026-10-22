package busy

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	//nolint:depguard // This is one of the few places where it is approrpiate to not go through
	// dlog: to initialize dlog.
	"github.com/sirupsen/logrus"
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

func TestIsGracefulShutdownError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "signal terminated",
			err:      errors.New("received signal terminated (triggering graceful shutdown)"),
			expected: true,
		},
		{
			name:     "signal interrupt",
			err:      errors.New("received signal interrupt (triggering graceful shutdown)"),
			expected: true,
		},
		{
			name:     "graceful shutdown already triggered",
			err:      errors.New("received signal terminated (graceful shutdown already triggered; triggering not-so-graceful shutdown)"),
			expected: true,
		},
		{
			name:     "not-so-graceful shutdown already triggered",
			err:      errors.New("received signal terminated (not-so-graceful shutdown already triggered)"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some other error occurred"),
			expected: false,
		},
		{
			name:     "empty error",
			err:      errors.New(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGracefulShutdownError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
