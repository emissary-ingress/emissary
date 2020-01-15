package types

import (
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
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
	// Kubernetes
	PodNamespace string `env:"POD_NAMESPACE ,parser=nonempty-string "` // there's a hack to set the default below

	// Ambassador
	AmbassadorID              string `env:"AMBASSADOR_ID               ,parser=nonempty-string ,default=default "`
	AmbassadorClusterID       string `env:"AMBASSADOR_CLUSTER_ID       ,parser=possibly-empty-string            "`
	AmbassadorNamespace       string `env:"AMBASSADOR_NAMESPACE        ,parser=nonempty-string ,default=default "`
	AmbassadorSingleNamespace bool   `env:"AMBASSADOR_SINGLE_NAMESPACE ,parser=empty/nonempty                   "`

	// General
	HTTPPort        string `env:"DEV_AES_HTTP_PORT ,parser=nonempty-string       ,default=8500 "`
	LogLevel        string `env:"AES_LOG_LEVEL     ,parser=logrus.ParseLevel     ,default=info "` // log level ("error" < "warn"/"warning" < "info" < "debug" < "trace")
	RedisPoolSize   int    `env:"REDIS_POOL_SIZE   ,parser=strconv.ParseInt      ,default=10   "`
	RedisSocketType string `env:"REDIS_SOCKET_TYPE ,parser=nonempty-string       ,default=tcp  "`
	RedisURL        string `env:"REDIS_URL         ,parser=possibly-empty-string               "` // if empty, disables AES features

	// Rate Limit
	RLSRuntimeDir              string `env:"DEV_RLS_RUNTIME_DIR           ,parser=nonempty-string       ,default=/tmp/amb/config "` // e.g.: "/tmp/amb-sidecar.XYZ/rls-snapshot"; same as the RUNTIME_ROOT for Lyft ratelimit.  Must point to a _symlink_ to a directory, not a real directory.  The symlink need not already exist at launch.  Unlike Lyft ratelimit, the parent doesn't need to exist at launch either.
	RLSRuntimeSubdir           string `env:",const=true                   ,parser=nonempty-string       ,default=config          "` // directory inside of RLSRuntimeDir; same as the RUNTIME_SUBDIRECTORY for Lyft ratelimit
	RedisPerSecond             bool   `env:"REDIS_PERSECOND               ,parser=strconv.ParseBool     ,default=false           "`
	RedisPerSecondPoolSize     int    `env:"REDIS_PERSECOND_POOL_SIZE     ,parser=strconv.ParseInt      ,default=10              "`
	RedisPerSecondSocketType   string `env:"REDIS_PERSECOND_SOCKET_TYPE   ,parser=possibly-empty-string                          "` // validated manually
	RedisPerSecondURL          string `env:"REDIS_PERSECOND_URL           ,parser=possibly-empty-string                          "` // validated manually
	ExpirationJitterMaxSeconds int64  `env:"EXPIRATION_JITTER_MAX_SECONDS ,parser=strconv.ParseInt      ,default=300             "`

	// Developer Portal
	AmbassadorAdminURL     *url.URL      `env:"DEV_AMBASSADOR_ADMIN_URL      ,parser=absolute-URL    ,default=http://127.0.0.1:8877/                            "`
	AmbassadorInternalURL  *url.URL      `env:"DEV_AMBASSADOR_INTERNAL_URL   ,parser=absolute-URL    ,default=https://127.0.0.1:8443/                           "`
	AmbassadorExternalURL  *url.URL      `env:"AMBASSADOR_URL                ,parser=absolute-URL    ,default=https://api.example.com/                          "`
	DevPortalPollInterval  time.Duration `env:"POLL_EVERY_SECS               ,parser=integer-seconds ,default=60                                                "`
	DevPortalContentURL    *url.URL      `env:"DEVPORTAL_CONTENT_URL         ,parser=absolute-URL    ,default=https://github.com/datawire/devportal-content.git "`
	DevPortalContentSubdir string        `env:"DEVPORTAL_CONTENT_DIR         ,parser=nonempty-string ,default=/"`
	DevPortalContentBranch string        `env:"DEVPORTAL_CONTENT_BRANCH      ,parser=nonempty-string ,default=master"`

	// Local development
	DevWebUIPort         string `env:"DEV_WEBUI_PORT ,parser=possibly-empty-string                                     "`
	DevWebUIDir          string `env:"DEV_WEBUI_DIR  ,parser=nonempty-string       ,default=/ambassador/webui/bindata/ "`
	DevWebUISnapshotHost string `env:"DEV_WEBUI_SNAPSHOT_HOST ,parser=possibly-empty-string"`
	DevWebUIWebstorm     string `env:"DEV_WEBUI_WEBSTORM ,parser=possibly-empty-string"`

	// gostats - This mimics vendor/github.com/lyft/gostats/settings.go,
	// but the defaults aren't nescessarily the same.
	UseStatsd     bool          `env:"USE_STATSD                     ,parser=strconv.ParseBool ,default=false     "`
	StatsdHost    string        `env:"STATSD_HOST                    ,parser=nonempty-string   ,default=localhost "`
	StatsdPort    int           `env:"STATSD_PORT                    ,parser=strconv.ParseInt  ,default=8125      "`
	FlushInterval time.Duration `env:"GOSTATS_FLUSH_INTERVAL_SECONDS ,parser=integer-seconds   ,default=5         "`
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

// podNamespace is stolen from
// "k8s.io/client-go/tools/clientcmd".inClusterConfig.Namespace()
func podNamespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	os.Setenv("POD_NAMESPACE", podNamespace())

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
