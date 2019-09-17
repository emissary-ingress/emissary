package types

import (
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type PortalConfig struct {
	AmbassadorAdminURL    *url.URL
	AmbassadorInternalURL *url.URL
	AmbassadorExternalURL *url.URL
	PollFrequency         time.Duration
	ContentURL            *url.URL
}

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

	// gostats - This mimics vendor/github.com/lyft/gostats/settings.go
	UseStatsd      bool
	StatsdHost     string
	StatsdPort     int
	FlushIntervalS int

	// DevPortal
	PortalConfig PortalConfig
}

func getenvDefault(varname, def string) string {
	ret := os.Getenv(varname)
	if ret == "" {
		ret = def
	}
	return ret
}

func getenvDefaultSeconds(varname, def string, warn []error, fatal []error) (time.Duration, []error, []error) {
	valueStr := getenvDefault(varname, def)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		value, err2 := strconv.Atoi(def)
		if err2 != nil {
			fatal = append(fatal,
				errors.Errorf("%s: Unparseable default duration '%s': %s",
					varname, def, err2))
		} else {
			warn = append(warn,
				errors.Errorf("%s: Using default %ds. %s",
					varname, value, err))
		}
	}
	return time.Second * time.Duration(value), warn, fatal
}

func getenvDefaultURL(varname, def string, warn []error, fatal []error) (*url.URL, []error, []error) {
	u, err := url.Parse(getenvDefault(varname, def))
	if err == nil {
		if !u.IsAbs() || u.Host == "" {
			err = errors.New("URL is not absolute")
		}
	}
	if err != nil {
		warn = append(warn, errors.Wrapf(err, "invalid %s (falling back to default %s)", varname, def))
		u, err = url.Parse(def)
		if err != nil {
			// Since the default should be a hard-coded
			// good value, this should _never_ happen, and
			// is a panic.
			panic(err)
		}
	}
	return u, warn, fatal
}

func PortalConfigFromEnv(warn []error, fatal []error) (PortalConfig, []error, []error) {
	cfg := PortalConfig{}

	cfg.AmbassadorAdminURL, warn, fatal = getenvDefaultURL("AMBASSADOR_ADMIN_URL", "http://127.0.0.1:8877/", warn, fatal)
	cfg.AmbassadorInternalURL, warn, fatal = getenvDefaultURL("AMBASSADOR_INTERNAL_URL", "https://127.0.0.1:8443/", warn, fatal)
	cfg.AmbassadorExternalURL, warn, fatal = getenvDefaultURL("AMBASSADOR_URL", "https://api.example.com", warn, fatal)
	cfg.PollFrequency, warn, fatal = getenvDefaultSeconds("POLL_EVERY_SECS", "60", warn, fatal)
	cfg.ContentURL, warn, fatal = getenvDefaultURL("APRO_DEVPORTAL_CONTENT_URL", "https://github.com/datawire/devportal-content", warn, fatal)

	return cfg, warn, fatal
}

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	// DevPortal
	portalConfig, warn, fatal := PortalConfigFromEnv(warn, fatal)

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

		// gostats - This mimics vendor/github.com/lyft/gostats/settings.go,
		// but the defaults aren't nescessarily the same.
		UseStatsd:      false, // set below
		StatsdHost:     getenvDefault("STATSD_HOST", "localhost"),
		StatsdPort:     0, // set below
		FlushIntervalS: 0, // set below

		// DevPortal
		PortalConfig: portalConfig, // parsed above
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
