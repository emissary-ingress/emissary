package v1

import (
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
)

type FilterPolicySpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`
	Rules        []Rule       `json:"rules"`
}

// Rule defines authorization rules object.
type Rule struct {
	Host    string            `json:"host"`
	Path    string            `json:"path"`
	Filters []FilterReference `json:"filters"`
}

type FilterReference struct {
	Name            string              `json:"name"`
	Namespace       string              `json:"namespace"`
	OnDeny          string              `json:"onDeny"`
	OnAllow         string              `json:"onAllow"`
	IfRequestHeader HeaderFieldSelector `json:"ifRequestHeader"`
	Arguments       interface{}         `json:"arguments"`
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
	FilterActionContinue = "continue"
	FilterActionBreak    = "break"
)

func (r *Rule) Validate(namespace string) error {
	for i := range r.Filters {
		if r.Filters[i].Namespace == "" {
			r.Filters[i].Namespace = namespace
		}

		switch r.Filters[i].OnDeny {
		case "":
			r.Filters[i].OnDeny = FilterActionBreak
		case FilterActionContinue, FilterActionBreak:
			// do nothing
		default:
			return errors.Errorf("onDeny=%q is invalid; valid values are %q",
				r.Filters[i].OnDeny, []string{FilterActionContinue, FilterActionBreak})
		}

		switch r.Filters[i].OnAllow {
		case "":
			r.Filters[i].OnAllow = FilterActionContinue
		case FilterActionContinue, FilterActionBreak:
			// do nothing
		default:
			return errors.Errorf("onAllow=%q is invalid; valid values are %q",
				r.Filters[i].OnAllow, []string{FilterActionContinue, FilterActionBreak})
		}

		if err := r.Filters[i].IfRequestHeader.Validate(); err != nil {
			return errors.Wrap(err, "ifRequestHeader")
		}
	}
	return nil
}
