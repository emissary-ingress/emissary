package pf

import (
	"fmt"
	"unsafe"
)

// #include <sys/ioctl.h>
// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// RuleSet represents a pf rule set that is a collection of
// rules
type RuleSet struct {
	tx   Transaction
	wrap *C.struct_pfioc_trans_e
}

// RuleSetType is the type of a given rule set
type RuleSetType int

const (
	// RuleSetScrub Scrub (packet normalization) rules.
	RuleSetScrub RuleSetType = C.PF_RULESET_SCRUB
	// RuleSetFilter Filter rules.
	RuleSetFilter RuleSetType = C.PF_RULESET_FILTER
	// RuleSetNAT NAT (Network Address Translation) rules.
	RuleSetNAT RuleSetType = C.PF_RULESET_NAT
	// RuleSetBINAT Bidirectional NAT rules.
	RuleSetBINAT RuleSetType = C.PF_RULESET_BINAT
	// RuleSetRedirect Redirect rules.
	RuleSetRedirect RuleSetType = C.PF_RULESET_RDR
	// RuleSetALTQ ALTQ disciplines.
	RuleSetALTQ RuleSetType = C.PF_RULESET_ALTQ
	// RuleSetTable Address tables.
	RuleSetTable RuleSetType = C.PF_RULESET_TABLE
)

// SetType can be used to change the type of a rule set
func (rs *RuleSet) SetType(t RuleSetType) {
	rs.wrap.rs_num = C.int(t)
}

// Type returns the type of the rule set
func (rs RuleSet) Type() RuleSetType {
	return RuleSetType(rs.wrap.rs_num)
}

// SetAnchor can be used to set the anchor path for the rule set
func (rs *RuleSet) SetAnchor(path string) error {
	return cStringCopy(&rs.wrap.anchor[0], path, int(C.MAXPATHLEN))
}

// Anchor returns the anchor of the rule set
func (rs RuleSet) Anchor() string {
	return C.GoString(&rs.wrap.anchor[0])
}

// AddRule adds the given rule to the end of the rule set
func (rs RuleSet) AddRule(r *Rule) error {
	if r == nil {
		panic(fmt.Errorf("Empty/nil rules aren't permitted by pf"))
	}
	defer func() { r.wrap.ticket = 0 }()
	r.wrap.ticket = rs.wrap.ticket

	err := cStringCopy(&r.wrap.anchor[0], rs.Anchor(), int(C.MAXPATHLEN))
	if err != nil {
		return fmt.Errorf("cStringCopy: %s", err)
	}

	r.wrap.pool_ticket, err = rs.tx.handle.poolTicket()
	if err != nil { return err }

	err = rs.tx.handle.ioctl(C.DIOCADDRULE, unsafe.Pointer(&r.wrap))
	if err != nil {
		return fmt.Errorf("DIOCADDRULE: %s", err)
	}
	return nil
}
