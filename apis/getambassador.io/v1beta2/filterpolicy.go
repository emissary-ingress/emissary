package v1

import (
	"github.com/gobwas/glob"
	"github.com/pkg/errors"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FilterPolicy struct {
	*metaV1.TypeMeta   `json:",inline"`
	*metaV1.ObjectMeta `json:"metadata"`
	Spec               *FilterPolicySpec   `json:"spec"`
	Status             *FilterPolicyStatus `json:"status"`
}

type FilterPolicySpec struct {
	AmbassadorID AmbassadorID `json:"ambassador_id"`
	Rules        []Rule       `json:"rules"`
}

const (
	FilterPolicyState_OK           = "OK"
	FilterPolicyState_Error        = "Error"
	FilterPolicyState_PartialError = "PartialError"
)

type FilterPolicyStatus struct {
	State        string       `json:"state"`
	Reason       string       `json:"reason"`
	RuleStatuses []RuleStatus `json:"ruleStatuses"`
}

const (
	RuleState_OK    = "OK"
	RuleState_Error = "Error"
)

type RuleStatus struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
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

func (fp *FilterPolicy) Validate() error {
	if fp.Spec == nil {
		err := errors.New("spec must be set")
		fp.Spec = &FilterPolicySpec{}
		fp.Status = &FilterPolicyStatus{
			State:  FilterPolicyState_Error,
			Reason: err.Error(),
		}
		return err
	}

	ruleErrors := 0
	fp.Status = &FilterPolicyStatus{
		RuleStatuses: make([]RuleStatus, 0, len(fp.Spec.Rules)),
	}
	for _, rule := range fp.Spec.Rules {
		if err := rule.Validate(fp.GetNamespace()); err == nil {
			fp.Status.RuleStatuses = append(fp.Status.RuleStatuses, RuleStatus{
				State: RuleState_OK,
			})
		} else {
			fp.Status.RuleStatuses = append(fp.Status.RuleStatuses, RuleStatus{
				State:  RuleState_Error,
				Reason: err.Error(),
			})
			ruleErrors++
		}
	}

	if ruleErrors > 0 {
		err := errors.Errorf("%d of the rules in .spec.rules have errors", ruleErrors)
		fp.Status.State = FilterPolicyState_PartialError
		fp.Status.Reason = err.Error()
		return err
	}

	fp.Status.State = FilterPolicyState_OK
	return nil
}

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
