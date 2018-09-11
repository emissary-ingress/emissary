package config

import (
	"errors"
	"flag"
	"os"
	"time"
)

//Config ..
type Config struct {
	Audience      string
	CallbackURL   string
	Domain        string
	ClientID      string
	Scheme        string
	Kubeconfig    string
	DenyOnFailure bool
	Quiet         bool
	StateTTL      time.Duration
}

// NewConfig ..
func NewConfig() (*Config, error) {
	c := &Config{}

	flag.StringVar(&c.Kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")
	flag.BoolVar(&c.Quiet, "quiet", false, "restrict logs to error only")
	flag.StringVar(&c.Audience, "audience", os.Getenv("AUTH_AUDIENCE"), "audience provided by the identity provider")
	flag.StringVar(&c.Domain, "domain", os.Getenv("AUTH_DOMAIN"), "authorization service domain")
	flag.StringVar(&c.ClientID, "client_id", os.Getenv("AUTH_CLIENT_ID"), "client id provided by the identity provider")
	flag.StringVar(&c.Scheme, "scheme", "https", "use secure scheme when calling the authorization server")
	flag.StringVar(&c.CallbackURL, "callback_url", os.Getenv("AUTH_CALLBACK_URL"), "url that the idp should call the authorization server")

	var stateTTL int64
	flag.Int64Var(&stateTTL, "state_ttl", 5, "TTL (in minutes) of a signed state token; default 5")

	var onFailure string
	flag.StringVar(&onFailure, "on_failure", os.Getenv("AUTH_ON_FAILURE"), "tells the app what to do in case of failure; eg. <deny>")

	flag.Parse()

	c.StateTTL = time.Duration(stateTTL) * time.Minute

	if onFailure == "deny" {
		c.DenyOnFailure = true
	} else {
		c.DenyOnFailure = false
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return c, nil
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
