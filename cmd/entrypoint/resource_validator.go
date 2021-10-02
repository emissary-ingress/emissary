package entrypoint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/datawire/ambassador/v2/pkg/kates"
)

type resourceValidator struct {
	invalid        map[string]*kates.Unstructured
	katesValidator *kates.Validator
}

func newResourceValidator() (*resourceValidator, error) {
	crdFile, err := findCRDFile()
	if err != nil {
		return nil, err
	}
	crdYAML, err := ioutil.ReadAll(crdFile)
	if err != nil {
		_ = crdFile.Close()
		return nil, err
	}
	if err := crdFile.Close(); err != nil {
		return nil, err
	}

	crdObjs, err := kates.ParseManifests(string(crdYAML))
	if err != nil {
		return nil, err
	}
	katesValidator, err := kates.NewValidator(nil, crdObjs)
	if err != nil {
		return nil, err
	}

	return &resourceValidator{
		katesValidator: katesValidator,
		invalid:        map[string]*kates.Unstructured{},
	}, nil
}

func (v *resourceValidator) isValid(ctx context.Context, un *kates.Unstructured) bool {
	key := string(un.GetUID())
	err := v.katesValidator.Validate(ctx, un)
	if err != nil {
		fmt.Printf("validation error: %s %s/%s -- %s\n", un.GetKind(), un.GetNamespace(), un.GetName(), err.Error())
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

func findCRDFile() (*os.File, error) {
	candidateFilepaths := []string{
		"/opt/ambassador/etc/crds.yaml",
		"manifests/emissary/emissary-crds.yaml",
		"ambassador/manifests/emissary/emissary-crds.yaml",
		"../../manifests/emissary/emissary-crds.yaml",
	}
	for _, filepath := range candidateFilepaths {
		file, err := os.Open(filepath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return file, nil
	}

	return nil, fmt.Errorf("couldn't find CRDs at any of the following locations: %q", candidateFilepaths)
}
