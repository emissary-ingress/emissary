package types

import (
	"testing"

	portal "github.com/datawire/apro/cmd/dev-portal-server/server"
)

type publicUrlChecker string

func publicUrlCheck(u string) publicUrlChecker {
	return publicUrlChecker(u)
}

func (c publicUrlChecker) make() portal.ServerConfig {
	return portal.ServerConfig{PublicURL: string(c)}
}

func (c publicUrlChecker) isOk(t *testing.T) {
	_, _, fatal := validatePortalConfig(c.make(), nil, nil)
	if len(fatal) != 0 {
		t.Errorf("Unexpected errors with %q: %v", c, fatal)
	}
}

func (c publicUrlChecker) isBad(t *testing.T) {
	_, _, fatal := validatePortalConfig(c.make(), nil, nil)
	if len(fatal) == 0 {
		t.Errorf("Unexpected success with %q", c)
	}
	t.Logf("%q is bad because %v", c, fatal)
}

func TestValidatePortalConfigPublicUrl(t *testing.T) {
	publicUrlCheck("http://ambassador").isOk(t)
	publicUrlCheck("https://ambassador").isOk(t)
	publicUrlCheck("https://ambassador/").isOk(t)
	publicUrlCheck("https://ambassador:80").isOk(t)
	publicUrlCheck("https://ambassador:80/").isOk(t)

	publicUrlCheck("ambassador").isBad(t)
	publicUrlCheck("ambassador:80").isBad(t)
	publicUrlCheck("ambassador:80/").isBad(t)
}
