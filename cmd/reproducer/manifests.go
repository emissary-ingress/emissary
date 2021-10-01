package reproducer

import (
	"bytes"
	"fmt"

	"github.com/datawire/ambassador/v2/pkg/kates/k8sresourcetypes"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func marshalManifests(manifests []*k8sresourcetypes.Unstructured) ([]byte, error) {
	result := bytes.NewBuffer(nil)

	pfx := ""
	for _, p := range manifests {
		marshalled, err := yaml.Marshal(p)
		if err != nil {
			return nil, errors.Wrapf(err, "marshalling processed")
		}
		result.WriteString(fmt.Sprintf("%s---\n%s", pfx, string(marshalled)))
		pfx = "\n"
	}

	return result.Bytes(), nil
}
