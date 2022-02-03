package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/datawire/dlib/dlog"
	"github.com/fsnotify/fsnotify"
)

// FSWatcher is a thing that can watch the filesystem for us, and
// call handler functions when things change.
//
// The core of an FSWatcher is fsnotify/fsnotify, but we wrap some
// state around it.
//
// First, fsnotify tries to mark the operation associated with a
// change -- however, these are not always accurate, since the
// filesystem tries to coalesce events that are close in time.
// Therefore FSWatcher doesn't actually look at the operation:
// everything is just "a change happened".
//
// This causes one interesting problem: given a touch of
// temporal separation between Create and Write, we may decide
// to trigger a reconfigure on the Create, before the data have
// been written. To mitigate against that, we'll wait up to half
// a second after an event to see if any other events will be
// happening (with the idea that if you've come within half a
// second of your cert expiring before renewing it, uh, yeah,
// maybe you _will_ have some transient errors).
//
// Second, when we start watching a directory, we make sure that
// "update" events get posted for every file in the directory.
// These are marked as "bootstrap" events.
//
// Finally, rather than posting things to channels, we call a
// handler function whenever anything interesting happens,
// where "interesting" is one of the events above, or an error.
type FSWatcher struct {
	FSW *fsnotify.Watcher

	mutex       sync.Mutex
	handlers    map[string]FSWEventHandler
	handleError FSWErrorHandler
	cTimer      *time.Timer
	marker      chan time.Time
	outstanding map[string]bool
}

// FSWEventHandler is a handler function for an interesting
// event.
type FSWEventHandler func(ctx context.Context, event FSWEvent)

// FSWErrorHandler is a handler function for an error.
type FSWErrorHandler func(ctx context.Context, err error)

// FSWOp specifies the operation for an event.
type FSWOp string

const (
	// FSWUpdate is an update operation
	FSWUpdate FSWOp = "update"

	// FSWDelete is a delete operation
	FSWDelete FSWOp = "delete"
)

// FSWEvent represents a single interesting event.
type FSWEvent struct {
	// Path is the fully-qualified path of the file that changed.
	Path string
	// Op is the operation for this event.
	Op FSWOp
	// Bootstrap is true IFF this is a synthesized event noting
	// that a file existed at the moment we started watching a
	// directory.
	Bootstrap bool
	// Time is when this event happened
	Time time.Time
}

// String returns a string representation of an FSEvent.
func (event FSWEvent) String() string {
	bstr := ""
	if event.Bootstrap {
		bstr = "B|"
	}

	return fmt.Sprintf("%s%s %s", bstr, event.Op, event.Path)
}

// NewFSWatcher instantiates an FSWatcher. At instantiation time,
// no directories are being watched.
func NewFSWatcher(ctx context.Context) (*FSWatcher, error) {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		dlog.Errorf(ctx, "FSW: could not initialize FSWatcher: %v", err)
		return nil, err
	}

	dlog.Debugf(ctx, "FSW: initialized FSWatcher!")

	fsw := &FSWatcher{
		FSW:         watcher,
		handlers:    make(map[string]FSWEventHandler),
		outstanding: make(map[string]bool),
		marker:      make(chan time.Time),
	}

	// Start with the default error handler...
	fsw.handleError = fsw.defaultErrorHandler

	return fsw, nil
}

// SetErrorHandler sets the function that will be used to respond to errors.
func (fsw *FSWatcher) SetErrorHandler(handler FSWErrorHandler) {
	fsw.handleError = handler
}

// WatchDir starts watching a directory, using a specific handler function.
// You'll need to separately call WatchDir for subdirectories if you want
// recursive watches.
func (fsw *FSWatcher) WatchDir(ctx context.Context, dir string, handler FSWEventHandler) error {
	fsw.mutex.Lock()
	defer fsw.mutex.Unlock()

	dlog.Infof(ctx, "FSW: watching %s", dir)

	if err := fsw.FSW.Add(dir); err != nil {
		return err
	}
	fsw.handlers[dir] = handler

	fileinfos, err := ioutil.ReadDir(dir)

	if err != nil {
		return err
	}

	for _, info := range fileinfos {
		fswevent := FSWEvent{
			Path:      path.Join(dir, info.Name()),
			Op:        FSWUpdate,
			Bootstrap: true,
			Time:      info.ModTime(),
		}

		dlog.Debugf(ctx, "FSWatcher: synthesizing %s", fswevent)

		handler(ctx, fswevent)
	}
	return nil
}

// The default error handler just logs the error.
func (fsw *FSWatcher) defaultErrorHandler(ctx context.Context, err error) {
	dlog.Errorf(ctx, "FSW: FSWatcher error: %s", err)
}

// Watch for events, and handle them.
func (fsw *FSWatcher) Run(ctx context.Context) {
	for {
		select {
		case event := <-fsw.FSW.Events:
			fsw.mutex.Lock()

			dlog.Debugf(ctx, "FSW: raw event %s", event)

			// Note that this path is outstanding.
			fsw.outstanding[event.Name] = true

			// Coalesce events for up to half a second.
			if fsw.cTimer != nil {
				dlog.Debugf(ctx, "FSW: stopping cTimer")

				if !fsw.cTimer.Stop() {
					<-fsw.cTimer.C
				}
			}

			dlog.Debugf(ctx, "FSW: starting cTimer")
			fsw.cTimer = time.AfterFunc(500*time.Millisecond, func() {
				fsw.marker <- time.Now()
			})

			dlog.Debugf(ctx, "FSW: unlocking")
			fsw.mutex.Unlock()

		case <-fsw.marker:
			fsw.mutex.Lock()
			dlog.Debugf(ctx, "FSW: MARKER LOCK")
			fsw.cTimer = nil

			keys := make([]string, 0, len(fsw.outstanding))
			for key := range fsw.outstanding {
				keys = append(keys, key)
			}

			fsw.outstanding = make(map[string]bool)

			fsw.mutex.Unlock()

			dlog.Debugf(ctx, "FSW: updates! %s", strings.Join(keys, ", "))

			for _, evtPath := range keys {
				dirname := filepath.Dir(evtPath)
				handler, handlerExists := fsw.handlers[dirname]

				if handlerExists {
					op := FSWUpdate

					info, err := os.Stat(evtPath)

					eventTime := time.Now()

					if err != nil {
						op = FSWDelete
					} else {
						eventTime = info.ModTime()
					}

					fswevent := FSWEvent{
						Path:      evtPath,
						Op:        op,
						Bootstrap: false,
						Time:      eventTime,
					}

					dlog.Debugf(ctx, "FSW: handling %s", fswevent)
					handler(ctx, fswevent)
				} else {
					dlog.Debugf(ctx, "FSW: drop, no handler for dir %s", dirname)
				}
			}

		case err := <-fsw.FSW.Errors:
			dlog.Errorf(ctx, "FSW: filesystem watch error %s", err)

			fsw.handleError(ctx, err)

		case <-ctx.Done():
			dlog.Infof(ctx, "FSW: ctx shutdown, exiting")
			return
		}
	}
}
