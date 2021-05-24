package pf

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// Rule wraps the pf rule (cgo)
type Rule struct {
	wrap C.struct_pfioc_rule
}

// RuleStats contains usefule pf rule statistics
type RuleStats struct {
	Evaluations         uint64
	PacketIn, PacketOut uint64
	BytesIn, BytesOut   uint64
}

// Stats copies the rule statistics into the passed
// RuleStats struct
func (r Rule) Stats(stats *RuleStats) {
	stats.Evaluations = uint64(r.wrap.rule.evaluations)
	stats.PacketIn = uint64(r.wrap.rule.packets[0])
	stats.PacketOut = uint64(r.wrap.rule.packets[1])
	stats.BytesIn = uint64(r.wrap.rule.bytes[0])
	stats.BytesOut = uint64(r.wrap.rule.bytes[1])
}

// SetProtocol sets the protocol matcher of the rule if the
func (r *Rule) SetProtocol(p Protocol) {
	r.wrap.rule.proto = C.u_int8_t(p)
}

// Protocol that is matched by the rule
func (r Rule) Protocol() Protocol {
	return Protocol(r.wrap.rule.proto)
}

// SetLog enables logging of packets to the log interface
func (r *Rule) SetLog(enabled bool) {
	if enabled {
		r.wrap.rule.log = 1
	} else {
		r.wrap.rule.log = 0
	}
}

// Log returns true if matching packets are logged
func (r Rule) Log() bool {
	return r.wrap.rule.log == 1
}

// SetQuick skips further evaluations if packet matched
func (r *Rule) SetQuick(enabled bool) {
	if enabled {
		r.wrap.rule.quick = 1
	} else {
		r.wrap.rule.quick = 0
	}
}

// Quick returns true if matching packets are last to evaluate in the rule list
func (r Rule) Quick() bool {
	return r.wrap.rule.quick == 1
}

// SetState sets if the rule keeps state or not
func (r *Rule) SetState(s State) {
	r.wrap.rule.keep_state = C.u_int8_t(s)
}

// State returns the state tracking configuration of the rule
func (r Rule) State() State {
	return State(r.wrap.rule.keep_state)
}

// SetDirection sets the direction the traffic flows
func (r *Rule) SetDirection(dir Direction) {
	r.wrap.rule.direction = C.u_int8_t(dir)
}

// Direction returns the rule matching direction
func (r Rule) Direction() Direction {
	return Direction(r.wrap.rule.direction)
}

// SetAction sets the action on the traffic flow
func (r *Rule) SetAction(a Action) {
	r.wrap.rule.action = C.u_int8_t(a)
}

// Action returns the action that is performed when rule matches
func (r Rule) Action() Action {
	return Action(r.wrap.rule.action)
}

// SetAddressFamily sets the address family to match on
func (r *Rule) SetAddressFamily(af AddressFamily) {
	r.wrap.rule.af = C.sa_family_t(af)
}

// AddressFamily returns the address family that is matched on
func (r Rule) AddressFamily() AddressFamily {
	return AddressFamily(r.wrap.rule.af)
}

// AnchorCall returns the anchor name for anchor call rules
func (r Rule) AnchorCall() string {
	return C.GoString(&r.wrap.anchor_call[0])
}

// SetAnchorCall sets the anchor to call
func (r *Rule) SetAnchorCall(anchor string) (err error) {
	err = cStringCopy(&r.wrap.anchor_call[0], anchor, C.MAXPATHLEN)
	if err != nil { return }
	return
}
