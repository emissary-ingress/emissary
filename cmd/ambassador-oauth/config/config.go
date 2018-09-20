package config

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Config is an sigleton-object that holds all configuration values
// used by this app.
type Config struct {
	Audience      string
	CallbackURL   string
	Domain        string
	ClientID      string
	Scheme        string
	Kubeconfig    string
	Level         string
	PKey          string
	PvtKPath      string
	PubKPath      string
	DenyOnFailure bool
	StateTTL      time.Duration
}

var instance *Config

// New loads all the env variables, cli parameters and returns
// a reference to the config instance.
func New() *Config {
	if instance == nil {
		instance = &Config{}

		flag.StringVar(&instance.Kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")
		flag.StringVar(&instance.Level, "level", logrus.DebugLevel.String(), "restrict logs to error only")
		flag.StringVar(&instance.Audience, "audience", os.Getenv("AUTH_AUDIENCE"), "audience provided by the identity provider")
		flag.StringVar(&instance.Domain, "domain", os.Getenv("AUTH_DOMAIN"), "authorization service domain")
		flag.StringVar(&instance.ClientID, "client_id", os.Getenv("AUTH_CLIENT_ID"), "client id provided by the identity provider")
		flag.StringVar(&instance.Scheme, "scheme", "https", "use secure scheme when calling the authorization server")
		flag.StringVar(&instance.CallbackURL, "callback_url", os.Getenv("AUTH_CALLBACK_URL"), "url that the idp should call the authorization server")
		flag.StringVar(&instance.PvtKPath, "private_key", os.Getenv("APP_PRIVATE_KEY_PATH"), "path for private key file")
		flag.StringVar(&instance.PubKPath, "public_key", os.Getenv("APP_PUBLIC_KEY_PATH"), "path for public key file")

		var stateTTL int64
		flag.Int64Var(&stateTTL, "state_ttl", 5, "TTL (in minutes) of a signed state token; default 5")

		var onFailure string
		flag.StringVar(&onFailure, "on_failure", os.Getenv("AUTH_ON_FAILURE"), "tells the app what to do in case of failure; eg. <deny>")

		flag.Parse()

		instance.StateTTL = time.Duration(stateTTL) * time.Minute

		if onFailure == "deny" {
			instance.DenyOnFailure = true
		} else {
			instance.DenyOnFailure = false
		}

		if err := instance.validate(); err != nil {
			log.Fatalf("terminating with config error: %v", err)
			return nil
		}
	}

	return instance
}

func (c *Config) validate() error {
	if len(c.Audience) < 3 {
		return errors.New("audience is require")
	}

	if len(c.Domain) < 3 {
		return errors.New("domain is require")
	}

	if len(c.ClientID) < 3 {
		return errors.New("client_id is required")
	}

	if len(c.CallbackURL) < 3 {
		return errors.New("callback_url is required")
	}

	return nil
}
