package entrypoint

import (
	"context"

	"github.com/datawire/dlib/dlog"
	getambassadorio "github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io"
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
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
	err := v.katesValidator.Validate(ctx, un)

	if err != nil {
		dlog.Errorf(ctx, "validation error: %s %s/%s -- %s", un.GetKind(), un.GetNamespace(), un.GetName(), err.Error())
		v.addInvalid(ctx, un, err.Error())
		return false
	} else {
		v.removeInvalid(ctx, un)
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

// The addInvalid method adds a resource to the Validator's list of invalid
// resources.
func (v *resourceValidator) addInvalid(ctx context.Context, un *kates.Unstructured, errorMessage string) {
	key := string(un.GetUID())

	copy := un.DeepCopy()
	copy.Object["errors"] = errorMessage
	v.invalid[key] = copy
}

// The removeInvalid method removes a resource from the Validator's list of
// invalid resources.
func (v *resourceValidator) removeInvalid(ctx context.Context, un *kates.Unstructured) {
	key := string(un.GetUID())
	delete(v.invalid, key)
}
