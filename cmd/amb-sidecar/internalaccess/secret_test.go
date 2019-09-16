package internalaccess

import (
	"testing"
)

func TestMakeSecret(t *testing.T) {
	s := GetInternalSecret()
	secret := s.Get()
	if len(secret) != 64 {
		t.Errorf("Wrong length for secret: %d", len(secret))
	}
	if ok := s.Compare(secret); ok != 1 {
		t.Errorf("secret does not match itself? got %d for '%s'", ok, secret)
	}
}
