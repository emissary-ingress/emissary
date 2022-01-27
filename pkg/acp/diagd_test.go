package acp_test

import (
	"testing"
	"time"

	"github.com/datawire/ambassador/v2/pkg/acp"
	"github.com/datawire/dlib/dtime"
)

type diagdMetadata struct {
	t  *testing.T
	ft *dtime.FakeTime
	dw *acp.DiagdWatcher
}

func (m *diagdMetadata) check(seq int, clock int, alive bool, ready bool) {
	if m.ft.TimeSinceBoot() != time.Duration(clock)*time.Second {
		m.t.Errorf("%d: fakeTime.TimeSinceBoot should be %ds, not %v", seq, clock, m.ft.TimeSinceBoot())
	}

	if m.dw.IsAlive() != alive {
		m.t.Errorf("%d: DiagdWatcher.IsAlive %t, wanted %t", seq, m.dw.IsAlive(), alive)
	}

	if m.dw.IsReady() != ready {
		m.t.Errorf("%d: DiagdWatcher.IsReady %t, wanted %t", seq, m.dw.IsReady(), ready)
	}
}

func (m *diagdMetadata) stepSec(step int) {
	m.ft.StepSec(step)
}

func newDiagdMetadata(t *testing.T) *diagdMetadata {
	ft := dtime.NewFakeTime()

	dw := acp.NewDiagdWatcher()
	dw.SetFetchTime(ft.Now)

	if dw == nil {
		t.Error("New DiagdWatcher is nil?")
	}

	bootGraceLength := dw.GraceEnd.Sub(ft.BootTime())
	tenMinutes := time.Minute * 10

	if bootGraceLength != tenMinutes {
		t.Errorf("GraceEnd is %v after bootTime, not %v", bootGraceLength, tenMinutes)
	}

	return &diagdMetadata{t: t, ft: ft, dw: dw}
}

func TestDiagdHappyPath(t *testing.T) {
	m := newDiagdMetadata(t)
	m.check(0, 0, true, false)

	// Advance the clock 10s.
	m.stepSec(10)

	m.check(1, 10, true, false)

	// Send a snapshot. We should still be alive but not ready.
	m.dw.NoteSnapshotSent()

	m.check(2, 10, true, false)

	// Advance another 30s.
	m.stepSec(30)

	// Mark the snapshot processed. We should now be ready.
	m.dw.NoteSnapshotProcessed()
	m.check(3, 40, true, true)
}

func TestDiagdVerySlowlySent(t *testing.T) {
	m := newDiagdMetadata(t)
	m.check(0, 0, true, false)

	// Don't send a snapshot, but advance the clock into the boot grace period.
	// We should still be alive.
	m.stepSec(300)
	m.check(1, 300, true, false)

	// Advance the clock to just before the end of the boot grace period.
	// We should still be alive.
	m.stepSec(299)
	m.check(1, 599, true, false)

	// Advance the clock to just past the end of the boot grace period.
	// We should no longer be alive.
	m.stepSec(1)
	m.check(2, 600, false, false)

	// Send a snapshot. We should stay dead.
	m.stepSec(1)
	m.dw.NoteSnapshotSent()
	m.check(3, 601, false, false)

	// Not very useful, but if we mark the snapshot processed, we should snap
	// to alive and ready.
	m.stepSec(1)
	m.dw.NoteSnapshotProcessed()
	m.check(2, 602, true, true)
}

func TestDiagdNeverProcessed(t *testing.T) {
	m := newDiagdMetadata(t)
	m.check(0, 0, true, false)

	// Advance the clock 10s.
	m.stepSec(10)

	m.check(1, 10, true, false)

	// Send a snapshot. We should still be alive but not ready.
	m.dw.NoteSnapshotSent()
	m.check(2, 10, true, false)

	// Advance another 9m (540s).
	m.stepSec(540)
	m.check(3, 550, true, false)

	// Advance the clock another minute. We're now past the ten-minute grace period.
	m.stepSec(60)
	m.check(4, 610, false, false)
}

func TestDiagdSlowlyProcessed(t *testing.T) {
	m := newDiagdMetadata(t)
	m.check(0, 0, true, false)

	// Advance the clock 10s.
	m.stepSec(10)
	m.check(1, 10, true, false)

	// Send a snapshot. We should still be alive but not ready.
	m.dw.NoteSnapshotSent()
	m.check(2, 10, true, false)

	// Mark the snapshot processed after another 10s. This should take
	// us to alive and ready.
	m.stepSec(10)
	m.dw.NoteSnapshotProcessed()
	m.check(3, 20, true, true)

	// Send another snapshot after 40s (to take us to an even minute).
	// Still alive and ready.
	m.stepSec(40)
	m.dw.NoteSnapshotSent()
	m.check(4, 60, true, true)

	// Don't process that snapshot. 9m59s after sending, we should
	// still be alive and ready.
	m.stepSec(599)
	m.check(5, 659, true, true)

	// At 10m after sending, we should become not alive and not ready.
	m.stepSec(1)
	m.check(6, 660, false, false)
}
