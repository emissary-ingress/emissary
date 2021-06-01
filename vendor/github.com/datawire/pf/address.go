package pf

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

// #include <net/if.h>
// #include <net/pfvar.h>
/*
int addr_type(struct pf_addr_wrap* addr) { return addr->type; }
void set_addr_type(struct pf_addr_wrap* addr, int type) { addr->type = type; }
int addr_cnt(struct pf_addr_wrap* addr) { return addr->p.tblcnt; }
char* addr_name(struct pf_addr_wrap* addr) { return &addr->v.ifname[0]; }
*/
import "C"

// Address wraps the pf address (cgo)
type Address struct {
	wrap *C.struct_pf_addr_wrap
	af   C.sa_family_t
}

func newAddress() *Address {
	var wrap C.struct_pf_addr_wrap
	a := Address{wrap: &wrap}
	return &a
}

// any is the pf represnetation of the any address
var any = net.IPNet{
	IP:   net.IPv6zero,
	Mask: net.IPMask(net.IPv6zero),
}
var singleIPv4 = net.CIDRMask(32, 128)
var singleIPv6 = net.CIDRMask(128, 128)

// DynamicFlag can be set on an address that is derived from
// an interface
type DynamicFlag uint8

const (
	// DynamicFlagNetwork translates to the network(s) attached to the interface
	DynamicFlagNetwork DynamicFlag = C.PFI_AFLAG_NETWORK
	// DynamicFlagBroadcast translates to the interface's broadcast address(es).
	DynamicFlagBroadcast DynamicFlag = C.PFI_AFLAG_BROADCAST
	// DynamicFlagPeer translates to the point-to-point interface's peer address(es).
	DynamicFlagPeer DynamicFlag = C.PFI_AFLAG_PEER
	// DynamicFlagNoAlias do not include interface aliases.
	DynamicFlagNoAlias DynamicFlag = C.PFI_AFLAG_NOALIAS
)

func (f DynamicFlag) String() string {
	switch f {
	case DynamicFlagNetwork:
		return "network"
	case DynamicFlagBroadcast:
		return "broadcast"
	case DynamicFlagPeer:
		return "peer"
	case DynamicFlagNoAlias:
		return "0"
	default:
		return fmt.Sprintf("DynamicFlag(%d)", int(f))
	}
}

// AllDynamicFlags contains all danymic flags in usual order
var AllDynamicFlags = []DynamicFlag{
	DynamicFlagNetwork,
	DynamicFlagBroadcast,
	DynamicFlagPeer,
	DynamicFlagNoAlias,
}

// Dynamic returns true if the address is dynamic
// based of the interface
func (a Address) Dynamic() bool {
	return C.addr_type(a.wrap) == C.PF_ADDR_DYNIFTL
}

// Interface the name of the interface (e..g. used for dynamic address),
// returns an empty string if no interface is set
func (a Address) Interface() string {
	return C.GoString(C.addr_name(a.wrap)) // ifname union
}

// SetInterface turns address into dynamic interface reference,
// type of interface reference can be changed with flags
func (a *Address) SetInterface(itf string) error {
	err := bytesCopy(a.wrap.v[:], itf, C.IFNAMSIZ)
	if err != nil {
		return err
	}
	C.set_addr_type(a.wrap, C.PF_ADDR_DYNIFTL)
	return nil
}

// Table returns true if the address references a table
func (a Address) Table() bool {
	return C.addr_type(a.wrap) == C.PF_ADDR_TABLE
}

// DynamicFlag returns true if the flag is set for the address
func (a Address) DynamicFlag(flag DynamicFlag) bool {
	return uint8(a.wrap.iflags)&uint8(flag) == uint8(flag)
}

// SetDynamicFlag sets the dynamic interface flag
func (a *Address) SetDynamicFlag(flag DynamicFlag) {
	a.wrap.iflags = C.u_int8_t(flag)
}

// DynamicCount returns the dynamic count
func (a Address) DynamicCount() int {
	return int(C.addr_cnt(a.wrap)) // dyncnt union
}

// TableName returns the name of the table or an empty string if not set
func (a Address) TableName() string {
	return C.GoString(C.addr_name(a.wrap)) // tblname union
}

// SetTableName turns address into table reference, using given name
func (a *Address) SetTableName(name string) error {
	err := bytesCopy(a.wrap.v[:], name, C.PF_TABLE_NAME_SIZE)
	if err != nil {
		return err
	}
	C.set_addr_type(a.wrap, C.PF_ADDR_TABLE)
	return nil
}

// TableCount returns the table count
func (a Address) TableCount() int {
	return int(C.addr_cnt(a.wrap)) // tblcnt union
}

// NoRoute any address which is not currently routable
func (a Address) NoRoute() bool {
	return C.addr_type(a.wrap) == C.PF_ADDR_NOROUTE
}

// SetNoRoute turns address into no routeable address
func (a *Address) SetNoRoute() {
	C.set_addr_type(a.wrap, C.PF_ADDR_NOROUTE)
}

// URPFFailed any source address that fails a unicast reverse
// path forwarding (URPF) check, i.e. packets coming
// in on an interface other than that which holds the
// route back to the packet's source address
func (a Address) URPFFailed() bool {
	return C.addr_type(a.wrap) == C.PF_ADDR_URPFFAILED
}

// SetURPFFailed see URPFFailed for details
func (a *Address) SetURPFFailed() {
	C.set_addr_type(a.wrap, C.PF_ADDR_URPFFAILED)
}

// Mask returns true if address is an ip address with mask
func (a Address) Mask() bool {
	return C.addr_type(a.wrap) == C.PF_ADDR_ADDRMASK
}

// Range returns true if is an address range with start
// and end ip addr
func (a Address) Range() bool {
	return C.addr_type(a.wrap) == C.PF_ADDR_RANGE
}

// Any returns true if address represents any address
func (a Address) Any() bool {
	if !a.Mask() {
		return false
	}

	return bytes.Compare(any.IP, a.wrap.v[0:16]) == 0 &&
		bytes.Compare(any.Mask, a.wrap.v[16:32]) == 0
}

// SetAny will turn the address into an any IP address
func (a *Address) SetAny() {
	a.SetIPNet(&any)
}

// IPNet returns the IPNetwork (IPv4/IPv6) of the address with mask
func (a Address) IPNet() *net.IPNet {
	var ipn net.IPNet

	if a.af == C.AF_INET {
		ipn.IP = a.wrap.v[0:4]     // addr union
		ipn.Mask = a.wrap.v[16:20] // mask union
	} else {
		ipn.IP = a.wrap.v[0:16]    // addr union
		ipn.Mask = a.wrap.v[16:32] // mask union
	}

	return &ipn
}

// IPRange returns the start and end ip address of the range
func (a Address) IPRange() (net.IP, net.IP) {
	start := net.IP(a.wrap.v[0:16])
	end := net.IP(a.wrap.v[16:32])
	return start, end
}

// SetIPRange sets start and end address and turns object
// into ip range
func (a *Address) SetIPRange(start, end net.IP) {
	copy(a.wrap.v[0:16], start)
	copy(a.wrap.v[16:32], end)
	C.set_addr_type(a.wrap, C.PF_ADDR_RANGE)
}

// SetIPNet updates the ip address and mask and changes
// the type to AddressMask
func (a *Address) SetIPNet(ipn *net.IPNet) {
	if ipv4 := ipn.IP.To4(); ipv4 != nil {
		copy(a.wrap.v[0:4], ipv4)
		copy(a.wrap.v[16:20], ipn.Mask)
		a.af = C.AF_INET
	} else {
		copy(a.wrap.v[0:16], ipn.IP)
		copy(a.wrap.v[16:32], ipn.Mask)
		a.af = C.AF_INET6
	}
	C.set_addr_type(a.wrap, C.PF_ADDR_ADDRMASK)
}

func (a Address) String() string {
	if a.Dynamic() {
		str := []string{a.Interface()}
		for _, flag := range AllDynamicFlags {
			if a.DynamicFlag(flag) {
				str = append(str, flag.String())
			}
		}
		return fmt.Sprintf("(%s)", strings.Join(str, ":"))
	} else if a.Table() {
		return fmt.Sprintf("<%s>", a.TableName())
	} else if a.NoRoute() {
		return "no-route"
	} else if a.URPFFailed() {
		return "urpf-failed"
	} else if a.Any() {
		return "any"
	} else if a.Mask() {
		return a.IPNet().String()
	} else if a.Range() {
		s, e := a.IPRange()
		return fmt.Sprintf("%s - %s", s, e)
	} else {
		return fmt.Sprintf("Address(%d)", C.addr_type(a.wrap))
	}
}

// ParseCIDR parses the passed address in CIDR notation
// and sets the extracted addess, mask and af. Id mask is missing
// IP address is assumed and mask is set to 32 IPv4 or 128 IPv6.
// May return a parse error if the address is invalid CIDR or
// IP address
func (a *Address) ParseCIDR(address string) error {
	if strings.ContainsRune(address, '/') {
		_, n, err := net.ParseCIDR(address)
		if err != nil {
			return err
		}
		a.SetIPNet(n)
	} else {
		var ipn net.IPNet
		ipn.IP = net.ParseIP(address)
		if ipn.IP.To4() != nil {
			ipn.Mask = singleIPv4
		} else {
			ipn.Mask = singleIPv6
		}
		a.SetIPNet(&ipn)
	}

	return nil
}
