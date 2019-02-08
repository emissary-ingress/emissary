package types

import (
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AmbassadorID              string
	AmbassadorNamespace       string
	AmbassadorSingleNamespace bool

	// Auth
	AuthProviderURL string        // the Identity Provider's URL
	LogLevel        string        // auth log level ("error" < "warn"/"warning" < "info" < "debug" < "trace")
	PvtKPath        string        // the path for private key file
	PubKPath        string        // the path for public key file
	BaseURL         *url.URL      // (this is just AuthProviderURL, but as a *url.URL)
	StateTTL        time.Duration // TTL (in minutes) of a signed state token

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

func ConfigFromEnv() (Config, []error) {
	var errs []error

	ret := Config{
		AmbassadorID:              getenvDefault("AMBASSADOR_ID", "default"),
		AmbassadorNamespace:       getenvDefault("AMBASSADOR_NAMESPACE", "default"),
		AmbassadorSingleNamespace: os.Getenv("AMBASSADOR_SINGLE_NAMESPACE") != "",

		// Auth
		AuthProviderURL: os.Getenv("AUTH_PROVIDER_URL"),
		//IssuerURL: (this is just AuthProviderURL, but as a *url.URL)
		LogLevel: getenvDefault("APP_LOG_LEVEL", "info"),
		PvtKPath: os.Getenv("APP_PRIVATE_KEY_PATH"),
		PubKPath: os.Getenv("APP_PUBLIC_KEY_PATH"),
		//BaseURL: (derived from AuthProviderURL)
		//StateTTL: (see below)

		// Rate Limit
		Output: os.Getenv("RLS_RUNTIME_DIR"),
	}

	u, err := url.Parse(ret.AuthProviderURL)
	switch {
	case err != nil:
		errs = append(errs, errors.Wrap(err, "invalid AUTH_PROVIDER_URL (disabling auth service)"))
	case u.Scheme == "":
		errs = append(errs, errors.Errorf("invalid AUTH_PROVIDER_URL (disabling auth service): missing scheme. Format is `SCHEME://HOST[:PORT]'. Got: %v", ret.AuthProviderURL))
	default:
		ret.BaseURL = u
	}

	if _, err := logrus.ParseLevel(ret.LogLevel); err != nil {
		errs = append(errs, errors.Wrap(err, "invalid APP_LOG_LEVEL (falling back to default \"info\")"))
		ret.LogLevel = "info"
	}

	ret.StateTTL, err = time.ParseDuration(getenvDefault("AUTH_STATE_TTL", "5m"))
	if err != nil {
		errs = append(errs, errors.Wrap(err, "invalid AUTH_STATE_TTL (falling back to default \"5m\")"))
		ret.StateTTL = 5 * time.Minute
	}

	if ret.Output == "" {
		errs = append(errs, errors.Errorf("must set RLS_RUNTIME_DIR (disabling ratelimit service)"))
	}

	return ret, errs
}
