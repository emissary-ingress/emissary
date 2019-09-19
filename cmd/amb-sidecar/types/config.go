package types

import (
	"net/url"
	"os"
	"strconv"
	"time"

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

	// Developer Portal
	AmbassadorAdminURL    *url.URL
	AmbassadorInternalURL *url.URL
	AmbassadorExternalURL *url.URL
	DevPortalPollInterval time.Duration
	DevPortalContentURL   *url.URL

	// gostats - This mimics vendor/github.com/lyft/gostats/settings.go
	UseStatsd      bool
	StatsdHost     string
	StatsdPort     int
	FlushIntervalS int
}

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

func parseAbsoluteURL(str string) (*url.URL, error) {
	u, err := url.Parse(str)
	if err != nil {
		return nil, err
	}
	if !u.IsAbs() || u.Host == "" {
		return nil, errors.New("URL is not absolute")
	}
	return u, nil
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

		// Developer Portal
		AmbassadorAdminURL:    nil, // set below
		AmbassadorInternalURL: nil, // set below
		AmbassadorExternalURL: nil, // set below
		DevPortalPollInterval: 0,   // set below
		DevPortalContentURL:   nil, // set below

		// gostats - This mimics vendor/github.com/lyft/gostats/settings.go,
		// but the defaults aren't nescessarily the same.
		UseStatsd:      false, // set below
		StatsdHost:     getenvDefault("STATSD_HOST", "localhost"),
		StatsdPort:     0, // set below
		FlushIntervalS: 0, // set below
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
	if cfg.AmbassadorAdminURL, err = parseAbsoluteURL(getenvDefault("AMBASSADOR_ADMIN_URL", "http://127.0.0.1:8877/")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid AMBASSADOR_ADMIN_URL (falling back to default http://127.0.0.1:8877/)"))
		cfg.AmbassadorAdminURL, _ = parseAbsoluteURL("http://127.0.0.1:8877/")
	}
	if cfg.AmbassadorInternalURL, err = parseAbsoluteURL(getenvDefault("AMBASSADOR_INTERNAL_URL", "https://127.0.0.1:8443/")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid AMBASSADOR_INTERNAL_URL (falling back to default https://127.0.0.1:8443/)"))
		cfg.AmbassadorInternalURL, _ = parseAbsoluteURL("https://127.0.0.1:8443/")
	}
	if cfg.AmbassadorExternalURL, err = parseAbsoluteURL(getenvDefault("AMBASSADOR_URL", "https://api.example.com/")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid AMBASSADOR_URL (falling back to default https://api.example.com/)"))
		cfg.AmbassadorExternalURL, _ = parseAbsoluteURL("https://api.example.com/")
	}
	if seconds, err := strconv.Atoi(getenvDefault("POLL_EVERY_SECS", "60")); err == nil {
		cfg.DevPortalPollInterval = time.Duration(seconds) * time.Second
	} else {
		warn = append(warn, errors.Wrap(err, "invalid POLL_EVERY_SECS (falling back to default 60)"))
		cfg.DevPortalPollInterval = 60 * time.Second
	}
	if cfg.DevPortalContentURL, err = parseAbsoluteURL(getenvDefault("APRO_DEVPORTAL_CONTENT_URL", "https://github.com/datawire/devportal-content.git")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid APRO_DEVPORTAL_CONTENT_URL (falling back to default https://github.com/datawire/devportal-content.git)"))
		cfg.DevPortalContentURL, _ = parseAbsoluteURL("https://github.com/datawire/devportal-content.git")
	}
	if cfg.UseStatsd, err = strconv.ParseBool(getenvDefault("USE_STATSD", "false")); err != nil { // NB: default here differs
		warn = append(warn, errors.Wrap(err, "invalid USE_STATSD (falling back to default false)"))
		cfg.UseStatsd = false
	}
	if cfg.StatsdPort, err = strconv.Atoi(getenvDefault("STATSD_PORT", "8125")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid STATSD_PORT (falling back to default 8125)"))
		cfg.StatsdPort = 8125
	}
	if cfg.FlushIntervalS, err = strconv.Atoi(getenvDefault("GOSTATS_FLUSH_INTERVAL_SECONDS", "5")); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid GOSTATS_FLUSH_INTERVAL_SECONDS (falling back to default 5)"))
		cfg.FlushIntervalS = 5
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

	// Export the settings so that
	// github.com/lyft/stats.GetSettings() sees them, since we may
	// have different defaults.
	os.Setenv("USE_STATSD", strconv.FormatBool(cfg.UseStatsd))
	os.Setenv("STATSD_HOST", cfg.StatsdHost)
	os.Setenv("STATSD_PORT", strconv.Itoa(cfg.StatsdPort))
	os.Setenv("GOSTATS_FLUSH_INTERVAL_SECONDS", strconv.Itoa(cfg.FlushIntervalS))

	return cfg, warn, fatal
}
