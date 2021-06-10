package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/datawire/ambassador/v2/pkg/kates"
)

type resourceValidator struct {
	invalid        map[string]*kates.Unstructured
	katesValidator *kates.Validator
}

func newResourceValidator() *resourceValidator {
	crdYAML, err := ioutil.ReadFile(findCRDFilename())
	if err != nil {
		panic(err)
	}

	crdObjs, err := kates.ParseManifests(string(crdYAML))
	if err != nil {
		panic(err)
	}
	katesValidator, err := kates.NewValidator(nil, crdObjs)
	if err != nil {
		panic(err)
	}

	return &resourceValidator{
		katesValidator: katesValidator,
		invalid:        map[string]*kates.Unstructured{},
	}
}

func (v *resourceValidator) isValid(ctx context.Context, un *kates.Unstructured) bool {
	key := string(un.GetUID())
	err := v.katesValidator.Validate(ctx, un)
	if err != nil {
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

func findCRDFilename() string {
	searchPath := []string{
		"/opt/ambassador/etc/crds.yaml",
		"manifests/emissary/emissary-crds.yaml",
		"ambassador/manifests/emissary/emissary-crds.yaml",
		"../../manifests/emissary/emissary-crds.yaml",
	}

	for _, candidate := range searchPath {
		if fileExists(candidate) {
			return candidate
		}
	}

	panic(fmt.Sprintf("couldn't find CRDs at any of the following locations: %s", strings.Join(searchPath, ", ")))
}
