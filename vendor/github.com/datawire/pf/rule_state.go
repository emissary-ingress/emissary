package pf

import (
	"fmt"
)

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// State wether the packet filter should keep
// track of the packet flows (stateful packet filter)
// or not (stateless packet filter).
type State uint8

const (
	// StateNo no state tracking with this rule
	StateNo State = 0
	// StateKeep track state inside the packet filter
	StateKeep State = C.PF_STATE_NORMAL
	// StateModulate keeps state and adds high quality random sequence numbers
	// for tcp
	StateModulate State = C.PF_STATE_MODULATE
	// StateSynproxy keeps state and creates new tcp connections to hide internals
	StateSynproxy State = C.PF_STATE_SYNPROXY
)

func (s State) String() string {
	switch s {
	case StateNo:
		return ""
	case StateKeep:
		return "keep state"
	case StateModulate:
		return "modulate state"
	case StateSynproxy:
		return "synproxy state"
	default:
		return fmt.Sprintf("State(%d)", s)
	}
}
