package types

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Ambassador
	AmbassadorID              string
	AmbassadorNamespace       string
	AmbassadorSingleNamespace bool

	// General
	HTTPPort        string
	LogLevel        string // log level ("error" < "warn"/"warning" < "info" < "debug" < "trace")
	RedisPoolSize   int
	RedisSocketType string
	RedisURL        string

	// Auth (filters)
	KeyPairSecretName      string
	KeyPairSecretNamespace string

	// Rate Limit
	Output string // e.g.: "/run/amb/config"; same as the RUNTIME_ROOT for Lyft ratelimit
}

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	// Set the things that don't require too much parsing
	cfg = Config{
		// Ambassador
		AmbassadorID:              getenvDefault("AMBASSADOR_ID", "default"),
		AmbassadorNamespace:       getenvDefault("AMBASSADOR_NAMESPACE", "default"),
		AmbassadorSingleNamespace: os.Getenv("AMBASSADOR_SINGLE_NAMESPACE") != "",

		// General
		HTTPPort:        getenvDefault("APRO_HTTP_PORT", "8500"),
		LogLevel:        getenvDefault("APP_LOG_LEVEL", "info"), // validated below
		RedisPoolSize:   0,                                      // set below
		RedisSocketType: os.Getenv("REDIS_SOCKET_TYPE"),
		RedisURL:        os.Getenv("REDIS_URL"),

		// Auth (filters)
		KeyPairSecretName:      getenvDefault("APRO_KEYPAIR_SECRET_NAME", "ambassador-pro-keypair"),
		KeyPairSecretNamespace: getenvDefault("APRO_KEYPAIR_SECRET_NAMESPACE", getenvDefault("AMBASSADOR_NAMESPACE", "default")),

		// Rate Limit
		Output: os.Getenv("RLS_RUNTIME_DIR"), // validated below
	}

	// Set the things marked "set below" (things that do require some parsing)
	var err error
	if cfg.RedisPoolSize, err = strconv.Atoi(getenvDefault("REDIS_POOL_SIZE", "10")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid REDIS_POOL_SIZE (falling back to default 10)"))
		cfg.RedisPoolSize = 10
	}

	// Validate things marked "validated below" (things that we can validate here)
	if _, err := logrus.ParseLevel(cfg.LogLevel); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid APP_LOG_LEVEL (falling back to default \"info\")"))
		cfg.LogLevel = "info"
	}
	if cfg.Output == "" {
		fatal = append(fatal, errors.New("must set RLS_RUNTIME_DIR (aborting)"))
	}

	return cfg, warn, fatal
}
