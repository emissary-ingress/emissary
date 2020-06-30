package kates

import (
	"context"
	"errors"
	"strings"
	"sync"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"

	"github.com/go-openapi/validate"
)

type Validator struct {
	client     *Client
	mutex      sync.Mutex
	validators map[string]*validate.SchemaValidator
}

func NewValidator(client *Client) *Validator {
	return &Validator{client: client, validators: make(map[string]*validate.SchemaValidator)}
}

func (v *Validator) getValidator(ctx context.Context, crd string) (*validate.SchemaValidator, error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	validator, ok := v.validators[crd]
	if !ok {
		obj := &CustomResourceDefinition{
			TypeMeta: TypeMeta{
				Kind: "CustomResourceDefinition",
			},
			ObjectMeta: ObjectMeta{
				Name: crd,
			},
		}
		err := v.client.Get(ctx, obj, obj)
		if err != nil {
			if IsNotFound(err) {
				v.validators[crd] = nil
				return nil, nil
			}

			return nil, err
		}

		extcrd := &apiextensions.CustomResourceDefinition{}
		err = v1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(obj, extcrd, nil)
		if err != nil {
			return nil, err
		}

		validator, _, err = validation.NewSchemaValidator(extcrd.Spec.Validation)
		if err != nil {
			return nil, err
		}

		v.validators[crd] = validator
	}
	return validator, nil
}

func (v *Validator) Validate(ctx context.Context, resource interface{}) error {
	var tm TypeMeta
	err := convert(resource, &tm)
	if err != nil {
		return err
	}

	mapping, err := v.client.mappingFor(tm.GroupVersionKind().GroupKind().String())
	if err != nil {
		return err
	}

	crd := mapping.Resource.GroupResource().String()

	validator, err := v.getValidator(ctx, crd)
	if err != nil {
		return err
	}

	result := validator.Validate(resource)

	var errs []error
	for _, e := range result.Errors {
		errs = append(errs, e)
	}

	for _, w := range result.Warnings {
		errs = append(errs, w)
	}

	if len(errs) > 0 {
		msg := strings.Builder{}
		for _, e := range errs {
			msg.WriteString(e.Error() + "\n")
		}
		return errors.New(msg.String())
	}

	return nil
}
