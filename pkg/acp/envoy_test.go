package acp_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/datawire/ambassador/v2/pkg/acp"
	"github.com/datawire/dlib/dlog"
)

type fakeReadyMode string

const (
	Happy   = fakeReadyMode("happy")
	Error   = fakeReadyMode("error")
	Failure = fakeReadyMode("failure")
)

type fakeReady struct {
	// What mode are we using right now?
	mode fakeReadyMode
}

func (f *fakeReady) setMode(mode fakeReadyMode) {
	f.mode = mode
}

func (f *fakeReady) readyCheck(ctx context.Context) (*acp.EnvoyFetcherResponse, error) {
	var resp *acp.EnvoyFetcherResponse
	var err error

	switch f.mode {
	case Happy:
		resp = &acp.EnvoyFetcherResponse{
			StatusCode: 200,
			Text:       []byte("Ready"),
		}
		err = nil

	case Error:
		resp = nil
		err = fmt.Errorf("fakeReady Error always errors")

	case Failure:
		resp = &acp.EnvoyFetcherResponse{
			StatusCode: 503,
			Text:       []byte("Not ready"),
		}
		err = nil
	}

	return resp, err
}

type envoyMetadata struct {
	t  *testing.T
	f  *fakeReady
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

func newEnvoyMetadata(t *testing.T, readyMode fakeReadyMode) *envoyMetadata {
	f := &fakeReady{mode: readyMode}

	ew := acp.NewEnvoyWatcher()
	ew.SetReadyCheck(f.readyCheck)

	if ew == nil {
		t.Error("New EnvoyWatcher is nil?")
	}

	return &envoyMetadata{t: t, f: f, ew: ew}
}

func TestEnvoyHappyPath(t *testing.T) {
	m := newEnvoyMetadata(t, Happy)
	m.check(0, false)

	// Fetch readiness.
	m.ew.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(1, true)
}

func TestEnvoyError(t *testing.T) {
	m := newEnvoyMetadata(t, Error)
	m.check(0, false)

	// Fetch readiness.
	m.ew.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(1, false)
}

func TestEnvoy503(t *testing.T) {
	m := newEnvoyMetadata(t, Failure)
	m.check(0, false)

	// Fetch readiness.
	m.ew.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(1, false)
}

func TestEnvoySadToHappy(t *testing.T) {
	m := newEnvoyMetadata(t, Error)
	m.check(0, false)

	// Fetch readiness.
	m.ew.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(1, false)

	// Switch to the happy path.
	m.f.setMode(Happy)
	m.ew.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(2, true)
}
