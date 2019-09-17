package types_test

import (
	"testing"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

type publicUrlChecker string

func publicUrlCheck(u string) publicUrlChecker {
	return publicUrlChecker(u)
}

func (c publicUrlChecker) make() types.PortalConfig {
	return types.PortalConfig{AmbassadorExternalURL: string(c)}
}

func (c publicUrlChecker) isOk(t *testing.T) {
	_, _, fatal := types.ValidatePortalConfig(c.make(), nil, nil)
	if len(fatal) != 0 {
		t.Errorf("Unexpected errors with %q: %v", c, fatal)
	}
}

func (c publicUrlChecker) isBad(t *testing.T) {
	_, _, fatal := types.ValidatePortalConfig(c.make(), nil, nil)
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
