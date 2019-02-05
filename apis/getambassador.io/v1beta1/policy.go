package v1

import (
	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/gobwas/glob"
)

type Policy struct {
	Metadata *metav1.ObjectMeta `json:"metadata"`
	Spec     *PolicySpec        `json:"spec"`
}

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

// register //////////////////////////////////////////////////////////

// Required to implement k8s.Resource
func (crd *Policy) GetMetadata() *metav1.ObjectMeta {
	return crd.Metadata
}

type PolicyList struct {
	Metadata *metav1.ListMeta `json:"metadata"`
	Items    []*Policy        `json:"items"`
}

// Require for PolicyList to implement k8s.ResourceList
func (crdl *PolicyList) GetMetadata() *metav1.ListMeta {
	return crdl.Metadata
}

func init() {
	k8s.Register("getambassador.io", "v1beta1", "policies", true, &Policy{})
	k8s.RegisterList("getambassador.io", "v1beta1", "policies", true, &PolicyList{})
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
