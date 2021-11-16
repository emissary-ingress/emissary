// +build !windows

package sigint

import (
	"os"
)

func SendInterrupt(proc *os.Process) error {
	return proc.Signal(os.Interrupt)
}
