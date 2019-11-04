package limiter

import (
	"time"
	"testing"

	. "github.com/onsi/gomega"
	jwt "github.com/dgrijalva/jwt-go"

	"github.com/datawire/apro/lib/licensekeys"
)

func TestCryptoRoundTrip(t *testing.T) {
	g := NewGomegaWithT(t)

	now := time.Now()
	expiresAt := now.Add(time.Duration(1) * 24 * time.Hour)

	claims := &licensekeys.LicenseClaimsV2{
		LicenseKeyVersion: "v2",
		CustomerID:        "datawire",
		CustomerEmail:     "fake@datawire.com",
		EnabledFeatures:   make([]licensekeys.Feature, 0),
		EnforcedLimits:    make([]licensekeys.LimitValue, 0),
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			NotBefore: now.Unix(),
			ExpiresAt: expiresAt.Unix(),
		},
	}
	lc := NewLimitCrypto(claims)

	rawStr := "10"
	encrypted, err := lc.EncryptString(rawStr)
	if err != nil {
		t.Fatalf("Shouldn't error out encrypting string: %s", err.Error())
	}
	decrypted, err := lc.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("Shouldn't error out decrypting string: %s", err.Error())
	}
	g.Expect(decrypted).To(Equal(rawStr))
}
