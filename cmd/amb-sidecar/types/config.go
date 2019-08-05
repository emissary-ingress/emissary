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
	RLSRuntimeDir              string // e.g.: "/tmp/amb-sidecar.XYZ/rls-snapshot"; same as the RUNTIME_ROOT for Lyft ratelimit.  Must point to a _symlink_ to a directory, not a real directory.  The symlink need not already exist at launch.  Unlike Lyft ratelimit, the parent doesn't need to exist at launch either.
	RLSRuntimeSubdir           string // directory inside of RLSRuntimeDir; same as the RUNTIME_SUBDIRECTORY for Lyft ratelimit
	RedisPerSecond             bool
	RedisPerSecondPoolSize     int
	RedisPerSecondSocketType   string
	RedisPerSecondURL          string
	ExpirationJitterMaxSeconds int64
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
		RedisSocketType: os.Getenv("REDIS_SOCKET_TYPE"),         // validated below
		RedisURL:        os.Getenv("REDIS_URL"),                 // validated below

		// Auth (filters)
		KeyPairSecretName:      getenvDefault("APRO_KEYPAIR_SECRET_NAME", "ambassador-pro-keypair"),
		KeyPairSecretNamespace: getenvDefault("APRO_KEYPAIR_SECRET_NAMESPACE", getenvDefault("AMBASSADOR_NAMESPACE", "default")),

		// Rate Limit
		RLSRuntimeDir:              getenvDefault("RLS_RUNTIME_DIR", "/tmp/amb/config"),
		RLSRuntimeSubdir:           "config",
		RedisPerSecond:             false,                                    // set below
		RedisPerSecondPoolSize:     0,                                        // set below
		RedisPerSecondSocketType:   os.Getenv("REDIS_PERSECOND_SOCKET_TYPE"), // validated below
		RedisPerSecondURL:          os.Getenv("REDIS_PERSECOND_URL"),         // validated below
		ExpirationJitterMaxSeconds: 0,                                        // set below
	}

	// Set the things marked "set below" (things that do require some parsing)
	var err error
	if cfg.RedisPoolSize, err = strconv.Atoi(getenvDefault("REDIS_POOL_SIZE", "10")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid REDIS_POOL_SIZE (falling back to default 10)"))
		cfg.RedisPoolSize = 10
	}
	if cfg.RedisPerSecond, err = strconv.ParseBool(getenvDefault("REDIS_PERSECOND", "false")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid REDIS_PERSECOND (falling back to default false)"))
		cfg.RedisPerSecond = false
	}
	if cfg.RedisPerSecond { // don't bother with REDIS_PER_SECOND_POOL_SIZE if !cfg.RedisPerSecond
		if cfg.RedisPerSecondPoolSize, err = strconv.Atoi(getenvDefault("REDIS_PERSECOND_POOL_SIZE", "10")); err != nil {
			warn = append(warn, errors.Wrap(err, "invalid REDIS_PERSECOND_POOL_SIZE (falling back to default 10)"))
			cfg.RedisPerSecondPoolSize = 10
		}
	}
	if cfg.ExpirationJitterMaxSeconds, err = strconv.ParseInt(getenvDefault("EXPIRATION_JITTER_MAX_SECONDS", "300"), 10, 0); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid EXPIRATION_JITTER_MAX_SECONDS (falling back to default 300)"))
		cfg.ExpirationJitterMaxSeconds = 300
	}

	// Validate things marked "validated below" (things that we can validate here)
	if _, err := logrus.ParseLevel(cfg.LogLevel); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid APP_LOG_LEVEL (falling back to default \"info\")"))
		cfg.LogLevel = "info"
	}
	if cfg.RedisSocketType == "" {
		fatal = append(fatal, errors.New("must set REDIS_SOCKET_TYPE (aborting)"))
	}
	if cfg.RedisURL == "" {
		fatal = append(fatal, errors.New("must set REDIS_URL (aborting)"))
	}
	if cfg.RedisPerSecond {
		if cfg.RedisPerSecondSocketType == "" {
			warn = append(warn, errors.Errorf("empty REDIS_PERSECOND_SOCKET_TYPE (disabling REDIS_PERSECOND)"))
			cfg.RedisPerSecond = false
		}
		if cfg.RedisPerSecondURL == "" {
			warn = append(warn, errors.Errorf("empty REDIS_PERSECOND_URL (disabling REDIS_PERSECOND)"))
			cfg.RedisPerSecond = false
		}
	}

	return cfg, warn, fatal
}
