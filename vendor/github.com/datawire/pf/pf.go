package pf

import (
	"fmt"
	"time"
	"unsafe"
)

// #include <arpa/inet.h>
// #include <sys/ioctl.h>
// #include <net/if.h>
// #include <net/pfvar.h>
// #include "helpers.h"
import "C"

// Handle to the pf kernel module using ioctl
type Handle struct {
	// Anchor root anchor (ruleset without anchor)
	Anchor
}

// Open pf ioctl dev
func Open() (*Handle, error) {
	dev, err := newIoctlDev("/dev/pf")
	if err != nil {
		return nil, err
	}
	h := &Handle{
		Anchor: Anchor{ioctlDev: dev, Path: ""},
	}
	return h, nil
}

func (h Handle) NatLook(saddr string, sport int, daddr string, dport int) (addr string, port int, err error) {
	var pnl C.struct_pfioc_natlook
	pnl.af = C.AF_INET
	pnl.proto = C.u_int8_t(6) // tcp
	pnl.proto_variant = C.u_int8_t(0)
	pnl.direction = C.PF_OUT
	cerr := C.inet_pton(C.int(pnl.af), C.CString(saddr), unsafe.Pointer(&pnl.saddr))
	if cerr != 1 {
		err = fmt.Errorf("inet_pton: %d", 1)
		return
	}
	cerr = C.inet_pton(C.int(pnl.af), C.CString(daddr), unsafe.Pointer(&pnl.daddr))
	if cerr != 1 {
		err = fmt.Errorf("inet_pton: %d", 1)
		return
	}
	C.set_natlook_sport(&pnl, C.htons_f(C.u_int16_t(sport)))
	C.set_natlook_dport(&pnl, C.htons_f(C.u_int16_t(dport)))

	err = h.ioctl(C.DIOCNATLOOK, unsafe.Pointer(&pnl))
	if err != nil { return }

	var dst C.struct_pton_addr
	caddr := C.inet_ntop(C.int(pnl.af), unsafe.Pointer(&pnl.rdaddr), C.get_pton_addr(&dst), C.INET_ADDRSTRLEN)
	if caddr == nil {
		err = fmt.Errorf("inet_ntop")
		return
	}
	addr = C.GoString(caddr)
	port = int(C.ntohs_f(C.get_natlook_rdport(&pnl)))
	return
}

// SetStatusInterface sets the status interface(s) for pf
// usually that is something like pflog0. The device needs
// to be created before using interface cloning.
func (h Handle) SetStatusInterface(dev string) error {
	var pi C.struct_pfioc_if
	err := cStringCopy(&pi.ifname[0], dev, C.IFNAMSIZ)
	if err != nil {
		return err
	}
	err = h.ioctl(C.DIOCSETSTATUSIF, unsafe.Pointer(&pi))
	if err != nil {
		return fmt.Errorf("DIOCSETSTATUSIF: %s", err)
	}
	return nil
}

// StatusInterface returns the currently configured status
// interface or an error.
func (h Handle) StatusInterface() (string, error) {
	var pi C.struct_pfioc_if
	err := h.ioctl(C.DIOCSETSTATUSIF, unsafe.Pointer(&pi))
	if err != nil {
		return "", fmt.Errorf("DIOCSETSTATUSIF: %s", err)
	}
	return C.GoString(&(pi.ifname[0])), nil
}

// Start the packet filter.
func (h Handle) Start() error {
	err := h.ioctl(C.DIOCSTART, nil)
	if err != nil {
		return fmt.Errorf("DIOCSTART: %s", err)
	}
	return nil
}

// Stop the packet filter
func (h Handle) Stop() error {
	err := h.ioctl(C.DIOCSTOP, nil)
	if err != nil {
		return fmt.Errorf("DIOCSTOP: %s", err)
	}
	return nil
}

// UpdateStatistics of the packet filter
func (h Handle) UpdateStatistics(stats *Statistics) error {
	err := h.ioctl(C.DIOCGETSTATUS, unsafe.Pointer(stats))
	if err != nil {
		return fmt.Errorf("DIOCGETSTATUS: %s", err)
	}
	return nil
}

// SetDebugMode of the packetfilter
func (h Handle) SetDebugMode(mode DebugMode) error {
	level := C.u_int32_t(mode)
	err := h.ioctl(C.DIOCSETDEBUG, unsafe.Pointer(&level))
	if err != nil {
		return fmt.Errorf("DIOCSETDEBUG: %s", err)
	}
	return nil
}

// ClearPerRuleStats clear per-rule statistics
func (h Handle) ClearPerRuleStats() error {
	err := h.ioctl(C.DIOCCLRRULECTRS, nil)
	if err != nil {
		return fmt.Errorf("DIOCCLRRULECTRS: %s", err)
	}
	return nil
}

// ClearPFStats clear the internal packet filter statistics
func (h Handle) ClearPFStats() error {
	err := h.ioctl(C.DIOCCLRSTATUS, nil)
	if err != nil {
		return fmt.Errorf("DIOCCLRSTATUS: %s", err)
	}
	return nil
}

// ClearSourceNodes clear the tree of source tracking nodes
func (h Handle) ClearSourceNodes() error {
	err := h.ioctl(C.DIOCCLRSRCNODES, nil)
	if err != nil {
		return fmt.Errorf("DIOCCLRSRCNODES: %s", err)
	}
	return nil
}

// SetHostID set the host ID, which is used by pfsync to identify
// which host created state table entries.
func (h Handle) SetHostID(id uint32) error {
	hostid := C.u_int32_t(id)
	err := h.ioctl(C.DIOCSETHOSTID, unsafe.Pointer(&hostid))
	if err != nil {
		return fmt.Errorf("DIOCSETHOSTID : %s", err)
	}
	return nil
}

// SetTimeout set the state timeout to specified duration
func (h Handle) SetTimeout(t Timeout, d time.Duration) error {
	var tm C.struct_pfioc_tm
	tm.timeout = C.int(t)
	tm.seconds = C.int(d / time.Second)
	err := h.ioctl(C.DIOCSETTIMEOUT, unsafe.Pointer(&tm))
	if err != nil {
		return fmt.Errorf("DIOCSETTIMEOUT: %s", err)
	}
	return nil
}

// Limit returns the currently configured limit for the memory pool
func (h Handle) Limit(l Limit) (uint, error) {
	var lm C.struct_pfioc_limit
	lm.index = C.int(l)
	err := h.ioctl(C.DIOCGETLIMIT, unsafe.Pointer(&lm))
	if err != nil {
		return uint(0), fmt.Errorf("DIOCGETLIMIT: %s", err)
	}
	return uint(lm.limit), nil
}

// SetLimit sets hard limits on the memory pools used by the packet filter
func (h Handle) SetLimit(l Limit, limit uint) error {
	var lm C.struct_pfioc_limit
	lm.index = C.int(l)
	lm.limit = C.uint(limit)
	err := h.ioctl(C.DIOCSETLIMIT, unsafe.Pointer(&lm))
	if err != nil {
		return fmt.Errorf("DIOCSETLIMIT: %s", err)
	}
	return nil
}

// Timeout returns the currently configured timeout duration
func (h Handle) Timeout(t Timeout) (time.Duration, error) {
	var tm C.struct_pfioc_tm
	var d time.Duration
	tm.timeout = C.int(t)
	err := h.ioctl(C.DIOCGETTIMEOUT, unsafe.Pointer(&tm))
	if err != nil {
		return d, fmt.Errorf("DIOCGETTIMEOUT: %s", err)
	}
	d = time.Duration(int(tm.seconds)) * time.Second
	return d, nil
}
