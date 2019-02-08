package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

const (
	pvtKEYPath = "./app.rsa"
	pubKEYPath = "./app.rsa.pub"
	pvtKEYType = "PRIVATE KEY"
	pubKEYType = "PUBLIC KEY"
	bitSize    = 2048
)

// Secret pointer to struct contains methods and fields to manage public and private keys.
type Secret struct {
	config      types.Config
	logger      types.Logger
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	signBytes   []byte
	verifyBytes []byte
}

var instance *Secret

// New returns singleton instance of secret.
func New(cfg types.Config, log types.Logger) (*Secret, error) {
	if instance == nil {
		instance = &Secret{config: cfg, logger: log}
		if cfg.PubKPath != "" && cfg.PvtKPath != "" {
			if err := instance.readPEMfromFile(instance.config.PubKPath, instance.config.PubKPath); err != nil {
				return nil, err
			}
		} else {
			if err := instance.generatePEM(pvtKEYPath, pubKEYPath); err != nil {
				return nil, err
			}
		}
		if err := instance.parsePrivateKey(); err != nil {
			return nil, err
		}
		if err := instance.parsePublicKey(); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

// GetPublicKeyPEM returns private key PEM.
func (k *Secret) GetPublicKeyPEM() []byte {
	return k.verifyBytes
}

// GetPrivateKeyPEM returns public key PEM.
func (k *Secret) GetPrivateKeyPEM() []byte {
	return k.signBytes
}

// GetPublicKey returns rsa public key object.
func (k *Secret) GetPublicKey() *rsa.PublicKey {
	return k.publicKey
}

// GetPrivateKey returns rsa private key object.
func (k *Secret) GetPrivateKey() *rsa.PrivateKey {
	return k.privateKey
}

func (k *Secret) parsePublicKey() error {
	key, err := jwt.ParseRSAPublicKeyFromPEM(k.verifyBytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse public key from PEM")
	}
	k.publicKey = key
	return nil
}

func (k *Secret) parsePrivateKey() error {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(k.signBytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse private key from PEM")
	}
	k.privateKey = key
	return nil
}

func (k *Secret) generatePEM(pubkPath string, pvtKeyPath string) error {
	// Private key
	var pvtkey *rsa.PrivateKey
	pvtkey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return errors.Wrap(err, "generating private key")
	}
	if err := pvtkey.Validate(); err != nil {
		return errors.Wrap(err, "validating private key")
	}

	pemBlock := pem.Block{
		Type:  pvtKEYType,
		Bytes: x509.MarshalPKCS1PrivateKey(pvtkey),
	}
	k.signBytes = pem.EncodeToMemory(&pemBlock)

	// Public key
	if pubKEYBytes, err := x509.MarshalPKIXPublicKey(&pvtkey.PublicKey); err != nil {
		return errors.Wrap(err, "error marshalling pub key")
	} else {
		pemBlockPub := pem.Block{
			Type:  pubKEYType,
			Bytes: pubKEYBytes,
		}
		k.verifyBytes = pem.EncodeToMemory(&pemBlockPub)
	}
	return nil
}

func (k *Secret) readPEMfromFile(pubkPath string, pvtKeyPath string) error {
	if vbyte, err := ioutil.ReadFile(k.config.PubKPath); err != nil {
		return errors.Wrap(err, "reading public key file")
	} else {
		k.verifyBytes = vbyte
	}

	if sbyte, err := ioutil.ReadFile(k.config.PvtKPath); err != nil {
		return errors.Wrap(err, "reading private key file")
	} else {
		k.signBytes = sbyte
	}
	return nil
}
