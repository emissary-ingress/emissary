package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/client"
	"github.com/sirupsen/logrus"
)

// Config is an sigleton-object that holds all configuration values
// used by this app.
type Config struct {
	Audience      string
	CallbackURL   string
	Domain        string
	ClientID      string
	ClientSecret  string
	Scheme        string
	Kubeconfig    string
	Level         string
	PKey          string
	PvtKPath      string
	PubKPath      string
	Secure        bool
	DenyOnFailure bool
	BaseURL       *url.URL
	StateTTL      time.Duration
}

var instance *Config

// New loads all the env variables, cli parameters and returns
// a reference to the config instance.
func New() *Config {
	if instance == nil {
		instance = &Config{}

		flag.StringVar(&instance.ClientSecret, "client_secret", os.Getenv("AUTH_CLIENT_SECRET"), "client secret configured for this app")
		flag.StringVar(&instance.Kubeconfig, "kubeconfig", os.Getenv("KUBECONFIG"), "absolute path to the kubeconfig file")
		flag.StringVar(&instance.Level, "level", logrus.DebugLevel.String(), "restrict logs to error only")
		flag.StringVar(&instance.Audience, "audience", os.Getenv("AUTH_AUDIENCE"), "audience provided by the identity provider")
		flag.StringVar(&instance.Domain, "domain", os.Getenv("AUTH_DOMAIN"), "authorization service domain")
		flag.StringVar(&instance.ClientID, "client_id", os.Getenv("AUTH_CLIENT_ID"), "client id provided by the identity provider")
		flag.StringVar(&instance.Scheme, "scheme", "https", "use secure scheme when calling the authorization server")
		flag.StringVar(&instance.CallbackURL, "callback_url", os.Getenv("AUTH_CALLBACK_URL"), "url that the idp should call the authorization server")
		flag.StringVar(&instance.PvtKPath, "private_key", os.Getenv("APP_PRIVATE_KEY_PATH"), "path for private key file")
		flag.StringVar(&instance.PubKPath, "public_key", os.Getenv("APP_PUBLIC_KEY_PATH"), "path for public key file")

		if os.Getenv("APP_NOT_SECURE") != "" {
			instance.Secure = false
		} else {
			instance.Secure = true
		}

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

		log.Println("validating required configuration")
		if err := instance.validate(); err != nil {
			log.Fatalf("terminating with config error: %v", err)
			return nil
		}

		instance.BaseURL = &url.URL{
			Host:   instance.Domain,
			Scheme: instance.Scheme,
		}

		// Validate Auth0 input
		if err := instance.validateAuth0Config(); err != nil {
			log.Fatalf("terminating with config error: %v", err)
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

func (c *Config) validateAuth0Config() error {
	if len(instance.ClientSecret) > 3 && strings.Contains(instance.Domain, "auth0.com") {
		log.Println("validating Auth0 configuration")
		cl := client.NewAuth0Client(client.NewRestClient(c.BaseURL), c.ClientSecret, c.ClientID, c.Audience)

		if err := cl.Authorize(); err != nil {
			a := "a) the client_id, client_secret, domain and audience provided are correct."
			b := "b) the Auth0 app is allowed to get token via Client Credentials grant type."
			return fmt.Errorf("client check failed to authorize with Auth0: %v \nMake sure that: \n %s\n %s", err, a, b)
		}

		clients, err := cl.GetClients()
		if err != nil {
			a := "a) the app is authorized to access the management api."
			b := "b) the management api has been granted with 'read:client' scope."
			return fmt.Errorf("client check failed to get clients: %v \nMake sure that: \n %s\n %s", err, a, b)
		}

		var clientAPP client.Client
		for _, v := range *clients {
			if v.ClientID == c.ClientID {
				clientAPP = v
			}
		}

		isCallback := false
		for _, v := range clientAPP.Callbacks {
			if v == c.CallbackURL {
				isCallback = true
			}
		}

		if !isCallback {
			return errors.New("client check failed: callback url provided is not set for this client ID")
		}

		grants, err := cl.GetClientGrants()
		if err != nil {
			a := "a) the management api has been granted with 'read:grants' scope."
			return fmt.Errorf("client check failed to get clients grants: %v \nMake sure that: \n %s", err, a)
		}

		var grant client.Grant
		for _, v := range *grants {
			if v.ClientID == c.ClientID {
				grant = v
			}
		}

		if grant.Audience == c.Audience {
			return errors.New("client check failed: audience provide not found for this client id")
		}

	}

	return nil
}
