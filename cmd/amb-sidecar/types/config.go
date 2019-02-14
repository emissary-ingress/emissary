package types

import (
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Ambassador
	AmbassadorID              string
	AmbassadorNamespace       string
	AmbassadorSingleNamespace bool

	// Auth
	AuthProviderURL *url.URL      // the Identity Provider's URL
	PvtKPath        string        // the path for private key file
	PubKPath        string        // the path for public key file
	StateTTL        time.Duration // TTL (in minutes) of a signed state token

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
		//AuthProviderURL: (see below),
		PvtKPath: os.Getenv("APP_PRIVATE_KEY_PATH"),
		PubKPath: os.Getenv("APP_PUBLIC_KEY_PATH"),
		//StateTTL: (see below),

		// Rate Limit
		Output: os.Getenv("RLS_RUNTIME_DIR"),

		// General
		LogLevel: getenvDefault("APP_LOG_LEVEL", "info"),
	}

	var err error
	cfg.AuthProviderURL, err = url.Parse(os.Getenv("AUTH_PROVIDER_URL"))
	switch {
	case err != nil:
		fatal = append(fatal, errors.Wrap(err, "invalid AUTH_PROVIDER_URL (aborting)"))
	case !cfg.AuthProviderURL.IsAbs():
		fatal = append(fatal, errors.Errorf("invalid AUTH_PROVIDER_URL (aborting): is not absolute"))
		cfg.AuthProviderURL = nil
	case cfg.AuthProviderURL.Path == "":
		// net/http doesn't normalize "" to "/" because they
		// are different for OPTIONS requests; but for all
		// other requests it should be normalized to "/"
		// (RFC7230§2.7.3).  We won't be doing any OPTIONS
		// requests, so normalize it.
		cfg.AuthProviderURL.Path = "/"
		cfg.AuthProviderURL.RawPath = "/"
	}

	if _, err := logrus.ParseLevel(cfg.LogLevel); err != nil {
		warn = append(warn, errors.Wrap(err, "invalid APP_LOG_LEVEL (falling back to default \"info\")"))
		cfg.LogLevel = "info"
	}

	cfg.StateTTL, err = time.ParseDuration(getenvDefault("AUTH_STATE_TTL", "5m"))
	if err != nil {
		warn = append(warn, errors.Wrap(err, "invalid AUTH_STATE_TTL (falling back to default \"5m\")"))
		cfg.StateTTL = 5 * time.Minute
	}

	if cfg.Output == "" {
		fatal = append(fatal, errors.Errorf("must set RLS_RUNTIME_DIR (aborting)"))
	}

	return cfg, warn, fatal
}
