package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"github.com/datawire/ambassador-oauth/cmd/ambassador-oauth/config"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

const (
	pvtKEYPath = "./app.rsa"
	pubKEYPath = "./app.rsa.pub"
	pvtKEYType = "PRIVATE KEY"
	pubKEYType = "PUBLIC KEY"
	bitSize    = 2048
)

// Secret TODO(gsagula): comment
type Secret struct {
	config      *config.Config
	logger      *logrus.Logger
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	signBytes   []byte
	verifyBytes []byte
}

var instance *Secret

// New TODO(gsagula): comment
func New(cfg *config.Config, log *logrus.Logger) *Secret {
	if instance == nil {
		instance = &Secret{config: cfg, logger: log}
		if cfg.PubKPath != "" && cfg.PvtKPath != "" {
			instance.readPEMfromFile(instance.config.PubKPath, instance.config.PubKPath)
		} else {
			instance.generatePEM(pvtKEYPath, pubKEYPath)
		}
		instance.parsePrivateKey()
		instance.parsePublicKey()
	}

	return instance
}

// GetPublicKeyPEM TODO(gsagula): comment
func (k *Secret) GetPublicKeyPEM() []byte {
	return k.verifyBytes
}

// GetPrivateKeyPEM TODO(gsagula): comment
func (k *Secret) GetPrivateKeyPEM() []byte {
	return k.signBytes
}

// GetPublicKey TODO(gsagula): comment
func (k *Secret) GetPublicKey() *rsa.PublicKey {
	return k.publicKey
}

// GetPrivateKey TODO(gsagula): comment
func (k *Secret) GetPrivateKey() *rsa.PrivateKey {
	return k.privateKey
}

func (k *Secret) parsePublicKey() {
	key, err := jwt.ParseRSAPublicKeyFromPEM(k.verifyBytes)
	if err != nil {
		k.logger.Fatalf("failed to parse public key from PEM: %v", err)
	}
	k.publicKey = key
}

func (k *Secret) parsePrivateKey() {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(k.signBytes)
	if err != nil {
		k.logger.Fatalf("failed to parse private key from PEM: %v", err)
	}
	k.privateKey = key
}

func (k *Secret) generatePEM(pubkPath string, pvtKeyPath string) {
	// Private key
	var pvtkey *rsa.PrivateKey
	pvtkey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		k.logger.Fatalf("generating private key: %v", err)
	}
	if err := pvtkey.Validate(); err != nil {
		k.logger.Fatalf("validating private key: %v", err)
	}

	pemBlock := pem.Block{
		Type:  pvtKEYType,
		Bytes: x509.MarshalPKCS1PrivateKey(pvtkey),
	}
	k.signBytes = pem.EncodeToMemory(&pemBlock)

	// Public key
	if pubKEYBytes, err := x509.MarshalPKIXPublicKey(&pvtkey.PublicKey); err != nil {
		k.logger.Fatalf("error marshalling pub key: %v", err)
	} else {
		pemBlockPub := pem.Block{
			Type:  pubKEYType,
			Bytes: pubKEYBytes,
		}
		k.verifyBytes = pem.EncodeToMemory(&pemBlockPub)
	}
}

func (k *Secret) readPEMfromFile(pubkPath string, pvtKeyPath string) {
	if vbyte, err := ioutil.ReadFile(k.config.PubKPath); err != nil {
		k.logger.Fatalf("reading public key file: %v", err)
	} else {
		k.verifyBytes = vbyte
	}

	if sbyte, err := ioutil.ReadFile(k.config.PvtKPath); err != nil {
		k.logger.Fatalf("reading private key file: %v", err)
	} else {
		k.signBytes = sbyte
	}
}
