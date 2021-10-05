package dtest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/datawire/dlib/dlog"
	"golang.org/x/sys/windows"
)

const pattern = "C:\\Windows\\Temp\\datawire-machine-scoped-%s.lock"

// WithNamedMachineLock executes the supplied body with a guarantee
// that it is the only code running (via WithMachineLock) on the
// machine. The name provides scope so this can be used in multiple
// independent ways without conflicts.
func WithNamedMachineLock(ctx context.Context, name string, body func(context.Context)) {
	lockAcquireStart := time.Now()
	filename := fmt.Sprintf(pattern, name)
	uPath, err := windows.UTF16PtrFromString(filename)
	if err != nil {
		exit(filename, err)
	}

	// CreateFile's third argument is a set of sharing flags -- in this case we pass no sharing flags, which means that no other process will be able to open the file, for anything.
	// Unfortunately, CreateFile doesn't support blocking until the sharing policy allows for the file to be open in whatever mode the caller is requesting.
	// So, we're gonna do something slightly disgusting and just poll the system call until it says we have the file
	createFile := func() (windows.Handle, error) {
		return windows.CreateFile(uPath, windows.GENERIC_READ, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	}
	h, err := createFile()
	for ; err != nil; h, err = createFile() {
		if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
			// If it's not a sharing violation then we have an actual, legit error
			exit(filename, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	defer windows.CloseHandle(h)

	dlog.Printf(ctx, "Acquiring machine lock %q took %.2f seconds\n", name, time.Since(lockAcquireStart).Seconds())
	body(ctx)
}
