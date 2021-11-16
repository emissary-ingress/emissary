package dexec

import (
	"syscall"
)

func (c *Cmd) canInterrupt() bool {
	return c != nil &&
		c.Cmd != nil &&
		c.Cmd.SysProcAttr != nil &&
		(c.Cmd.SysProcAttr.CreationFlags&syscall.CREATE_NEW_PROCESS_GROUP) != 0
}
