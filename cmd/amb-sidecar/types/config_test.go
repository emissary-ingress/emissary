package types_test

import (
	"os"
	"testing"

	"github.com/datawire/apro/cmd/amb-sidecar/types"
)

type assertGen struct {
	t *testing.T
}

func Assert(t *testing.T) assertGen {
	return assertGen{
		t: t,
	}
}

type assertConfig struct {
	t     *testing.T
	value types.Config
	warn  []error
	fatal []error
}

func (g assertGen) Config(value types.Config, warn []error, fatal []error) assertConfig {
	return assertConfig{
		t:     g.t,
		value: value,
		warn:  warn,
		fatal: fatal,
	}
}

func (a assertConfig) HasNWarnings(n int) assertConfig {
	a.t.Helper()
	if len(a.warn) != n {
		a.t.Errorf("asserted %d warnings; got %d", n, len(a.warn))
		a.t.Logf("warnings: %v", a.warn)
	}
	return a
}

func (a assertConfig) HasNFatals(n int) assertConfig {
	a.t.Helper()
	if len(a.fatal) != n {
		a.t.Errorf("asserted %d fatals; got %d", n, len(a.fatal))
		a.t.Logf("fatals: %v", a.fatal)
	}
	return a
}

func TestAmbassadorExternalURLValidation(t *testing.T) {
	check := func(u string) (value types.Config, warn []error, fatal []error) {
		os.Clearenv()
		os.Setenv("REDIS_SOCKET_TYPE", "unix")
		os.Setenv("REDIS_URL", "/run/redis.sock")
		os.Setenv("AMBASSADOR_URL", u)
		return types.ConfigFromEnv()
	}

	Assert(t).Config(check("http://ambassador")).HasNWarnings(0).HasNFatals(0)
	Assert(t).Config(check("https://ambassador")).HasNWarnings(0).HasNFatals(0)
	Assert(t).Config(check("https://ambassador/")).HasNWarnings(0).HasNFatals(0)
	Assert(t).Config(check("https://ambassador:80")).HasNWarnings(0).HasNFatals(0)
	Assert(t).Config(check("https://ambassador:80/")).HasNWarnings(0).HasNFatals(0)

	// these should fall back to the default value; have a warning but not a fatal
	Assert(t).Config(check("ambassador")).HasNWarnings(1).HasNFatals(0)
	Assert(t).Config(check("ambassador:80")).HasNWarnings(1).HasNFatals(0)
	Assert(t).Config(check("ambassador:80/")).HasNWarnings(1).HasNFatals(0)
}
