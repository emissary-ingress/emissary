package gateway

import (
	"fmt"

	"github.com/datawire/ambassador/v2/pkg/kates"
)

type Source interface {
	Location() string
}

type k8sSource struct {
	resource kates.Object
}

func (s *k8sSource) Location() string {
	return fmt.Sprintf("%s %s.%s", s.resource.GetObjectKind().GroupVersionKind().Kind, s.resource.GetName(), s.resource.GetNamespace())
}

func SourceFromResource(resource kates.Object) Source {
	return &k8sSource{resource}
}

type patternSource struct {
	pattern string
	args    []interface{}
}

func Sourcef(pattern string, args ...interface{}) Source {
	return &patternSource{pattern, args}
}

func (c *patternSource) Location() string {
	var args []interface{}
	for _, a := range c.args {
		switch s := a.(type) {
		case Source:
			args = append(args, s.Location())
		default:
			args = append(args, a)
		}
	}
	return fmt.Sprintf(c.pattern, args...)
}
