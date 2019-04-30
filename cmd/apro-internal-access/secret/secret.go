package secret

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

// Load a file and hash it with SHA256 in order to create a shared secret.
func LoadSecret(sharedSecretPath string) string {
	f, err := os.Open(sharedSecretPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
