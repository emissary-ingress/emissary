package entrypoint

import (
	"context"

	getambassadorio "github.com/datawire/ambassador/v2/pkg/api/getambassador.io"
	"github.com/datawire/ambassador/v2/pkg/kates"
	"github.com/datawire/dlib/dlog"
)

type resourceValidator struct {
	invalid        map[string]*kates.Unstructured
	katesValidator *kates.Validator
}

func newResourceValidator() (*resourceValidator, error) {
	return &resourceValidator{
		katesValidator: getambassadorio.NewValidator(),
		invalid:        map[string]*kates.Unstructured{},
	}, nil
}

func (v *resourceValidator) isValid(ctx context.Context, un *kates.Unstructured) bool {
	key := string(un.GetUID())
	err := v.katesValidator.Validate(ctx, un)
	if err != nil {
		dlog.Errorf(ctx, "validation error: %s %s/%s -- %s", un.GetKind(), un.GetNamespace(), un.GetName(), err.Error())
		copy := un.DeepCopy()
		copy.Object["errors"] = err.Error()
		v.invalid[key] = copy
		return false
	} else {
		delete(v.invalid, key)
		return true
	}
}

func (v *resourceValidator) getInvalid() []*kates.Unstructured {
	var result []*kates.Unstructured
	for _, inv := range v.invalid {
		result = append(result, inv)
	}
	return result
}
