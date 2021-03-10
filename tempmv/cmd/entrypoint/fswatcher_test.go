package entrypoint_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/datawire/ambassador/cmd/entrypoint"
	"github.com/datawire/dlib/dlog"
)

type fswMetadata struct {
	t            *testing.T
	fsw          *entrypoint.FSWatcher
	dir          string
	bootstrapped map[string]bool
	updates      map[string]int
	deletes      map[string]int
	errorCount   int
}

func newMetadata(t *testing.T) (*fswMetadata, error) {
	m := &fswMetadata{t: t}
	m.bootstrapped = make(map[string]bool)
	m.updates = make(map[string]int)
	m.deletes = make(map[string]int)

	var err error

	m.dir, err = ioutil.TempDir("", "fswatcher_test")

	if err != nil {
		t.Errorf("could not create tempdir: %s", err)
		return nil, err
	}

	m.fsw, err = entrypoint.NewFSWatcher(context.TODO())

	if err != nil {
		t.Errorf("could not instantiate FSWatcher: %s", err)
		return nil, err
	}

	m.fsw.SetErrorHandler(m.errorHandler)

	return m, nil
}

func (m *fswMetadata) done() {
	// You would think that a call to os.RemoveAll() would suffice
	// here, but nope. Turns out that on MacOS, at least, that won't
	// guarantee that we get events for deleting all the files in the
	// directory before the directory goes, and the test wants to see
	// all the files get deleted. Sigh. So. Do it by hand.

	files, err := ioutil.ReadDir(m.dir)

	if err != nil {
		dlog.Errorf(context.TODO(), "m.done: couldn't scan %s: %s", m.dir, err)
		return
	}

	for _, file := range files {
		path := filepath.Join(m.dir, file.Name())

		err = os.Remove(path)

		if err != nil {
			dlog.Errorf(context.TODO(), "m.done: couldn't remove %s: %s", path, err)
		}
	}

	// Sleep to make sure the file-deletion events get handled.
	time.Sleep(250 * time.Millisecond)

	// After scrapping the files, remove the directory too...
	os.Remove(m.dir)

	// ...and sleep once more to make sure the event for the directory
	// deletion makes it through.
	time.Sleep(250 * time.Millisecond)
}

// Error handler: just count errors received.
func (m *fswMetadata) errorHandler(ctx context.Context, err error) {
	dlog.Infof(ctx, "errorHandler: got %s", err)
	m.errorCount++
	dlog.Infof(ctx, "errorHandler: errorCount now %d", m.errorCount)
}

// Event handler: separately keep track of bootstrapped, updated, and deleted
// for each distinct basename we see.
func (m *fswMetadata) eventHandler(ctx context.Context, event entrypoint.FSWEvent) {
	dir := filepath.Dir(event.Path)
	base := filepath.Base(event.Path)

	bstr := ""
	if event.Bootstrap {
		bstr = "B|"
	}

	opStr := fmt.Sprintf("%s %s%s", event.Time, bstr, event.Op)

	dlog.Infof(context.TODO(), "eventHandler %s %s (dir %s)", opStr, base, dir)

	if dir != m.dir {
		m.t.Errorf("eventHandler: event for %s arrived, but we're watching %s", event.Path, m.dir)
		return
	}

	if event.Bootstrap {
		// Handle bootstrap events, which cannot be deletes.
		if event.Op == entrypoint.FSWDelete {
			m.t.Errorf("eventHandler: impossible bootstrap delete of %s arrived", event.Path)
			return
		}

		// Not a delete, so remember that this was a bootstrapped file.
		m.bootstrapped[base] = true
	}

	// Next, count updates and deletes.
	which := m.updates

	if event.Op == entrypoint.FSWDelete {
		which = m.deletes
	}

	count, ok := which[base]

	if ok {
		which[base] = count + 1
	} else {
		which[base] = 1
	}
}

// Make sure that per-file stats match what we expect.
func (m *fswMetadata) check(key string, wantedBootstrap bool, wantedUpdates int, wantedDeletes int) {
	bootstrapped, ok := m.bootstrapped[key]

	if !ok {
		dlog.Infof(context.TODO(), "%s bootstrapped: wanted %v, got nothing", key, wantedBootstrap)
		bootstrapped = false
	} else {
		dlog.Infof(context.TODO(), "%s bootstrapped: wanted %v, got %v", key, wantedBootstrap, bootstrapped)
	}

	if bootstrapped != wantedBootstrap {
		m.t.Errorf("%s bootstrapped: wanted %v, got %v", key, wantedBootstrap, bootstrapped)
	}

	got, ok := m.updates[key]

	if !ok {
		dlog.Infof(context.TODO(), "%s updates: wanted %d, got nothing", key, wantedUpdates)
		got = 0
	} else {
		dlog.Infof(context.TODO(), "%s updates: wanted %d, got %d", key, wantedUpdates, got)
	}

	if got != wantedUpdates {
		m.t.Errorf("%s updates: wanted %d, got %d", key, wantedUpdates, got)
	}

	got, ok = m.deletes[key]

	if !ok {
		dlog.Infof(context.TODO(), "%s deletes: wanted %d, got nothing", key, wantedDeletes)
		got = 0
	} else {
		dlog.Infof(context.TODO(), "%s deletes: wanted %d, got %d", key, wantedDeletes, got)
	}

	if got != wantedDeletes {
		m.t.Errorf("%s deletes: wanted %d, got %d", key, wantedDeletes, got)
	}
}

// Make sure that the error count is what we expect.
func (m *fswMetadata) checkErrors(wanted int) {
	dlog.Infof(context.TODO(), "checkErrors: wanted %d, have %d", wanted, m.errorCount)

	if m.errorCount != wanted {
		m.t.Errorf("errors: wanted %d, got %d", wanted, m.errorCount)
	}
}

// Write a file, generating a certain number of Write events for it.
func (m *fswMetadata) writeFile(name string, count int, slow bool) bool {
	path := filepath.Join(m.dir, name)

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0755)

	if err != nil {
		m.t.Errorf("could not open %s: %s", path, err)
		return false
	}

	dlog.Infof(context.TODO(), "%s: opened %s", runtime.GOOS, path)

	// If our caller wants slowness, give 'em slowness.
	if slow {
		time.Sleep(time.Second)
	}

	for i := 0; i < count; i++ {
		dlog.Infof(context.TODO(), "writing chunk %d of %s", i, path)

		_, err = f.WriteString("contents!\n")

		if err != nil {
			m.t.Errorf("could not write chunk %d of %s: %s", i, path, err)
			return false
		}

		dlog.Infof(context.TODO(), "syncing chunk %d of %s", i, path)

		// Make sure to flush the file.
		err = f.Sync()

		if err != nil {
			m.t.Errorf("could not sync chunk %d of %s: %s", i, path, err)
			return false
		}

		// If our caller wants slowness, give 'em slowness.
		if slow {
			time.Sleep(time.Second)
		}
	}

	err = f.Close()

	if err != nil {
		m.t.Errorf("could not close %s: %s", path, err)
	}

	dlog.Infof(context.TODO(), "closed %s", path)

	return true
}

// Send an error, to test the error-handler path.
//
// XXX This is a pretty blatant hack, since we're just suborning an
// implementation detail of the FSWatcher to do this. Oh well.
func (m *fswMetadata) sendError() {
	m.fsw.FSW.Errors <- errors.New("OH GOD AN ERROR")

	// This seems necessary to give the goroutine running in the
	// FSWatcher a chance to process the error before our caller
	// tries to check things.
	time.Sleep(250 * time.Millisecond)
}

func TestFSWatcherExtantFiles(t *testing.T) {
	m, err := newMetadata(t)

	if err != nil {
		return
	}

	dlog.Infof(context.TODO(), "FSW initialized for ExtantFiles (%s)", m.dir)

	defer m.done()

	if !m.writeFile("f1", 1, false) {
		return
	}

	if !m.writeFile("f2", 2, false) {
		return
	}

	if !m.writeFile("f3", 3, false) {
		return
	}

	m.fsw.WatchDir(context.TODO(), m.dir, m.eventHandler)

	m.check("f1", true, 1, 0)
	m.check("f2", true, 1, 0)
	m.check("f3", true, 1, 0)

	m.checkErrors(0)

	m.sendError()

	m.checkErrors(1)
}

func TestFSWatcherNoExtantFiles(t *testing.T) {
	m, err := newMetadata(t)

	if err != nil {
		return
	}

	dlog.Infof(context.TODO(), "FSW initialized for NonExtantFiles (%s)", m.dir)

	m.fsw.WatchDir(context.TODO(), m.dir, m.eventHandler)

	if !m.writeFile("f1", 1, false) {
		return
	}

	if !m.writeFile("f2", 2, false) {
		return
	}

	if !m.writeFile("f3", 3, false) {
		return
	}

	time.Sleep(1 * time.Second)

	m.check("f1", false, 1, 0)
	m.check("f2", false, 1, 0)
	m.check("f3", false, 1, 0)

	m.done()

	time.Sleep(1 * time.Second)

	m.check("f1", false, 1, 1)
	m.check("f2", false, 1, 1)
	m.check("f3", false, 1, 1)

	m.checkErrors(0)
}

func TestFSWatcherSlow(t *testing.T) {
	m, err := newMetadata(t)

	if err != nil {
		return
	}

	dlog.Infof(context.TODO(), "FSW initialized for NonExtantFiles (%s)", m.dir)

	m.fsw.WatchDir(context.TODO(), m.dir, m.eventHandler)

	if !m.writeFile("f1", 1, true) {
		return
	}

	if !m.writeFile("f2", 2, true) {
		return
	}

	if !m.writeFile("f3", 3, true) {
		return
	}

	time.Sleep(1 * time.Second)

	// Each of these should now register an event for creation, plus an
	// event for each write.
	m.check("f1", false, 2, 0)
	m.check("f2", false, 3, 0)
	m.check("f3", false, 4, 0)

	m.done()

	time.Sleep(1 * time.Second)

	m.check("f1", false, 2, 1)
	m.check("f2", false, 3, 1)
	m.check("f3", false, 4, 1)

	m.checkErrors(0)
}
