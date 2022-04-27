package acp_test

import (
	"testing"
	"time"

	"github.com/datawire/dlib/dlog"
	"github.com/datawire/dlib/dtime"
	"github.com/emissary-ingress/emissary/v3/pkg/acp"
)

type awMetadata struct {
	t  *testing.T
	ft *dtime.FakeTime
	aw *acp.AmbassadorWatcher
}

func (m *awMetadata) check(seq int, clock int, alive bool, ready bool) {
	if m.ft.TimeSinceBoot() != time.Duration(clock)*time.Second {
		m.t.Errorf("%d: FakeTime.TimeSinceBoot should be %ds, not %v", seq, clock, m.ft.TimeSinceBoot())
	}

	if m.aw.IsAlive() != alive {
		m.t.Errorf("%d: DiagdWatcher.IsAlive %t, wanted %t", seq, m.aw.IsAlive(), alive)
	}

	if m.aw.IsReady() != ready {
		m.t.Errorf("%d: DiagdWatcher.IsReady %t, wanted %t", seq, m.aw.IsReady(), ready)
	}
}

func (m *awMetadata) stepSec(step int) {
	m.ft.StepSec(step)
}

func newAWMetadata(t *testing.T) *awMetadata {
	ft := dtime.NewFakeTime()
	f := &fakeReady{mode: Happy}

	dw := acp.NewDiagdWatcher()
	dw.SetFetchTime(ft.Now)

	if dw == nil {
		t.Error("New DiagdWatcher is nil?")
	}

	ew := acp.NewEnvoyWatcher()
	ew.SetReadyCheck(f.readyCheck)

	if ew == nil {
		t.Error("New EnvoyWatcher is nil?")
	}

	aw := acp.NewAmbassadorWatcher(ew, dw)
	aw.SetFetchTime(ft.Now)

	return &awMetadata{t: t, ft: ft, aw: aw}
}

func TestAmbassadorHappyPath(t *testing.T) {
	m := newAWMetadata(t)
	m.check(0, 0, true, false)

	// Advance the clock 10s.
	m.stepSec(10)
	m.check(1, 10, true, false)

	// Send a snapshot.
	m.stepSec(10)
	m.aw.NoteSnapshotSent()
	m.check(2, 20, true, false)

	// Mark the snapshot processed.
	m.stepSec(10)
	m.aw.NoteSnapshotProcessed()
	m.check(3, 30, true, false)

	// Fetch readiness.
	m.stepSec(10)
	m.aw.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(4, 40, true, true)

	// Make sure it stays happy.
	m.stepSec(10)
	m.check(5, 50, true, true)
}

func TestAmbassadorUnrealisticallyHappy(t *testing.T) {
	m := newAWMetadata(t)
	m.check(0, 0, true, false)

	// Advance the clock 10s.
	m.stepSec(10)
	m.check(1, 10, true, false)

	// Send a snapshot, mark it processed, and have Envoy come up all
	// in the same instant. This is _highly unlikely_ but WTF, let's
	// try it. We expect to see alive and ready here.
	m.stepSec(10)
	m.aw.NoteSnapshotSent()
	m.aw.NoteSnapshotProcessed()
	m.aw.FetchEnvoyReady(dlog.NewTestContext(t, false))
	m.check(2, 20, true, true)

	// Make sure it stays happy.
	m.stepSec(10)
	m.check(3, 30, true, true)
}

func TestAmbassadorNoSnapshots(t *testing.T) {
	m := newAWMetadata(t)
	m.check(0, 0, true, false)

	// Advance the clock halfway through the diagd boot grace period.
	// We should see alive but not ready.
	m.stepSec(300)
	m.check(1, 300, true, false)

	// Advance nearly to the end of diagd boot grace period.
	// We should see alive but not ready.
	m.stepSec(299)
	m.check(2, 599, true, false)

	// Advance to the end of diagd boot grace period.
	// We should see neither alive nor ready.
	m.stepSec(1)
	m.check(3, 600, false, false)

	// Nothing should change after another minute.
	m.stepSec(60)
	m.check(4, 660, false, false)
}
