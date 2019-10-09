package internalaccess

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type InternalSecret struct {
	secret string
}

var secret InternalSecret

func GetInternalSecret() (s *InternalSecret) {
	return &secret
}

func (s *InternalSecret) Get() string {
	return s.secret
}

func (s *InternalSecret) Compare(secret string) int {
	return subtle.ConstantTimeCompare([]byte(s.secret), []byte(secret))
}

func init() {
	secret.secret = MakeSecret()
}

// Load a file and hash it with SHA256 in order to create a shared secret.
func MakeSecret() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	h := sha256.New()
	_, _ = h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
