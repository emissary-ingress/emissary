package main

import (
	"fmt"
	"strings"

	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

const (
	TargetAPIServerKubectl  = "apiserver-kubectl"
	TargetAPIServerKAT      = "apiserver-kat"
	TargetInternalValidator = "internal-validator"
)

var Targets = []string{
	TargetAPIServerKubectl,
	TargetAPIServerKAT,
	TargetInternalValidator,
}

// Like apiext.CustomResourceDefinition, but we have a little more
// control over serialization.
type CRD struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
		Name        string            `json:"name"`
		Labels      map[string]string `json:"labels,omitempty"`
		Annotations map[string]string `json:"annotations,omitempty"`
		// Other things, like CreationTimestamp, are
		// intentionally omited.
	} `json:"metadata"`

	Spec struct {
		Group                    string                                   `json:"group"`
		Version                  *NilableString                           `json:"version,omitempty"`
		Names                    apiext.CustomResourceDefinitionNames     `json:"names"`
		Scope                    apiext.ResourceScope                     `json:"scope"`
		Validation               *apiext.CustomResourceValidation         `json:"validation,omitempty"`
		Subresources             *apiext.CustomResourceSubresources       `json:"subresources,omitempty"`
		Versions                 []apiext.CustomResourceDefinitionVersion `json:"versions,omitempty"`
		AdditionalPrinterColumns []apiext.CustomResourceColumnDefinition  `json:"additionalPrinterColumns,omitempty"`
		Conversion               *apiext.CustomResourceConversion         `json:"conversion,omitempty"`
		// Explicitly setting 'preserveUnknownFields: false' is important even though that's
		// the default; it's important when upgrading from CRDv1beta1 to CRDv1; the default
		// was true in v1beta1, but we need it to be false.
		PreserveUnknownFields bool `json:"preserveUnknownFields"`
	} `json:"spec"`
}

func FixCRD(args Args, crd *CRD) error {
	// Really, Golang? You couldn't just have a "set" type?
	// TODO(Flynn): Look into letting kubebuilder generate our unserved v1.
	CRDsWithNoUnservedV1 := map[string]interface{}{
		"filterpolicies.getambassador.io": struct{}{},
		"filters.getambassador.io":        struct{}{},
		"ratelimits.getambassador.io":     struct{}{},
	}

	// sanity check
	if crd.Kind != "CustomResourceDefinition" || !strings.HasPrefix(crd.APIVersion, "apiextensions.k8s.io/") {
		return fmt.Errorf("not a CRD: %#v", crd)
	}

	// hack around non-structural schemas; see the comments in
	// `pkg/api/getambassdor.io/v2/common.go`.
	if err := VisitAllSchemaProps(crd, func(version string, node *apiext.JSONSchemaProps) error {
		if strings.HasPrefix(node.Type, "d6e-union:") {
			if strings.HasPrefix(version, "v3") {
				return fmt.Errorf("v3 schemas should not contain d6e-union types")
			}
			if args.Target != TargetInternalValidator {
				return ErrExcludeFromSchema
			}
			types := strings.Split(strings.TrimPrefix(node.Type, "d6e-union:"), ",")
			node.Type = ""
			node.OneOf = nil
			for _, typ := range types {
				node.OneOf = append(node.OneOf, apiext.JSONSchemaProps{
					Type: typ,
				})
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// fix labels
	if args.Target != TargetInternalValidator {
		if crd.Metadata.Labels == nil {
			crd.Metadata.Labels = make(map[string]string)
		}
		for k, v := range globalLabels {
			crd.Metadata.Labels[k] = v
		}
	}

	// fix annotations
	if crd.Metadata.Annotations == nil {
		crd.Metadata.Annotations = make(map[string]string)
	}
	delete(crd.Metadata.Annotations, "helm.sh/hook")

	// fix categories
	if !inArray("ambassador-crds", crd.Spec.Names.Categories) {
		crd.Spec.Names.Categories = append(crd.Spec.Names.Categories, "ambassador-crds")
	}

	// fix conversion
	if len(crd.Spec.Versions) > 1 {
		crd.Spec.Conversion = &apiext.CustomResourceConversion{
			Strategy: apiext.WebhookConverter,
			Webhook: &apiext.WebhookConversion{
				// 'ClientConfig' will get overwritten by Emissary's 'apiext'
				// controller.
				ClientConfig: &apiext.WebhookClientConfig{
					Service: &apiext.ServiceReference{
						Name:      apiextSvcName,
						Namespace: namespace,
					},
				},
				// Which versions of the conversion API our webhook supports.  Since
				// we use sigs.k8s.io/controller-runtime/pkg/webhook/conversion to
				// implement the webhook this list should be kept in-sync with what
				// that package supports.
				ConversionReviewVersions: []string{"v1beta1"},
			},
		}

		// If we're generating stuff for the APIServer, make sure we have an unserved,
		// unstored v1 version, unless it's one of the CRDs for which we shouldn't do that.
		//
		// TODO(Flynn): Look into letting kubebuilder generate our unserved v1.
		if args.Target == TargetAPIServerKubectl {
			_, skip := CRDsWithNoUnservedV1[crd.Metadata.Name]

			if !skip {
				var v2desc string
				versions := make([]apiext.CustomResourceDefinitionVersion, 0, len(crd.Spec.Versions))

				for _, version := range crd.Spec.Versions {
					if version.Name != "v1" {
						versions = append(versions, version)
					}

					if version.Name == "v2" {
						v2desc = version.Schema.OpenAPIV3Schema.Description

						// I don't want multiline descriptions in the unserved v1, so let's see
						// if we have a newline in v2desc.
						idx := strings.Index(v2desc, "\n")

						if idx > 0 {
							// Yup. Toss everything after it.
							v2desc = v2desc[:idx]
						}

						// Finally, strip whitespace whether we found a newline or not.
						v2desc = strings.TrimSpace(v2desc)
					}
				}

				preserveUnknownFields := true

				versions = append(versions, apiext.CustomResourceDefinitionVersion{
					Name: "v1",
					Schema: &apiext.CustomResourceValidation{
						OpenAPIV3Schema: &apiext.JSONSchemaProps{
							Description:            v2desc,
							Type:                   "object",
							XPreserveUnknownFields: &preserveUnknownFields,
						},
					},
					Served:  false,
					Storage: false,
				})

				crd.Spec.Versions = versions
			}
		}
	}

	return nil
}
