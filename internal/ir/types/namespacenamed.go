package types

// NamespacedName comprises a resource name, with a mandatory namespace,
// rendered as "<namespace>/<name>".  Being a type captures intent and
// helps make sure namespaced names and non-namespaced names
// do not get conflated in code.
//
// This was modified k8s/apimacginary version for two reasons:
// 1. so that we can add json tags for decoding tests
// 2. debug friendly when printing interal representations
type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

const (
	Separator = '/'
)

// String returns the general purpose string representation
func (n NamespacedName) String() string {
	return n.Namespace + string(Separator) + n.Name
}
