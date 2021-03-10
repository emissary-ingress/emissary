package acp

import (
	"net"
)

// HostPortIsLocal returns true IFF the host:port string from a URL refers to the
// local host. The comparison is simple: if it's "localhost" or "127.0.0.1" or "::1",
// it refers to the local host.
//
// Note that HostPortIsLocal _requires_ the ":port" part, because net.SplitHostPort
// requires it, and because the whole point here is that IPv6 is a pain. Sigh.
func HostPortIsLocal(hostport string) bool {
	// Split out the host part...
	host, _, err := net.SplitHostPort(hostport)

	// If something went wrong, it ain't local.
	if err != nil {
		// log.Println(fmt.Errorf("HostPortIsLocal: %s got error %v", hostport, err))
		return false
	}

	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
