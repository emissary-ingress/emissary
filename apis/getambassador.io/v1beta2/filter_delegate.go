package v1

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Based off of ambassador/schemas/v1/AuthService.schema
type FilterDelegate struct {
	AuthService string `json:"auth_service"` // required
	PathPrefix  string `json:"path_prefix"`
	// As an experimental feature, Ambassador supports setting
	// "tls" to the name of a TLS context (rather than setting it
	// to a boolean).  Ambassador Pro Delegate filters do NOT
	// support that; if present, the "tls" field must be a
	// boolean.
	TLS bool `json:"tls"`

	Proto                       string        `json:"proto"` // either "http" or "grpc"
	AllowRequestBody            bool          `json:"allow_request_body"`
	RawTimeout                  int64         `json:"timeout_ms"`
	Timeout                     time.Duration `json:"-"`
	AllowedRequestHeaders       []string      `json:"allowed_request_headers"`
	AllowedAuthorizationHeaders []string      `json:"allowed_authorization_headers"`
}

var (
	alwaysAllowedRequestHeaders = []string{
		"authorization",
		"cookie",
		"from",
		"proxy-authorization",
		"user-agent",
		"x-forwarded-for",
		"x-forwarded-host",
		"x-forwarded-proto",
	}
	alwaysAllowedAuthorizationHeaders = []string{
		"location",
		"authorization",
		"proxy-authenticate",
		"set-cookie",
		"www-authenticate",
	}
)

func normalizeUnion(a, b []string) []string {
	set := make(map[string]struct{}, len(a)+len(b))
	for _, s := range a {
		set[http.CanonicalHeaderKey(s)] = struct{}{}
	}
	for _, s := range b {
		set[http.CanonicalHeaderKey(s)] = struct{}{}
	}
	ret := make([]string, 0, len(set))
	for s := range set {
		ret = append(ret, s)
	}
	sort.Strings(ret)
	return ret
}

func (m *FilterDelegate) Validate() error {
	// Fill in defaults
	if m.Proto == "" {
		m.Proto = "http"
	}
	if m.RawTimeout == 0 {
		m.RawTimeout = 5000
	}
	m.AllowedRequestHeaders = normalizeUnion(m.AllowedRequestHeaders, alwaysAllowedRequestHeaders)
	m.AllowedAuthorizationHeaders = normalizeUnion(m.AllowedAuthorizationHeaders, alwaysAllowedAuthorizationHeaders)

	// Validate

	// How sure am I that no non-ASCII Unicode characters map to
	// ASCII "h"/"p"/"s"/"t" when passed through
	// strings.ToLower()?
	if strings.HasPrefix(strings.ToLower(m.AuthService), "https://") {
		m.TLS = true
		m.AuthService = m.AuthService[len("https://"):]
	} else if strings.HasPrefix(strings.ToLower(m.AuthService), "http://") {
		m.AuthService = m.AuthService[len("http://"):]
	}

	if m.Proto != "grpc" && m.Proto != "http" {
		return errors.Errorf("invalid proto, must be \"http\" or \"grpc\": %q", m.Proto)
	}
	m.Timeout = time.Duration(m.RawTimeout) * time.Millisecond

	return nil
}
