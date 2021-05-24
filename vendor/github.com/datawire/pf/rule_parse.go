package pf

import (
	"fmt"
	"strconv"
	"strings"
	"text/scanner"
)

// #include <net/if.h>
// #include <net/pfvar.h>
// #include "helpers.h"
import "C"

// ParseSource sets the source ip (inet and inet6) based on the
// passed strings, if parsing failes err is returned
func (r *Rule) ParseSource(src, port string, neg bool) error {
	err := parsePort(&r.wrap.rule.src, port)
	if err != nil {
		return err
	}

	err = parseAddress(&r.wrap.rule.src, src, neg)
	if err != nil {
		return err
	}

	// determine if it is IPv6 or IPv4
	if strings.ContainsRune(src, ':') {
		r.SetAddressFamily(AddressFamilyInet6)
	} else {
		r.SetAddressFamily(AddressFamilyInet)
	}

	return nil
}

// ParseDestination sets the destination (inet and inet6) based on
// the passed strings, if parsing failes err returned
func (r *Rule) ParseDestination(dst, port string, neg bool) error {
	err := parsePort(&r.wrap.rule.dst, port)
	if err != nil {
		return err
	}

	return parseAddress(&r.wrap.rule.dst, dst, neg)
}

// parseAddress parses the passed string into the addr structure
func parseAddress(addr *C.struct_pf_rule_addr, address string, negative bool) error {
	if negative {
		addr.neg = 1
	}

	a := Address{wrap: &addr.addr}
	return a.ParseCIDR(address)
}

// parsePort parses the passed port into the address structure port section
func parsePort(addr *C.struct_pf_rule_addr, port string) error {
	s := scanner.Scanner{}
	s.Init(strings.NewReader(port))
	C.set_addr_port_op(addr, C.PF_OP_NONE)

	var tok rune
	curPort := 0
	for tok != scanner.EOF {
		tok = s.Scan()
		switch tok {
		case -3:
			if curPort >= 2 {
				return fmt.Errorf("Unexpected 3rd number in port range: %s",
					s.TokenText())
			}
			val, err := strconv.ParseUint(s.TokenText(), 10, 16)
			if err != nil {
				return fmt.Errorf("Number not allowed in port range: %s",
					s.TokenText())
			}
			if val < 0 {
				return fmt.Errorf("Port number can't be negative: %d", val)
			}

			C.set_addr_port(addr, C.int(curPort), C.u_int16_t(C.htons_f(C.uint16_t(val))))
			curPort++

			// if it is the first number and after there is nothing, set none
		case ':':
			C.set_addr_port_op(addr, C.PF_OP_RRG)
		case '!':
			if curPort != 0 {
				return fmt.Errorf("Unexpected number before '!'")
			}
			if s.Peek() == '=' {
				s.Next() // consume
				C.set_addr_port_op(addr, C.PF_OP_NE)
			} else {
				return fmt.Errorf("Expected '=' after '!'")
			}
		case '<':
			if s.Peek() == '>' {
				s.Next() // consume
				C.set_addr_port_op(addr, C.PF_OP_XRG)
			} else if s.Peek() == '=' {
				s.Next() // consume
				C.set_addr_port_op(addr, C.PF_OP_LE)
			} else if s.Peek() >= '0' && s.Peek() <= '9' { // int
				// next is port number continue
				C.set_addr_port_op(addr, C.PF_OP_LT)
			} else {
				return fmt.Errorf("Expected port number not '%c'", s.Peek())
			}
		case '>':
			if s.Peek() == '<' {
				s.Next() // consume
				C.set_addr_port_op(addr, C.PF_OP_IRG)
			} else if s.Peek() == '=' {
				s.Next() // consume
				C.set_addr_port_op(addr, C.PF_OP_GE)
			} else if s.Peek() >= '0' && s.Peek() <= '9' { // int
				// next is port number continue
				C.set_addr_port_op(addr, C.PF_OP_GT)
			} else {
				return fmt.Errorf("Expected port number not '%c'", s.Peek())
			}
		case -1:
			// if no operation was set
			if curPort == 1 && C.get_addr_port_op(addr) == C.PF_OP_NONE { // one port
				C.set_addr_port_op(addr, C.PF_OP_EQ)
			}
		default:
			return fmt.Errorf("Unexpected char '%c'", s.Peek())
		}
	}
	return nil
}
