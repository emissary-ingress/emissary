package entrypoint

import (
	"context"
	"encoding/json"

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
	rewrite := ""
	rewrite_is_present := false

	kind := un.GetKind()
	ns := un.GetNamespace()
	name := un.GetName()

	var spec map[string]interface{}

	if kind == "Mapping" {
		spec = un.Object["spec"].(map[string]interface{})
		rewrite, rewrite_is_present = spec["rewrite"].(string)

		if rewrite_is_present {
			dlog.Debugf(ctx, "validating Mapping %s/%s: Spec.Rewrite is '%s'", ns, name, rewrite)

			if rewrite == "" {
				// This case currently screws up in validation, so we'll hack around it for
				// a moment.
				spec["rewrite"] = "&notset&"
			}
		} else {
			dlog.Debugf(ctx, "validating Mapping %s/%s: Spec.Rewrite is missing", ns, name)
		}

		jsonBytes, err := json.MarshalIndent(un, "", "  ")

		if err != nil {
			dlog.Debugf(ctx, "validating Mapping %s/%s: is unmarshalable?? %s", ns, name, err.Error())
		} else {
			dlog.Debugf(ctx, "validating Mapping %s/%s: is\n%s", ns, name, string(jsonBytes))
		}
	}

	err := v.katesValidator.Validate(ctx, un)

	// If we mucked with the rewrite, reset it.
	if kind == "Mapping" && rewrite_is_present && (rewrite == "") {
		spec["rewrite"] = ""
	}

	if err != nil {
		dlog.Errorf(ctx, "validation error: %s %s/%s -- %s", kind, ns, name, err.Error())
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
