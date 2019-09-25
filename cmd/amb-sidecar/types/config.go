package types

import (
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/types/internal/envconfig"
)

// Config stores all of the global amb-sidecar settings, which are configured from environment variables.  We should
// keep these to a minimum, and configure as much as we can in CRDs.
//
// This uses a custom `envconfig` annotation processor, in a similar spirit to github.com/kelseyhightower/envconfig
// (which has much less flexibility around falling back to default values and error handling).
//
//  - Add new field types by editing ./internal/envconfig/envconfig_types.go (should be straight-forward)
//  - Add new parser= values by editing ./internal/envconfig/envconfig_types.go (should be straight-forward)
//  - Add new key=value options by editing ./internal/envconfig/envconfig.go (involves "reflect" wizardry)
type Config struct {
	// Ambassador
	AmbassadorID              string `env:"AMBASSADOR_ID,parser=nonempty-string,default=default"`
	AmbassadorNamespace       string `env:"AMBASSADOR_NAMESPACE,parser=nonempty-string,default=default"`
	AmbassadorSingleNamespace bool   `env:"AMBASSADOR_SINGLE_NAMESPACE,parser=empty/nonempty"`

	// General
	HTTPPort        string `env:"APRO_HTTP_PORT,parser=nonempty-string,default=8500"`
	LogLevel        string `env:"APP_LOG_LEVEL,default=info,parser=logrus.ParseLevel"` // log level ("error" < "warn"/"warning" < "info" < "debug" < "trace")
	RedisPoolSize   int    `env:"REDIS_POOL_SIZE,default=10"`
	RedisSocketType string `env:"REDIS_SOCKET_TYPE,parser=nonempty-string"`
	RedisURL        string `env:"REDIS_URL,parser=nonempty-string"`

	// Auth (filters)
	KeyPairSecretName      string `env:"APRO_KEYPAIR_SECRET_NAME,parser=nonempty-string,default=ambassador-pro-keypair"`
	KeyPairSecretNamespace string `env:"APRO_KEYPAIR_SECRET_NAMESPACE,parser=nonempty-string,defaultFrom=AmbassadorNamespace"`

	// Rate Limit
	RLSRuntimeDir              string `env:"RLS_RUNTIME_DIR,parser=nonempty-string,default=/tmp/amb/config"` // e.g.: "/tmp/amb-sidecar.XYZ/rls-snapshot"; same as the RUNTIME_ROOT for Lyft ratelimit.  Must point to a _symlink_ to a directory, not a real directory.  The symlink need not already exist at launch.  Unlike Lyft ratelimit, the parent doesn't need to exist at launch either.
	RLSRuntimeSubdir           string `env:",const=true,parser=nonempty-string,default=config"`              // directory inside of RLSRuntimeDir; same as the RUNTIME_SUBDIRECTORY for Lyft ratelimit
	RedisPerSecond             bool   `env:"REDIS_PERSECOND,parser=strconv.ParseBool,default=false"`
	RedisPerSecondPoolSize     int    `env:"REDIS_PERSECOND_POOL_SIZE,default=10"`
	RedisPerSecondSocketType   string `env:"REDIS_PERSECOND_SOCKET_TYPE,parser=possibly-empty-string"` // validated manually
	RedisPerSecondURL          string `env:"REDIS_PERSECOND_URL,parser=possibly-empty-string"`         // validated manually
	ExpirationJitterMaxSeconds int64  `env:"EXPIRATION_JITTER_MAX_SECONDS,default=300"`

	// Developer Portal
	AmbassadorAdminURL    *url.URL      `env:"AMBASSADOR_ADMIN_URL,default=http://127.0.0.1:8877/"`
	AmbassadorInternalURL *url.URL      `env:"AMBASSADOR_INTERNAL_URL,default=https://127.0.0.1:8443/"`
	AmbassadorExternalURL *url.URL      `env:"AMBASSADOR_URL,default=https://api.example.com/"`
	DevPortalPollInterval time.Duration `env:"POLL_EVERY_SECS,parser=integer-seconds,default=60"`
	DevPortalContentURL   *url.URL      `env:"APRO_DEVPORTAL_CONTENT_URL,default=https://github.com/datawire/devportal-content.git"`

	// gostats - This mimics vendor/github.com/lyft/gostats/settings.go,
	// but the defaults aren't nescessarily the same.
	UseStatsd     bool          `env:"USE_STATSD,parser=strconv.ParseBool,default=false"`
	StatsdHost    string        `env:"STATSD_HOST,parser=nonempty-string,default=localhost"`
	StatsdPort    int           `env:"STATSD_PORT,default=8125"`
	FlushInterval time.Duration `env:"GOSTATS_FLUSH_INTERVAL_SECONDS,parser=integer-seconds,default=5"`
}

var configParser = func() envconfig.StructParser {
	ret, err := envconfig.GenerateParser(reflect.TypeOf(Config{}))
	if err != nil {
		// panic, because it means that the definition of
		// 'Config' is invalid.
		panic(err)
	}
	return ret
}()

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	warn, fatal = configParser.ParseFromEnv(&cfg)

	if cfg.RedisPerSecond {
		if cfg.RedisPerSecondSocketType == "" {
			warn = append(warn, errors.Wrap(envconfig.ErrorNotSet, "invalid REDIS_PERSECOND_SOCKET_TYPE (disabling REDIS_PERSECOND)"))
			cfg.RedisPerSecond = false
		}
		if cfg.RedisPerSecondURL == "" {
			warn = append(warn, errors.Wrap(envconfig.ErrorNotSet, "invalid REDIS_PERSECOND_URL (disabling REDIS_PERSECOND)"))
			cfg.RedisPerSecond = false
		}
	}

	// Export the settings so that
	// github.com/lyft/stats.GetSettings() sees them, since we may
	// have different defaults.
	os.Setenv("USE_STATSD", strconv.FormatBool(cfg.UseStatsd))
	os.Setenv("STATSD_HOST", cfg.StatsdHost)
	os.Setenv("STATSD_PORT", strconv.Itoa(cfg.StatsdPort))
	os.Setenv("GOSTATS_FLUSH_INTERVAL_SECONDS", strconv.Itoa(int(cfg.FlushInterval.Seconds())))

	return cfg, warn, fatal
}
