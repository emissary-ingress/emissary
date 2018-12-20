package config

import (
	"flag"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Config is an sigleton-object that holds all configuration values
// used by this app.
type Config struct {
	Domain     string
	Kubeconfig string
	Level      string
	PKey       string
	PvtKPath   string
	PubKPath   string
	BaseURL    *url.URL
	StateTTL   time.Duration
}

var instance *Config

// New loads all the env variables, cli parameters and returns
// a reference to the config instance.
func New() *Config {
	if instance == nil {
		instance = &Config{}

		// TODO(gsagula): rename this to auth_domain or something else.
		flag.StringVar(&instance.Domain, "domain", os.Getenv("AUTH_DOMAIN"), "authorization and identity provider's domain")
		flag.StringVar(&instance.Level, "level", logrus.DebugLevel.String(), "sets log level")
		flag.StringVar(&instance.PvtKPath, "private_key", os.Getenv("APP_PRIVATE_KEY_PATH"), "path for private key file")
		flag.StringVar(&instance.PubKPath, "public_key", os.Getenv("APP_PUBLIC_KEY_PATH"), "path for public key file")

		var stateTTL int64
		flag.Int64Var(&stateTTL, "state_ttl", 5, "TTL (in minutes) of a signed state token; default 5")

		flag.Parse()

		instance.StateTTL = time.Duration(stateTTL) * time.Minute

		log.Println("validating required configuration")
		if err := instance.Validate(); err != nil {
			log.Printf("config error: %v", err)
		}

		// TODO(gsagula): sure, but there is a better way to do this.
		s := "https"
		if !strings.Contains(instance.Domain, "auth0") {
			s = "http"
		}

		instance.BaseURL = &url.URL{
			Host:   instance.Domain,
			Scheme: s,
		}
	}

	return instance
}

// Validate checks the supplied configuration.
func (c *Config) Validate() error {
	messages := []string{}
	msg := func(m string) {
		messages = append(messages, m)
	}

	if len(c.Domain) < 3 {
		msg("domain")
	}

	return nil
}
