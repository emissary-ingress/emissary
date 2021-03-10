// +build windows

package daemon

import (
	"github.com/pkg/errors"

	"github.com/datawire/ambassador/pkg/supervisor"
)

// GuessRunAsInfo attempts to construct a RunAsInfo for the user logged in at
// the primary display
func GuessRunAsInfo(_ *supervisor.Process) (*RunAsInfo, error) {
	return nil, errors.New("Not implemented on this platform")
}

// GetFreePort asks the kernel for a free open port that is ready to use.
// Similar to telepresence.utilities.find_free_port()
func GetFreePort() (int, error) {
	return 0, errors.New("Not implemented on this platform")
}
