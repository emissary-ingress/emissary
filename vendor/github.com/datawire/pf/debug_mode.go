package pf

import (
	"fmt"
)

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// DebugMode of the packet filter
type DebugMode int

const (
	// DebugModeNone debugging is disabled
	DebugModeNone DebugMode = C.PF_DEBUG_NONE
	// DebugModeUrgent only urgent info
	DebugModeUrgent DebugMode = C.PF_DEBUG_URGENT
	// DebugModeMisc some more info
	DebugModeMisc DebugMode = C.PF_DEBUG_MISC
	// DebugModeNoisy lots of debug messages
	DebugModeNoisy DebugMode = C.PF_DEBUG_NOISY
)

func (d DebugMode) String() string {
	switch d {
	case DebugModeNone:
		return "none"
	case DebugModeUrgent:
		return "urgent"
	case DebugModeMisc:
		return "misc"
	case DebugModeNoisy:
		return "noisy"
	default:
		return fmt.Sprintf("DebugMode(%d)", d)
	}
}
