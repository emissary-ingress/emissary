package getambassadorio

import (
	_ "embed"

	"k8s.io/apimachinery/pkg/runtime"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v2"
	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/datawire/ambassador/v2/pkg/kates"
)

func AddToScheme(scheme *runtime.Scheme) error {
	if err := v2.AddToScheme(scheme); err != nil {
		return err
	}
	if err := v3alpha1.AddToScheme(scheme); err != nil {
		return err
	}
	return nil
}

func BuildScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	runtimeutil.Must(AddToScheme(scheme))
	return scheme
}

//go:embed crds.yaml
var crdYAML string

func NewValidator() *kates.Validator {
	crdObjs, err := kates.ParseManifests(crdYAML)
	runtimeutil.Must(err)
	validator, err := kates.NewValidator(nil, crdObjs)
	runtimeutil.Must(err)
	return validator
}
