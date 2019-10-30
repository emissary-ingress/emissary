// initiator.go deals with initiating conversations with the ACME
// server.

package acmeclient

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"time"

	"github.com/mediocregopher/radix.v2/pool"

	"github.com/go-acme/lego/v3/certcrypto"
	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/registration"
)

// acmeUser implements registration.User
type acmeUser struct {
	Email        string
	Registration *registration.Resource
	PrivateKey   crypto.PrivateKey
}

func (u *acmeUser) GetEmail() string {
	return u.Email
}
func (u acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.PrivateKey
}

func newClient(httpClient *http.Client, ca string, user registration.User) (*lego.Client, error) {
	config := &lego.Config{
		CADirURL:   ca,
		User:       user,
		HTTPClient: httpClient,
		// this mimics lego.NewConfig()
		Certificate: lego.CertificateConfig{
			KeyType: certcrypto.RSA2048,
			Timeout: 30 * time.Second,
		},
	}

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func registerUser(httpClient *http.Client, ca, email string, privateKey crypto.PrivateKey) (registration.User, error) {
	var err error
	if privateKey == nil {
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
	}

	user := &acmeUser{
		Email:        email,
		PrivateKey:   privateKey,
		Registration: nil, // we get this from client.Registration.Register() below
	}

	client, err := newClient(httpClient, ca, user)
	if err != nil {
		return nil, err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	return user, nil
}

// challengeProvider implements challenge.Provider
type challengeProvider struct {
	redisPool *pool.Pool
}

func (p *challengeProvider) Present(domain, token, keyAuth string) error {
	var timeout time.Duration = 10 * time.Minute

	redisClient, err := p.redisPool.Get()
	if err != nil {
		return err
	}
	defer p.redisPool.Put(redisClient)

	if err := redisClient.Cmd("SET", "acme-challenge:"+token, keyAuth).Err; err != nil {
		return err
	}
	if err := redisClient.Cmd("EXPIRE", "acme-challenge:"+token, int64(timeout.Seconds())).Err; err != nil {
		return err
	}

	return nil
}

func (p *challengeProvider) CleanUp(domain, token, keyAuth string) error {
	redisClient, err := p.redisPool.Get()
	if err != nil {
		return err
	}
	defer p.redisPool.Put(redisClient)

	return redisClient.Cmd("DEL", "acme-challenge:"+token).Err
}

func obtainCertificate(httpClient *http.Client, redisPool *pool.Pool, ca string, user registration.User, subjects []string) (*certificate.Resource, error) {
	client, err := newClient(httpClient, ca, user)
	if err != nil {
		return nil, err
	}

	err = client.Challenge.SetHTTP01Provider(&challengeProvider{redisPool})
	if err != nil {
		return nil, err
	}

	certResource, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: subjects,
		Bundle:  true,
	})
	if err != nil {
		return nil, err
	}

	return certResource, nil
}
