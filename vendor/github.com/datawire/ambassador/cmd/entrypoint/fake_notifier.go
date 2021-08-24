package entrypoint

import (
	"sync"
)

// The Notifier struct buffers up notifications to multiple listeners. This is used as plumbing to
// wire up watchers for the K8sStore and ConsulStore. A monotonically increasing changeCount field
// functions as a logical clock tracking how many changes have occured. The notifyCount field tracks
// how many of these changes are to be communicated to listeners. Each listener also tracks its own
// count which starts at zero. This ensures that new listeners are always notified of changes that
// have happened prior to the listener being created.
type Notifier struct {
	cond        *sync.Cond
	autoNotify  bool
	changeCount int // How many total changes have occurred.
	notifyCount int // How many total changes are to be communicated to listeners. This must be <= changeCount.
}

// NewNotifier constructs a new notifier struct that is ready for use.
func NewNotifier() *Notifier {
	return &Notifier{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

// Changed signals that a change has occured that will eventually need to be communicated to all
// listeners.
func (n *Notifier) Changed() {
	callNotify := false
	func() {
		n.cond.L.Lock()
		defer n.cond.L.Unlock()
		n.changeCount += 1
		if n.autoNotify {
			callNotify = true
		}
	}()

	if callNotify {
		n.Notify()
	}
}

// AutoNotify controls the notification mode.
func (n *Notifier) AutoNotify(enabled bool) {
	func() {
		n.cond.L.Lock()
		defer n.cond.L.Unlock()
		n.autoNotify = enabled
	}()

	if enabled {
		n.Notify()
	}
}

// Notify listeners of an and all outstanding changes.
func (n *Notifier) Notify() {
	n.cond.L.Lock()
	defer n.cond.L.Unlock()
	n.notifyCount = n.changeCount
	n.cond.Broadcast()
}

type StopFunc func()

// Listen will invoke the supplied function whenever a change is signaled. Changes will be coalesced
// if they happen quickly enough. A stop function is returned that when invoked will prevent future
// changes from notifying the Listener.
func (n *Notifier) Listen(onChange func()) StopFunc {
	stopped := false
	go func() {
		n.cond.L.Lock()
		defer n.cond.L.Unlock()
		count := 0
		for {
			if stopped {
				return
			}
			if count < n.notifyCount {
				onChange()
				count = n.notifyCount
			}
			n.cond.Wait()
		}
	}()

	return func() {
		n.cond.L.Lock()
		defer n.cond.L.Unlock()
		stopped = true
		n.cond.Broadcast()
	}
}
