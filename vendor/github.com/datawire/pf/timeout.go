package pf

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// Timeout configuration identifier
type Timeout int

const (
	// TimeoutTCPFirstPacket first tcp packet (default 2 min)
	TimeoutTCPFirstPacket Timeout = C.PFTM_TCP_FIRST_PACKET
	// TimeoutTCPOpening no response yet (default 30 sec)
	TimeoutTCPOpening Timeout = C.PFTM_TCP_OPENING
	// TimeoutTCPEstablished connection established (default 1 day)
	TimeoutTCPEstablished Timeout = C.PFTM_TCP_ESTABLISHED
	// TimeoutTCPClosing half closed connection (default 15 min)
	TimeoutTCPClosing Timeout = C.PFTM_TCP_CLOSING
	// TimeoutTCPFinWait got both FIN's (default 45 sec)
	TimeoutTCPFinWait Timeout = C.PFTM_TCP_FIN_WAIT
	// TimeoutTCPClosed got a RST (default 1 min 30 sec)
	TimeoutTCPClosed Timeout = C.PFTM_TCP_CLOSED
	// TimeoutUDPFirstPacket first udp packet (default 1 min)
	TimeoutUDPFirstPacket Timeout = C.PFTM_UDP_FIRST_PACKET
	// TimeoutUDPSingle unidirectional (default 30 sec)
	TimeoutUDPSingle Timeout = C.PFTM_UDP_SINGLE
	// TimeoutUDPMultiple bidirectional (default 1 min)
	TimeoutUDPMultiple Timeout = C.PFTM_UDP_MULTIPLE
	// TimeoutICMPFirstPacket first ICMP packet (default 20 sec)
	TimeoutICMPFirstPacket Timeout = C.PFTM_ICMP_FIRST_PACKET
	// TimeoutICMPErrorReply go error response (default 10 sec)
	TimeoutICMPErrorReply Timeout = C.PFTM_ICMP_ERROR_REPLY
	// TimeoutOtherFirstPacket first packet (default 1 min)
	TimeoutOtherFirstPacket Timeout = C.PFTM_OTHER_FIRST_PACKET
	// TimeoutOtherSingle unidirectional (default 30 sec)
	TimeoutOtherSingle Timeout = C.PFTM_OTHER_SINGLE
	// TimeoutOtherMultiple bidirectional (default 1 min)
	TimeoutOtherMultiple Timeout = C.PFTM_OTHER_MULTIPLE
	// TimeoutFragment fragment expire (default 30 sec)
	TimeoutFragment Timeout = C.PFTM_FRAG
	// TimeoutInterval expire interval (default 10 sec)
	TimeoutInterval Timeout = C.PFTM_INTERVAL
	// TimeoutAdaptiveStart adaptive start
	TimeoutAdaptiveStart Timeout = C.PFTM_ADAPTIVE_START
	// TimeoutAdaptiveEnd adaptive end
	TimeoutAdaptiveEnd Timeout = C.PFTM_ADAPTIVE_END
	// TimeoutSourceNode source tracking (default 0 sec)
	TimeoutSourceNode Timeout = C.PFTM_SRC_NODE
	// TimeoutTSDiff allowed TS diff (default 30 sec)
	TimeoutTSDiff Timeout = C.PFTM_TS_DIFF
	// TimeoutPurge purge
	TimeoutPurge Timeout = C.PFTM_PURGE
	// TimeoutUnlinked unlinked
	TimeoutUnlinked Timeout = C.PFTM_UNLINKED
)
