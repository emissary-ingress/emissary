package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Config is an sigleton-object that holds all configuration values
// used by this app.
type Config struct {
	AuthProviderURL string
	IssuerURL       string
	LogLevel        string
	PKey            string
	PvtKPath        string
	PubKPath        string
	BaseURL         *url.URL
	StateTTL        time.Duration
}

var instance *Config

// New loads all the env variables, cli parameters and returns
// a reference to the config instance.
func New() *Config {
	if instance == nil {
		instance = &Config{}

		flag.StringVar(&instance.AuthProviderURL, "auth_provider_url", os.Getenv("AUTH_PROVIDER_URL"), "sets the authorization provider's url")
		flag.StringVar(&instance.LogLevel, "log_level", os.Getenv("APP_LOG_LEVEL"), "sets app's log level")
		flag.StringVar(&instance.PvtKPath, "private_key", os.Getenv("APP_PRIVATE_KEY_PATH"), "set's the path for private key file")
		flag.StringVar(&instance.PubKPath, "public_key", os.Getenv("APP_PUBLIC_KEY_PATH"), "set's the path for public key file")

		var stateTTL int64
		flag.Int64Var(&stateTTL, "state_ttl", 5, "TTL (in minutes) of a signed state token; default 5")

		flag.Parse()

		instance.StateTTL = time.Duration(stateTTL) * time.Minute

		if instance.LogLevel == "" {
			instance.LogLevel = logrus.InfoLevel.String()
		}

		if err := instance.Validate(); err != nil {
			log.Printf("config error: %v", err)
		}
	}

	return instance
}

// Validate checks the supplied configuration.
func (c *Config) Validate() error {
	u, err := url.Parse(c.AuthProviderURL)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("parsing AUTH_PROVIDER_URL: %v", err))
	}

	if u.Scheme == "" {
		return errors.New("AUTH_PROVIDER_URL is missing scheme. Acceptable formats: [scheme]://[host] or [scheme]://[host]:[port]")
	}

	instance.BaseURL = u
	instance.IssuerURL = fmt.Sprintf("%s://%s/", u.Scheme, u.Host)

	return nil
}
