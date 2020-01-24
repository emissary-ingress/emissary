package v1

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Based off of ambassador/schemas/v1/AuthService.schema
type FilterExternal struct {
	AuthService string `json:"auth_service"` // required
	PathPrefix  string `json:"path_prefix"`
	// As an experimental feature, Ambassador supports setting
	// "tls" to the name of a TLS context (rather than setting it
	// to a boolean).  Ambassador Pro External filters do NOT
	// support that; if present, the "tls" field must be a
	// boolean.
	TLS bool `json:"tls"`

	Proto                       string        `json:"proto"` // either "http" or "grpc"
	RawTimeout                  int64         `json:"timeout_ms"`
	Timeout                     time.Duration `json:"-"`
	AllowedRequestHeaders       []string      `json:"allowed_request_headers"`
	AllowedAuthorizationHeaders []string      `json:"allowed_authorization_headers"`
	DeprecatedAllowRequestBody  *bool         `json:"allow_request_body"` // deprecated in favor of include_body
	AddLinkerdHeaders           *bool         `json:"add_linkerd_headers"`
	IncludeBody                 *IncludeBody  `json:"include_body"`
	StatusOnError               struct {
		Code int `json:"code"`
	} `json:"status_on_error"`
	FailureModeAllow bool `json:"failure_mode_allow"`
}

type IncludeBody struct {
	MaxBytes     int  `json:"max_bytes"` // required
	AllowPartial bool `json:"max_bytes"` // required
}

// Keep these in-sync with v2listener.py
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

func (m *FilterExternal) Validate() error {
	// Convert deprecated fields
	if m.DeprecatedAllowRequestBody != nil {
		if m.IncludeBody != nil {
			return errors.New("it is invalid to set both \"allow_request\" and \"include_body\"; \"allow_request\" is deprecated and should be replaced by \"include_body\"")
		}
		if *m.DeprecatedAllowRequestBody {
			m.IncludeBody = &IncludeBody{
				// Keep this in-sync with v2listener.py
				MaxBytes:     4096,
				AllowPartial: true,
			}
		}
		m.DeprecatedAllowRequestBody = nil
	}

	// Fill in defaults
	if m.Proto == "" {
		m.Proto = "http"
	}
	if m.RawTimeout == 0 {
		m.RawTimeout = 5000
	}
	m.AllowedRequestHeaders = normalizeUnion(m.AllowedRequestHeaders, alwaysAllowedRequestHeaders)
	m.AllowedAuthorizationHeaders = normalizeUnion(m.AllowedAuthorizationHeaders, alwaysAllowedAuthorizationHeaders)
	if m.StatusOnError.Code == 0 {
		m.StatusOnError.Code = http.StatusForbidden
	}
	if m.AddLinkerdHeaders == nil {
		// TODO(lukeshu): Per irauth.py, this default should be
		// `ir.ambassador_module.get('add_linkerd_headers', False)`.
		// But getting that info to here is a pain, so for now I'm just
		// having the default be `false`, and documenting this as a
		// difference between AuthServices and External Filters.
		value := false
		m.AddLinkerdHeaders = &value
	}

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
