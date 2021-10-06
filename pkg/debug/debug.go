package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// This struct serves as the root of all runtime debug info for the process. This consists of timers
// that aggregate timing info for various kinds of actions, as well as atomic values that can be
// updated as relevant state changes.
type Debug struct {
	mutex  sync.Mutex        // Protects the whole debug struct.
	timers map[string]*Timer // Holds the debug timers.
	values map[string]*Value // holds the debug values.

	clock ClockFunc // clock function to pass to all the timers
}

// An atomic.Value with custom json marshalling.
type Value atomic.Value

func (v *Value) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent((*atomic.Value)(v).Load(), "", "  ")
}

// The default debug root.
var root = NewDebug()

// A key for use with context values.
var key = &struct{}{}

// The NewContext function creates a child context associated with the given Debug root.
func NewContext(parent context.Context, debug *Debug) context.Context {
	return context.WithValue(parent, key, debug)
}

// The FromContext function retrieves the correct debug root for the given context.
func FromContext(ctx context.Context) (result *Debug) {
	value := ctx.Value(key)
	if value != nil {
		result = value.(*Debug)
	} else {
		result = root
	}
	return
}

// Create a new set of debug info.
func NewDebug() *Debug {
	return NewDebugWithClock(time.Now)
}

// Create a new set of debug info with the specified clock function.
func NewDebugWithClock(clock ClockFunc) *Debug {
	return &Debug{clock: clock, timers: map[string]*Timer{}, values: map[string]*Value{}}
}

// Access the contexts of the debug info while holding the mutex.
func (d *Debug) withMutex(f func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	f()
}

// The Timer() method ensures the named timer exists and returns it.
func (d *Debug) Timer(name string) (result *Timer) {
	d.withMutex(func() {
		var ok bool
		result, ok = d.timers[name]
		if !ok {
			result = NewTimerWithClock(d.clock)
			d.timers[name] = result
		}
	})
	return
}

// The Value() method ensures the named atomic.Value exists and returns it.
func (d *Debug) Value(name string) (result *atomic.Value) {
	d.withMutex(func() {
		r, ok := d.values[name]
		if !ok {
			r = &Value{}
			d.values[name] = r
		}
		result = (*atomic.Value)(r)
	})
	return
}

// The ServeHTTP() method will serve a json representation of the contents of the debug root.
func (d *Debug) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.withMutex(func() {
		bytes, err := json.MarshalIndent(map[string]interface{}{
			"timers": d.timers,
			"values": d.values,
		}, "", "  ")

		if err != nil {
			http.Error(w, fmt.Sprintf("error marshalling debug info: %v", err), http.StatusInternalServerError)
		} else {
			_, _ = w.Write(append(bytes, '\n'))
		}
	})
}
