package pf

import (
	"fmt"
)

// #include <net/if.h>
// #include <net/pfvar.h>
import "C"

// AddressFamily that should be filtered by pf (inet / inet6)
type AddressFamily uint8

const (
	// AddressFamilyAny Any matches any address family
	AddressFamilyAny AddressFamily = 0
	// AddressFamilyInet IPv4
	AddressFamilyInet AddressFamily = C.AF_INET
	// AddressFamilyInet6 IPv6
	AddressFamilyInet6 AddressFamily = C.AF_INET6
)

func (af AddressFamily) String() string {
	switch af {
	case AddressFamilyAny:
		return "any"
	case AddressFamilyInet:
		return "inet"
	case AddressFamilyInet6:
		return "inet6"
	default:
		return fmt.Sprintf("AddressFamily(%d)", af)
	}
}
