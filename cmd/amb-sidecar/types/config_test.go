package types_test

import (
	"os"
	"testing"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

func check(u string) (value types.PortalConfig, warn []error, fatal []error) {
	os.Setenv("AMBASSADOR_ADMIN_URL", "http://127.0.0.1:8877/")
	os.Setenv("AMBASSADOR_INTERNAL_URL", "https://127.0.0.1:8443/")
	os.Setenv("AMBASSADOR_URL", u)
	os.Setenv("POLL_EVERY_SECS", "60")
	os.Setenv("APRO_DEVPORTAL_CONTENT_URL", "https://github.com/datawire/devportal-content")
	return types.PortalConfigFromEnv(nil, nil)
}

type assertGen struct {
	t *testing.T
}

func Assert(t *testing.T) assertGen {
	return assertGen{
		t: t,
	}
}

type assertPortalConfig struct {
	t     *testing.T
	value types.PortalConfig
	warn  []error
	fatal []error
}

func (g assertGen) PortalConfig(value types.PortalConfig, warn []error, fatal []error) assertPortalConfig {
	return assertPortalConfig{
		t:     g.t,
		value: value,
		warn:  warn,
		fatal: fatal,
	}
}

func (a assertPortalConfig) HasNWarnings(n int) assertPortalConfig {
	if len(a.warn) != n {
		a.t.Errorf("asserted %d warnings; got %d", n, len(a.warn))
	}
	return a
}

func (a assertPortalConfig) HasNFatals(n int) assertPortalConfig {
	if len(a.fatal) != n {
		a.t.Errorf("asserted %d fatals; got %d", n, len(a.fatal))
	}
	return a
}

func TestValidatePortalConfigPublicUrl(t *testing.T) {
	Assert(t).PortalConfig(check("http://ambassador")).HasNWarnings(0).HasNFatals(0)
	Assert(t).PortalConfig(check("https://ambassador")).HasNWarnings(0).HasNFatals(0)
	Assert(t).PortalConfig(check("https://ambassador/")).HasNWarnings(0).HasNFatals(0)
	Assert(t).PortalConfig(check("https://ambassador:80")).HasNWarnings(0).HasNFatals(0)
	Assert(t).PortalConfig(check("https://ambassador:80/")).HasNWarnings(0).HasNFatals(0)

	// these should fall back to the default value; have a warning but not a fatal
	Assert(t).PortalConfig(check("ambassador")).HasNWarnings(1).HasNFatals(0)
	Assert(t).PortalConfig(check("ambassador:80")).HasNWarnings(1).HasNFatals(0)
	Assert(t).PortalConfig(check("ambassador:80/")).HasNWarnings(1).HasNFatals(0)
}
