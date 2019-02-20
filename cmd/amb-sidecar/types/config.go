package types

import (
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Ambassador
	AmbassadorID              string
	AmbassadorNamespace       string
	AmbassadorSingleNamespace bool

	// Auth
	PvtKPath string // the path for private key file
	PubKPath string // the path for public key file

	// Rate Limit
	Output string // e.g.: "/run/amb/config"; same as the RUNTIME_ROOT for Lyft ratelimit

	// General
	LogLevel string // log level ("error" < "warn"/"warning" < "info" < "debug" < "trace")
}

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	cfg = Config{
		// Ambassador
		AmbassadorID:              getenvDefault("AMBASSADOR_ID", "default"),
		AmbassadorNamespace:       getenvDefault("AMBASSADOR_NAMESPACE", "default"),
		AmbassadorSingleNamespace: os.Getenv("AMBASSADOR_SINGLE_NAMESPACE") != "",

		// Auth
		PvtKPath: os.Getenv("APP_PRIVATE_KEY_PATH"),
		PubKPath: os.Getenv("APP_PUBLIC_KEY_PATH"),

		// Rate Limit
		Output: os.Getenv("RLS_RUNTIME_DIR"),

		// General
		LogLevel: getenvDefault("APP_LOG_LEVEL", "info"),
	}

	if _, err := logrus.ParseLevel(cfg.LogLevel); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid APP_LOG_LEVEL (falling back to default \"info\")"))
		cfg.LogLevel = "info"
	}

	if cfg.Output == "" {
		fatal = append(fatal, errors.Errorf("must set RLS_RUNTIME_DIR (aborting)"))
	}

	return cfg, warn, fatal
}
