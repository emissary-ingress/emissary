package v1

import (
	"github.com/gobwas/glob"
)

type PolicySpec struct {
	Rules []Rule `json:"rules"`
}

// Rule defines authorization rules object.
type Rule struct {
	Host   string          `json:"host"`
	Path   string          `json:"path"`
	Public bool            `json:"public"`
	Scope  string          `json:"scope"`
	Scopes map[string]bool `json:"-"` // is calculated from Scope
}

//////////////////////////////////////////////////////////////////////

// MatchHTTPHeaders return true if rules matches the supplied hostname and path.
func (r Rule) MatchHTTPHeaders(host, path string) bool {
	return match(r.Host, host) && match(r.Path, path)
}

func match(pattern, input string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		return false
	}

	return g.Match(input)
}

const (
	// DefaultScope is normally used for when no rule has matched the request path or host.
	DefaultScope = "offline_access"
)

// MatchScope return true if rule scope.
func (r Rule) MatchScope(scope string) bool {
	return r.Scope == DefaultScope || r.Scopes[scope]
}
