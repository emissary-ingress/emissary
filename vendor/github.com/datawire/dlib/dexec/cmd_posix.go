// +build !windows

package dexec

func (c *Cmd) canInterrupt() bool {
	return true
}
