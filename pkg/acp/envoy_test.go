package acp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/datawire/ambassador/pkg/acp"
)

type fakeStatsMode string

const (
	Happy   = fakeStatsMode("happy")
	Error   = fakeStatsMode("error")
	Failure = fakeStatsMode("failure")
)

type fakeStats struct {
	// What mode are we using right now?
	mode fakeStatsMode
}

func (f *fakeStats) setMode(mode fakeStatsMode) {
	f.mode = mode
}

func (f *fakeStats) fetchStats(ctx context.Context) (*acp.EnvoyFetcherResponse, error) {
	var resp *acp.EnvoyFetcherResponse
	var err error

	switch f.mode {
	case Happy:
		resp = &acp.EnvoyFetcherResponse{
			StatusCode: 200,
			Text:       []byte("Lovely stats yay"),
		}
		err = nil

	case Error:
		resp = nil
		err = fmt.Errorf("fakeStatsError always errors")

	case Failure:
		resp = &acp.EnvoyFetcherResponse{
			StatusCode: 503,
			Text:       []byte("No stats here"),
		}
		err = nil
	}

	return resp, err
}

type envoyMetadata struct {
	t  *testing.T
	f  *fakeStats
	ew *acp.EnvoyWatcher
}

func (m *envoyMetadata) check(seq int, alive bool) {
	if m.ew.IsAlive() != alive {
		m.t.Errorf("%d: EnvoyWatcher.IsAlive %t, wanted %t", seq, m.ew.IsAlive(), alive)
	}

	if m.ew.IsReady() != alive {
		m.t.Errorf("%d: EnvoyWatcher.IsReady %t, wanted %t (same as alive)", seq, m.ew.IsAlive(), alive)
	}
}

func newEnvoyMetadata(t *testing.T, statsMode fakeStatsMode) *envoyMetadata {
	f := &fakeStats{mode: statsMode}

	ew := acp.NewEnvoyWatcher()
	ew.SetFetchStats(f.fetchStats)

	if ew == nil {
		t.Error("New EnvoyWatcher is nil?")
	}

	return &envoyMetadata{t: t, f: f, ew: ew}
}

func TestEnvoyHappyPath(t *testing.T) {
	m := newEnvoyMetadata(t, Happy)
	m.check(0, false)

	// Fetch stats.
	m.ew.FetchEnvoyStats(context.Background())
	m.check(1, true)
}

func TestEnvoyError(t *testing.T) {
	m := newEnvoyMetadata(t, Error)
	m.check(0, false)

	// Fetch stats.
	m.ew.FetchEnvoyStats(context.Background())
	m.check(1, false)
}

func TestEnvoy503(t *testing.T) {
	m := newEnvoyMetadata(t, Failure)
	m.check(0, false)

	// Fetch stats.
	m.ew.FetchEnvoyStats(context.Background())
	m.check(1, false)
}

func TestEnvoySadToHappy(t *testing.T) {
	m := newEnvoyMetadata(t, Error)
	m.check(0, false)

	// Fetch stats.
	m.ew.FetchEnvoyStats(context.Background())
	m.check(1, false)

	// Switch to the happy path.
	m.f.setMode(Happy)
	m.ew.FetchEnvoyStats(context.Background())
	m.check(2, true)
}
