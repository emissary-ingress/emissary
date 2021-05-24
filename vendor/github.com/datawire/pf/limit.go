package pf

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// Limit represents a hard packet filter limit
type Limit int

const (
	// LimitStates limits the number of pf states
	LimitStates Limit = C.PF_LIMIT_STATES
	// LimitSourceNodes limits the number of pf source nodes
	LimitSourceNodes Limit = C.PF_LIMIT_SRC_NODES
	// LimitFragments limits the number of pf fragments
	LimitFragments Limit = C.PF_LIMIT_FRAGS
	// LimitTableEntries limits the number of addresses in a table
	LimitTableEntries Limit = C.PF_LIMIT_TABLE_ENTRIES
)
