package pf

import (
	"fmt"
	"strings"
	"time"
)

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// SendReceivedStats stats for send and received IPv4/6 traffic
type SendReceivedStats struct {
	SendIPv4, ReceivedIPv4, SendIPv6, ReceivedIPv6 uint64
}

// PacketsPass num of packets passed for the interface
func (s Statistics) PacketsPass() SendReceivedStats {
	return SendReceivedStats{
		SendIPv4:     uint64(s.wrap.pcounters[0][0][C.PF_PASS]),
		ReceivedIPv4: uint64(s.wrap.pcounters[0][1][C.PF_PASS]),
		SendIPv6:     uint64(s.wrap.pcounters[1][0][C.PF_PASS]),
		ReceivedIPv6: uint64(s.wrap.pcounters[1][1][C.PF_PASS]),
	}
}

// PacketsDrop num of packets droped for the interface
func (s Statistics) PacketsDrop() SendReceivedStats {
	return SendReceivedStats{
		SendIPv4:     uint64(s.wrap.pcounters[0][0][C.PF_DROP]),
		ReceivedIPv4: uint64(s.wrap.pcounters[0][1][C.PF_DROP]),
		SendIPv6:     uint64(s.wrap.pcounters[1][0][C.PF_DROP]),
		ReceivedIPv6: uint64(s.wrap.pcounters[1][1][C.PF_DROP]),
	}
}

// Bytes returns num of send and received bytes for the interface
func (s Statistics) Bytes() SendReceivedStats {
	return SendReceivedStats{
		SendIPv4:     uint64(s.wrap.bcounters[0][0]),
		ReceivedIPv4: uint64(s.wrap.bcounters[0][1]),
		SendIPv6:     uint64(s.wrap.bcounters[1][0]),
		ReceivedIPv6: uint64(s.wrap.bcounters[1][1]),
	}
}

// Statistics about the internal packet filter
type Statistics struct {
	wrap C.struct_pf_status
}

// Running returns true if packet filter enabled
func (s Statistics) Running() bool {
	return s.wrap.running == 1
}

// RunningSince returns time since the packet filter is enabled
func (s Statistics) RunningSince() time.Time {
	return time.Unix(int64(s.wrap.since), 0)
}

// States num states in the packet filter
func (s Statistics) States() int {
	return int(s.wrap.states)
}

// SourceNodes num source nodes in the packet filter
func (s Statistics) SourceNodes() int {
	return int(s.wrap.src_nodes)
}

// Debug returns debug mode enabdled
func (s Statistics) Debug() DebugMode {
	return DebugMode(s.wrap.debug)
}

// HostID returns the ID of the host
func (s Statistics) HostID() uint32 {
	return uint32(s.wrap.hostid)
}

// Interface return the name of the interface if any (otherwise empty string)
func (s Statistics) Interface() string {
	return C.GoString(&s.wrap.ifname[0])
}

// ChecksumMD5 of the statistics
func (s Statistics) ChecksumMD5() []byte {
	sum := make([]byte, int(C.PF_MD5_DIGEST_LENGTH))
	for i := 0; i < int(C.PF_MD5_DIGEST_LENGTH); i++ {
		sum[i] = byte(s.wrap.pf_chksum[i])
	}
	return sum
}

/* Reasons code for passing/dropping a packet */

// ReasonMatch num of explicit match of a rule
func (s Statistics) ReasonMatch() uint64 {
	return uint64(s.wrap.counters[C.PFRES_MATCH])
}

// ReasonBadOffset num of bad offset for pull_hdr
func (s Statistics) ReasonBadOffset() uint64 {
	return uint64(s.wrap.counters[C.PFRES_BADOFF])
}

// ReasonFragment num dropping following fragment
func (s Statistics) ReasonFragment() uint64 {
	return uint64(s.wrap.counters[C.PFRES_FRAG])
}

// ReasonShort num dropping short packet
func (s Statistics) ReasonShort() uint64 {
	return uint64(s.wrap.counters[C.PFRES_SHORT])
}

// ReasonNormalizer num dropping by normalizer
func (s Statistics) ReasonNormalizer() uint64 {
	return uint64(s.wrap.counters[C.PFRES_NORM])
}

// ReasonMemory num dropped die to lacking mem
func (s Statistics) ReasonMemory() uint64 {
	return uint64(s.wrap.counters[C.PFRES_MEMORY])
}

// ReasonBadTimestamp num of bad TCP Timestamp (RFC1323)
func (s Statistics) ReasonBadTimestamp() uint64 {
	return uint64(s.wrap.counters[C.PFRES_TS])
}

// ReasonCongestion num of congestion of ipintrq
func (s Statistics) ReasonCongestion() uint64 {
	return uint64(s.wrap.counters[C.PFRES_CONGEST])
}

// ReasonIPOption num IP option
func (s Statistics) ReasonIPOption() uint64 {
	return uint64(s.wrap.counters[C.PFRES_IPOPTIONS])
}

// ReasonProtocolChecksum num protocol checksum invalid
func (s Statistics) ReasonProtocolChecksum() uint64 {
	return uint64(s.wrap.counters[C.PFRES_PROTCKSUM])
}

// ReasonBadState num of state mismatch
func (s Statistics) ReasonBadState() uint64 {
	return uint64(s.wrap.counters[C.PFRES_BADSTATE])
}

// ReasonStateInsertion num of state insertion failure
func (s Statistics) ReasonStateInsertion() uint64 {
	return uint64(s.wrap.counters[C.PFRES_STATEINS])
}

// ReasonMaxStates num of state limit
func (s Statistics) ReasonMaxStates() uint64 {
	return uint64(s.wrap.counters[C.PFRES_MAXSTATES])
}

// ReasonSourceLimit num of source node/conn limit
func (s Statistics) ReasonSourceLimit() uint64 {
	return uint64(s.wrap.counters[C.PFRES_SRCLIMIT])
}

// ReasonSynProxy num SYN proxy
func (s Statistics) ReasonSynProxy() uint64 {
	return uint64(s.wrap.counters[C.PFRES_SYNPROXY])
}

// ReasonMapFailed num pf_map_addr() failed
// Does not exist on Darwin:
/*
func (s Statistics) ReasonMapFailed() uint64 {
	return uint64(s.wrap.counters[C.PFRES_MAPFAILED])
}
*/

/* Counters for other things we want to keep track of */

// CounterStates num states
func (s Statistics) CounterStates() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_STATES])
}

// CounterSrcStates max src states
func (s Statistics) CounterSrcStates() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_SRCSTATES])
}

// CounterSrcNodes max src nodes
func (s Statistics) CounterSrcNodes() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_SRCNODES])
}

// CounterSrcConn max src conn
func (s Statistics) CounterSrcConn() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_SRCCONN])
}

// CounterSrcConnRate max src conn rate
func (s Statistics) CounterSrcConnRate() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_SRCCONNRATE])
}

// CounterOverloadTable entry added to overload table
func (s Statistics) CounterOverloadTable() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_OVERLOAD_TABLE])
}

// CounterOverloadFlush state entries flushed
func (s Statistics) CounterOverloadFlush() uint64 {
	return uint64(s.wrap.lcounters[C.LCNT_OVERLOAD_FLUSH])
}

/* state operation counters */

// CounterStateSearch num state search
func (s Statistics) CounterStateSearch() uint64 {
	return uint64(s.wrap.fcounters[C.FCNT_STATE_SEARCH])
}

// CounterStateInsert num state insert
func (s Statistics) CounterStateInsert() uint64 {
	return uint64(s.wrap.fcounters[C.FCNT_STATE_INSERT])
}

// CounterStateRemovals num state insert
func (s Statistics) CounterStateRemovals() uint64 {
	return uint64(s.wrap.fcounters[C.FCNT_STATE_REMOVALS])
}

/* src_node operation counters */

// CounterNodeSearch num state search
func (s Statistics) CounterNodeSearch() uint64 {
	return uint64(s.wrap.scounters[C.SCNT_SRC_NODE_SEARCH])
}

// CounterNodeInsert num state insert
func (s Statistics) CounterNodeInsert() uint64 {
	return uint64(s.wrap.scounters[C.SCNT_SRC_NODE_INSERT])
}

// CounterNodeRemovals num state insert
func (s Statistics) CounterNodeRemovals() uint64 {
	return uint64(s.wrap.scounters[C.SCNT_SRC_NODE_REMOVALS])
}

func (s Statistics) String() string {
	dump := []struct {
		name  string
		value uint64
	}{
		{"match", s.ReasonMatch()},
		{"bad-offset", s.ReasonBadOffset()},
		{"fragment", s.ReasonFragment()},
		{"short", s.ReasonShort()},
		{"normalize", s.ReasonNormalizer()},
		{"memory", s.ReasonMemory()},
		{"bad-timestamp", s.ReasonBadTimestamp()},
		{"congestion", s.ReasonCongestion()},
		{"ip-option", s.ReasonIPOption()},
		{"proto-cksum", s.ReasonProtocolChecksum()},
		{"state-mismatch", s.ReasonBadState()},
		{"state-insert", s.ReasonStateInsertion()},
		{"state-limit", s.ReasonMaxStates()},
		{"src-limit", s.ReasonSourceLimit()},
		{"synproxy", s.ReasonSynProxy()},
// Does not exist on Darwin: //		{"map-failed", s.ReasonMapFailed()},

		{"max-states-per-rule", s.CounterStates()},
		{"max-src-states", s.CounterSrcStates()},
		{"max-src-nodes", s.CounterSrcNodes()},
		{"max-src-conn", s.CounterSrcConn()},
		{"max-src-conn-rate", s.CounterSrcConnRate()},
		{"overload-table-insertion", s.CounterOverloadTable()},
		{"overload-flush-states", s.CounterOverloadFlush()},

		{"counter-state-search", s.CounterStateSearch()},
		{"counter-state-insert", s.CounterStateInsert()},
		{"counter-state-removals", s.CounterStateRemovals()},

		{"counter-node-search", s.CounterNodeSearch()},
		{"counter-node-insert", s.CounterNodeInsert()},
		{"counter-node-removals", s.CounterNodeRemovals()},
	}
	list := make([]string, 0, len(dump)+11)

	list = append(list, fmt.Sprintf("running: %v", s.Running()))
	list = append(list, fmt.Sprintf("states: %d", s.States()))
	list = append(list, fmt.Sprintf("src-nodes: %d", s.SourceNodes()))
	list = append(list, fmt.Sprintf("since: '%s'", s.RunningSince()))
	list = append(list, fmt.Sprintf("debug: %s", s.Debug()))
	list = append(list, fmt.Sprintf("hostid: %d", s.HostID()))
	list = append(list, fmt.Sprintf("interface: '%s'", s.Interface()))
	list = append(list, fmt.Sprintf("interface-bytes: %+v", s.Bytes()))
	list = append(list, fmt.Sprintf("interface-packets-pass: %+v", s.PacketsPass()))
	list = append(list, fmt.Sprintf("interface-packets-drop: %+v", s.PacketsDrop()))
	list = append(list, fmt.Sprintf("checksum: %+v", s.ChecksumMD5()))

	for _, line := range dump {
		list = append(list, fmt.Sprintf("%s: %d", line.name, line.value))
	}

	return strings.Join(list, " ")
}
