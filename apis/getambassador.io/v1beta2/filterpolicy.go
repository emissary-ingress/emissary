package v1

import (
	"github.com/gobwas/glob"
)

type FilterPolicySpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`
	Rules        []Rule       `json:"rules"`
}

// Rule defines authorization rules object.
type Rule struct {
	Host   string          `json:"host"`
	Path   string          `json:"path"`
	Public bool            `json:"public"`
	Filter FilterReference `json:"filter"`
}

type FilterReference struct {
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Arguments interface{} `json:"arguments"`
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

func (r *Rule) Validate(namespace string) error {
	if r.Filter.Namespace == "" {
		r.Filter.Namespace = namespace
	}
	return nil
}
