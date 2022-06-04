package emissaryutil

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// schemeChars mimics Python `from urllib.parse import scheme_chars`.
const schemeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+-."

// wouldConfuseURLParse mimics `python/ambassador/ir/irbasemapping.py:would_confuse_urlparse()`.
// Please keep them in-sync.
func wouldConfuseURLParse(url string) bool {
	if strings.Contains(url, ":") && strings.HasPrefix(strings.TrimLeft(url, schemeChars), "://") {
		// has a scheme
		return false
	}
	if strings.HasPrefix(url, "//") {
		// does not have a scheme, but has the "//" URL authority marker
		return false
	}
	return true
}

type GlobalResolverConfig interface {
	AmbassadorNamespace() string
	UseAmbassadorNamespaceForServiceResolution() bool
}

// ParseServiceName mimics the first half of
// `python/ambassador/ir/irbasemapping.py:normalize_service_name()`.  Please keep them in-sync.
func ParseServiceName(svcStr string) (scheme, hostname string, port uint16, err error) {
	origSvcStr := svcStr
	if wouldConfuseURLParse(svcStr) {
		svcStr = "//" + svcStr
	}
	parsed, err := url.Parse(svcStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("service %q: %w", origSvcStr, err)
	}
	scheme = parsed.Scheme
	hostname = parsed.Hostname()
	portStr := parsed.Port()
	if portStr != "" {
		// Use net.SplitHostPort because does validation that we want; compared to
		// net/url.URL.{Hostname,Port}(), which do the same splitting but not the
		// validation.
		hostname, portStr, err = net.SplitHostPort(parsed.Host)
		if err != nil {
			return "", "", 0, fmt.Errorf("service %q: %w", origSvcStr, err)
		}
	}
	if hostname == "" {
		return "", "", 0, fmt.Errorf("service %q: address %s: no hostname", origSvcStr, parsed.Host)
	}
	var port64 uint64
	if portStr != "" {
		port64, err = strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return "", "", 0, fmt.Errorf("service %q: port %s: %w", origSvcStr, portStr, err)
		}
	}
	return scheme, hostname, uint16(port64), nil
}

// NormalizeServiceName mimics `python/ambassador/ir/irbasemapping.py:normalize_service_name()`.
// Please keep them in-sync.
func NormalizeServiceName(ir GlobalResolverConfig, svcStr, mappingNamespace, resolverKind string) (string, error) {
	scheme, hostname, port, err := ParseServiceName(svcStr)
	if err != nil {
		return "", err
	}

	// Consul Resolvers don't allow service names to include subdomains, but
	// Kubernetes Resolvers _require_ subdomains to correctly handle namespaces.
	wantQualified := !ir.UseAmbassadorNamespaceForServiceResolution() && strings.HasPrefix(resolverKind, "Kubernetes")

	isQualified := strings.ContainsAny(hostname, ".:") || hostname == "localhost"

	if mappingNamespace != "" && mappingNamespace != ir.AmbassadorNamespace() && wantQualified && !isQualified {
		hostname += "." + mappingNamespace
	}

	ret := url.PathEscape(hostname)
	if strings.Contains(ret, ":") {
		ret = "[" + ret + "]"
	}
	if scheme != "" {
		ret = scheme + "://" + ret
	}
	if port != 0 {
		ret = fmt.Sprintf("%s:%d", ret, port)
	}
	return ret, nil
}
