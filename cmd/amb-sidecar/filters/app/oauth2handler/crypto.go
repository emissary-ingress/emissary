package oauth2handler

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
)

// TODO(lukeshu): Someone should yell at me for my choices of
// algorithms.

func (c *OAuth2Filter) cryptoSign(body []byte) (signature []byte, err error) {
	sum := sha256.Sum256(body)
	return rsa.SignPSS(rand.Reader, c.PrivateKey, crypto.SHA256, sum[:], nil)
}

func (c *OAuth2Filter) cryptoVerify(body, signature []byte) error {
	sum := sha256.Sum256(body)
	return rsa.VerifyPSS(c.PublicKey, crypto.SHA256, sum[:], signature, nil)
}

func (c *OAuth2Filter) cryptoEncrypt(cleartext, label []byte) (ciphertext []byte, err error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, c.PublicKey, cleartext, label)
}

func (c *OAuth2Filter) cryptoDecrypt(ciphertext, label []byte) (cleartext []byte, err error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, c.PrivateKey, ciphertext, label)
}

func (c *OAuth2Filter) cryptoSignAndEncrypt(cleartextBody, label []byte) (ciphertext []byte, err error) {
	signature, err := c.cryptoSign(cleartextBody)
	if err != nil {
		return nil, err
	}
	cleartext := make([]byte, 8+len(cleartextBody)+len(signature))
	binary.BigEndian.PutUint64(cleartext[0:], uint64(len(cleartextBody)))
	copy(cleartext[8:], cleartextBody)
	copy(cleartext[8+len(cleartextBody):], signature)
	return c.cryptoEncrypt(cleartext, label)
}

func (c *OAuth2Filter) cryptoDecryptAndVerify(ciphertext, label []byte) (cleartext []byte, err error) {
	cleartext, err = c.cryptoDecrypt(ciphertext, label)
	if err != nil {
		return nil, err
	}
	cleartextBodyLen := binary.BigEndian.Uint64(cleartext)
	cleartextBody := cleartext[8:8+cleartextBodyLen]
	signature := cleartext[8+cleartextBodyLen:]
	if err = c.cryptoVerify(cleartextBody, signature); err != nil {
		return nil, err
	}
	return cleartextBody, nil
}
