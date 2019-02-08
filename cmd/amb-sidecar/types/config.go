package types

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type Config struct {
	AmbassadorID              string
	AmbassadorNamespace       string
	AmbassadorSingleNamespace bool

	// Auth
	AuthProviderURL string
	IssuerURL       string
	LogLevel        string
	PKey            string
	PvtKPath        string
	PubKPath        string
	BaseURL         *url.URL
	StateTTL        time.Duration

	// Rate Limit
	Output string

	Error error
}

func InitializeFlags(flags *pflag.FlagSet) func() *Config {
	authCfg := &Config{}

	flags.StringVar(&authCfg.AuthProviderURL, "auth_provider_url", os.Getenv("AUTH_PROVIDER_URL"), "sets the authorization provider's url")
	flags.StringVar(&authCfg.LogLevel, "log_level", os.Getenv("APP_LOG_LEVEL"), "sets app's log level")
	flags.StringVar(&authCfg.PvtKPath, "private_key", os.Getenv("APP_PRIVATE_KEY_PATH"), "set's the path for private key file")
	flags.StringVar(&authCfg.PubKPath, "public_key", os.Getenv("APP_PUBLIC_KEY_PATH"), "set's the path for public key file")

	var stateTTL int64
	flags.Int64Var(&stateTTL, "state_ttl", 5, "TTL (in minutes) of a signed state token; default 5")

	return func() *Config {
		authCfg.AmbassadorID = os.Getenv("AMBASSADOR_ID")
		if authCfg.AmbassadorID == "" {
			authCfg.AmbassadorID = "default"
		}
		authCfg.AmbassadorNamespace = os.Getenv("AMBASSADOR_NAMESPACE")
		if authCfg.AmbassadorNamespace == "" {
			authCfg.AmbassadorNamespace = "default"
		}
		authCfg.AmbassadorSingleNamespace = os.Getenv("AMBASSADOR_SINGLE_NAMESPACE") != ""

		authCfg.StateTTL = time.Duration(stateTTL) * time.Minute

		if authCfg.LogLevel == "" {
			authCfg.LogLevel = logrus.InfoLevel.String()
		}

		u, err := url.Parse(authCfg.AuthProviderURL)
		if err != nil {
			authCfg.Error = errors.Errorf("parsing AUTH_PROVIDER_URL: %v", err)
			return authCfg
		}

		if u.Scheme == "" {
			authCfg.Error = errors.Errorf("AUTH_PROVIDER_URL is missing scheme. Format is `SCHEME://HOST[:PORT]'. Got: %v", authCfg.AuthProviderURL)
			return authCfg
		}

		authCfg.BaseURL = u
		authCfg.IssuerURL = fmt.Sprintf("%s://%s/", u.Scheme, u.Host)

		return authCfg
	}
}
