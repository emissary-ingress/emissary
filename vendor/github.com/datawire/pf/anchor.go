package pf

import (
	"fmt"
	"unsafe"
)

// #include <sys/ioctl.h>
// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// Anchor allows to read and manipulate rulesets without
// requiring a transaction
type Anchor struct {
	*ioctlDev
	Path string
}

// Rules returns all rules using one ticket
func (a Anchor) Rules(action Action) ([]Rule, error) {
	var rules C.struct_pfioc_rule
	rules.rule.action = C.u_int8_t(action)
	err := cStringCopy(&rules.anchor[0], a.Path, C.MAXPATHLEN)
	if err != nil { return nil, err }
	err = a.ioctl(C.DIOCGETRULES, unsafe.Pointer(&rules))
	if err != nil {
		return nil, fmt.Errorf("DIOCGETRULES: %s", err)
	}
	ruleList := make([]Rule, rules.nr)

	for i := 0; i < int(rules.nr); i++ {
		err = a.rule(action, int(rules.ticket), i, &ruleList[i])
		if err != nil {
			return nil, fmt.Errorf("DIOCGETRULE: %s", err)
		}
	}

	return ruleList, nil
}

// Rule uses the passed ticket to return the rule at the given index
func (a Anchor) rule(action Action, ticket, index int, rule *Rule) error {
	if ticket <= 0 || index < 0 {
		return fmt.Errorf("Invalid ticket or index: ticket %d index %d",
			ticket, index)
	}
	if rule == nil {
		panic(fmt.Errorf("Can't store rule data in nil value"))
	}
	rule.wrap.rule.action = C.u_int8_t(action)
	err := cStringCopy(&rule.wrap.anchor[0], a.Path, C.MAXPATHLEN)
	if err != nil { return err }
	rule.wrap.nr = C.u_int32_t(index)
	rule.wrap.ticket = C.u_int32_t(ticket)
	return a.ioctl(C.DIOCGETRULE, unsafe.Pointer(&rule.wrap))
}

// Anchors returns all sub anchors
func (a Anchor) Anchors() (anchors []Anchor, err error) {
	var ruleset C.struct_pfioc_ruleset
	err = cStringCopy(&ruleset.path[0], a.Path, int(C.MAXPATHLEN))
	if err != nil { return }
	err = a.ioctl(C.DIOCGETRULESETS, unsafe.Pointer(&ruleset))
	if err != nil { return }

	anchors = make([]Anchor, ruleset.nr)
	for idx, _ := range anchors {
		ruleset.nr = C.uint(idx)
		err = a.ioctl(C.DIOCGETRULESET, unsafe.Pointer(&ruleset))
		anchors[idx] = Anchor{a.ioctlDev, fmt.Sprintf("%s/%s", a.Path, C.GoString(&ruleset.name[0]))}
	}

	return
}

func (a Anchor) AnchorMap() (result map[string]Anchor, err error) {
	anchors, err := a.Anchors()
	if err != nil { return }

	result = make(map[string]Anchor)
	for _, a := range anchors {
		result[a.Path] = a
	}
	return
}

func (a Anchor) poolTicket() (ticket C.u_int32_t, err error) {
	var pool C.struct_pfioc_pooladdr
	err = a.ioctl(C.DIOCBEGINADDRS, unsafe.Pointer(&pool))
	if err != nil {
		err = fmt.Errorf("DIOCBEGINADDRS: %s", err)
		return
	}
	ticket = pool.ticket
	return
}

func (a Anchor) _addRule(r Rule, action C.uint) (err error) {
	err = cStringCopy(&r.wrap.anchor[0], a.Path, C.MAXPATHLEN)
	if err != nil { return }

	r.wrap.pool_ticket, err = a.poolTicket()
	if err != nil { return }

	r.wrap.action = C.PF_CHANGE_GET_TICKET
	err = a.ioctl(C.DIOCCHANGERULE, unsafe.Pointer(&r.wrap))
	if err != nil { return }
	r.wrap.action = action
	err = a.ioctl(C.DIOCCHANGERULE, unsafe.Pointer(&r.wrap))
	if err != nil { return }
	return
}

func (a Anchor) PrependRule(r Rule) (err error) {
	return a._addRule(r, C.PF_CHANGE_ADD_HEAD)
}

func (a Anchor) AppendRule(r Rule) (err error) {
	return a._addRule(r, C.PF_CHANGE_ADD_TAIL)
}

func (a Anchor) RemoveRule(r Rule) (err error) {
	r.wrap.action = C.PF_CHANGE_GET_TICKET
	err = a.ioctl(C.DIOCCHANGERULE, unsafe.Pointer(&r.wrap))
	if err != nil { return }
	r.wrap.action = C.PF_CHANGE_REMOVE
	err = a.ioctl(C.DIOCCHANGERULE, unsafe.Pointer(&r.wrap))
	if err != nil { return }
	return
}
