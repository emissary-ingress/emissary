package kates

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sync"

	apiextVInternal "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextV1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/kube-openapi/pkg/validation/validate"

	"github.com/datawire/dlib/derror"
)

// A Validator may be used in concert with a Client to perform
// validate of freeform jsonish data structures as kubernetes CRDs.
type Validator struct {
	client *Client
	static map[TypeMeta]*apiextVInternal.CustomResourceDefinition

	mutex      sync.Mutex
	validators map[TypeMeta]*validate.SchemaValidator
}

// The NewValidator constructor returns a *Validator that uses the
// provided *Client to fetch CustomResourceDefinitions from kubernetes
// on demand as needed to validate data passed to the Validator.Validate()
// method.
func NewValidator(client *Client, staticCRDs []Object) (*Validator, error) {
	if client == nil && len(staticCRDs) == 0 {
		return nil, errors.New("at least 1 client or static CRD must be provided")
	}

	static := make(map[TypeMeta]*apiextVInternal.CustomResourceDefinition, len(staticCRDs))
	for i, untypedCRD := range staticCRDs {
		var crd apiextVInternal.CustomResourceDefinition
		switch untypedCRD.GetObjectKind().GroupVersionKind() {
		case apiextV1beta1.SchemeGroupVersion.WithKind("CustomResourceDefinition"):
			var crdV1beta1 apiextV1beta1.CustomResourceDefinition
			if err := convert(untypedCRD, &crdV1beta1); err != nil {
				return nil, fmt.Errorf("staticCRDs[%d]: %w", i, err)
			}
			apiextV1beta1.SetDefaults_CustomResourceDefinition(&crdV1beta1)
			if err := apiextV1beta1.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&crdV1beta1, &crd, nil); err != nil {
				return nil, fmt.Errorf("staticCRDs[%d]: %w", i, err)
			}
		case apiextV1.SchemeGroupVersion.WithKind("CustomResourceDefinition"):
			var crdV1 apiextV1.CustomResourceDefinition
			if err := convert(untypedCRD, &crdV1); err != nil {
				return nil, fmt.Errorf("staticCRDs[%d]: %w", i, err)
			}
			apiextV1.SetDefaults_CustomResourceDefinition(&crdV1)
			if err := apiextV1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(&crdV1, &crd, nil); err != nil {
				return nil, fmt.Errorf("staticCRDs[%d]: %w", i, err)
			}
		default:
			err := fmt.Errorf("unrecognized CRD GroupVersionKind: %v", untypedCRD.GetObjectKind().GroupVersionKind())
			return nil, fmt.Errorf("staticCRDs[%d]: %w", i, err)
		}
		for _, version := range crd.Spec.Versions {
			static[TypeMeta{
				APIVersion: crd.Spec.Group + "/" + version.Name,
				Kind:       crd.Spec.Names.Kind,
			}] = &crd
		}
	}

	return &Validator{
		client: client,
		static: static,

		validators: make(map[TypeMeta]*validate.SchemaValidator),
	}, nil
}

func (v *Validator) getCRD(ctx context.Context, tm TypeMeta) (*apiextVInternal.CustomResourceDefinition, error) {
	if crd, ok := v.static[tm]; ok {
		return crd, nil
	}
	if v.client != nil {
		mapping, err := v.client.mappingFor(tm.GroupVersionKind().GroupKind().String())
		if err != nil {
			return nil, err
		}
		crd := mapping.Resource.GroupResource().String()

		obj := &apiextV1.CustomResourceDefinition{
			TypeMeta: TypeMeta{
				Kind: "CustomResourceDefinition",
			},
			ObjectMeta: ObjectMeta{
				Name: crd,
			},
		}
		err = v.client.Get(ctx, obj, obj)
		if err != nil {
			if IsNotFound(err) {
				return nil, nil
			}

			return nil, err
		}

		var ret apiextVInternal.CustomResourceDefinition
		err = apiextV1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(obj, &ret, nil)
		if err != nil {
			return nil, err
		}
		return &ret, nil
	}
	return nil, nil
}

func (v *Validator) getValidator(ctx context.Context, tm TypeMeta) (*validate.SchemaValidator, error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	validator, ok := v.validators[tm]
	if !ok {
		crd, err := v.getCRD(ctx, tm)
		if err != nil {
			return nil, err
		}

		if crd != nil {
			if crd.Spec.Validation != nil {
				validator, _, err = validation.NewSchemaValidator(crd.Spec.Validation)
				if err != nil {
					return nil, err
				}
			} else {
				tmVersion := path.Base(tm.APIVersion)
				for _, version := range crd.Spec.Versions {
					if version.Name == tmVersion {
						validator, _, err = validation.NewSchemaValidator(version.Schema)
						if err != nil {
							return nil, err
						}
						break
					}
				}
			}
		}

		v.validators[tm] = validator // even if validator is nil; cache negative responses
	}
	return validator, nil
}

// The Validate method validates the supplied jsonish object as a
// kubernetes CRD instance.
//
// If the supplied object is *not* a CRD instance but instead a
// regular kubernetes instance, the Validate method will assume that
// the supplied object is valid.
//
// If the supplied object is not a valid kubernetes resource at all,
// the Validate method will return an error.
//
// Typically the Validate method will perform only local operations,
// however the first time an instance of a given Kind is supplied, the
// Validator needs to query the cluster to figure out if it is a CRD
// and if so to fetch the schema needed to perform validation. All
// subsequent Validate() calls for that Kind will be local.
func (v *Validator) Validate(ctx context.Context, resource interface{}) error {
	var tm TypeMeta
	err := convert(resource, &tm)
	if err != nil {
		return err
	}

	validator, err := v.getValidator(ctx, tm)
	if err != nil {
		return err
	}

	result := validator.Validate(resource)

	var errs derror.MultiError
	for _, e := range result.Errors {
		errs = append(errs, e)
	}

	for _, w := range result.Warnings {
		errs = append(errs, w)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
