package limiter

import (
	"golang.org/x/crypto/argon2"

	"github.com/o1egl/paseto"
	"github.com/pkg/errors"

	"github.com/datawire/apro/lib/licensekeys"
)

// DeriveKeyFromKDF is KDF (key derivation function) that will derive an
// encryption key from a normal set of strings.
//
// It does this using argon2i which is side-channel resistant, and is implemented
// in golang proper.
//
// The only use for this currently is deriving an encryption key for the
// limiter encryption. The limiter encryption can be figured out by manually
// debugging the code, but this function would be secure if a user didn't
// have the encryption key just sitting in memory (or the field which we
// derived from).
// Basically: If you wanna copy this and use it for an actual KDF you can :)
// it will meet your security requirements, but won't fix a crappy situation.
func DeriveKeyFromKDF(toDerive string, salt string) []byte {
	// Values here:
	//   - toDerive: the string to derive the key from.
	//   - salt: a unique 32 byte value.
	//   - time=3 (comes as the recommendation from the RFC).
	//   - memory=32mb (comes as the recommendation from the RFC).
	//   - threads=4 (comes as the recommendation from the RFC).
	//   - keyLen = the resulting key length in bytes.
	return argon2.Key([]byte(toDerive), []byte(salt), 3, 32*1024, 4, 32)
}

// LimitCrypto manages a series of encryptions for limiters.
type LimitCrypto struct {
	// underlyingKey is the key passed to paseto for encryption.
	// derived from customer info.
	underlyingKey []byte
	// tokenCreator is used to actually create the tokens.
	tokenCreator *paseto.V2
}

// NewLimitCrypto creates a new limiter crypto engine based off of a license key claims.
func NewLimitCrypto(claims *licensekeys.LicenseClaimsLatest) *LimitCrypto {
	keyToUse := "unknown@unknown.com"
	if claims.CustomerID != "" {
		keyToUse = claims.CustomerID
	}
	saltToUse := "unknown"
	if claims.CustomerEmail != "" {
		saltToUse = claims.CustomerEmail
	}

	return &LimitCrypto{
		underlyingKey: DeriveKeyFromKDF(keyToUse, saltToUse),
		tokenCreator:  paseto.NewV2(),
	}
}

// EncryptString encrypts a particular string, for passing back into limit crypto later.
func (this *LimitCrypto) EncryptString(toEncrypt string) (string, error) {
	jsonToken := paseto.JSONToken{}
	jsonToken.Set("limiter_encrypted_value", toEncrypt)

	return this.tokenCreator.Encrypt(this.underlyingKey, jsonToken, "ambassador")
}

// DecryptString decrypts a particular string that was encrypted with limiter EncryptString.
func (this *LimitCrypto) DecryptString(toDecrypt string) (string, error) {
	var decryptedToken paseto.JSONToken
	footer := "ambassador"
	err := this.tokenCreator.Decrypt(toDecrypt, this.underlyingKey, &decryptedToken, &footer)
	if err != nil {
		return "", err
	}
	if footer != "ambassador" {
		return "", errors.New("Bad footer in encrypted data")
	}

	return decryptedToken.Get("limiter_encrypted_value"), nil
}
