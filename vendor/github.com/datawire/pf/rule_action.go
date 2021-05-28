package pf

import (
	"fmt"
)

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// Action that should be performed by pf
type Action uint8

const (
	// ActionPass Filter rule action to pass the traffic
	ActionPass Action = C.PF_PASS
	// ActionDrop Filter rule action to drop the traffic
	ActionDrop Action = C.PF_DROP

	// ActionScrub Scrub rule action to do scrubbing
	ActionScrub Action = C.PF_SCRUB
	// ActionNoScrub Srub rule action to not do scrubbing
	ActionNoScrub Action = C.PF_NOSCRUB

	// ActionNAT NAT rule action to to NAT
	ActionNAT Action = C.PF_NAT
	// ActionNoNAT NAT rule action to not do NAT
	ActionNoNAT Action = C.PF_NONAT

	// ActionBINAT NAT rule action to to BINAT
	ActionBINAT Action = C.PF_BINAT
	// ActionNoBINAT NAT rule action to not do BINAT
	ActionNoBINAT Action = C.PF_NOBINAT

	// ActionRDR RDR rule action to to RDR
	ActionRDR Action = C.PF_RDR
	// ActionNoRDR RDR rule action to not do RDR
	ActionNoRDR Action = C.PF_NORDR

	// ActionSynProxyDrop TODO
	ActionSynProxyDrop Action = C.PF_SYNPROXY_DROP

	// ActionDefer TODO is this divert?
	// Does not exist on Darwin: // ActionDefer Action = C.PF_DEFER
)

func (a Action) String() string {
	switch a {
	case ActionPass:
		return "pass"
	case ActionDrop:
		return "drop"
	case ActionScrub:
		return "scrub"
	case ActionNoScrub:
		return "no scrub"
	case ActionNAT:
		return "nat"
	case ActionNoNAT:
		return "no nat"
	case ActionBINAT:
		return "binat"
	case ActionNoBINAT:
		return "no binat"
	case ActionRDR:
		return "rdr"
	case ActionNoRDR:
		return "no rdr"
	case ActionSynProxyDrop:
		return "synproxy drop"
// Does not exist on Darwin:
/*	case ActionDefer:
		return "defer"
*/
	default:
		return fmt.Sprintf("Action(%d)", a)
	}
}

func (a Action) AnchorString() string {
	switch a {
	case ActionPass, ActionDrop, ActionScrub, ActionNoScrub:
		return "anchor"
	case ActionNAT, ActionNoNAT:
		return "nat-anchor"
	case ActionBINAT, ActionNoBINAT:
		return "binat-anchor"
	case ActionRDR, ActionNoRDR:
		return "rdr-anchor"
	default:
		return fmt.Sprintf("AnchorString(%d)", a)
	}
}
