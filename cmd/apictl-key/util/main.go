package util

import (
	"net"
	"net/http"
	"strings"
)

func ExtractRequesterIP(r *http.Request) string {
	xForwardedFor := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	if len(xForwardedFor) > 0 && net.ParseIP(xForwardedFor[0]) != nil {
		return xForwardedFor[0]
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
