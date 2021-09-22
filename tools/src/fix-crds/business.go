package main

import (
	"strings"

	"github.com/pkg/errors"

	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

const (
	ProductAES  = Product("aes")
	ProductOSS  = Product("oss")
	ProductHelm = Product("helm")
)

var Products = []Product{
	ProductAES,
	ProductOSS,
	ProductHelm,
}

var (
// old_pro_crds = []string{
// 	"Filter",
// 	"FilterPolicy",
// 	"RateLimit",
// }

// old_oss_crds = []string{
// 	"AuthService",
// 	"ConsulResolver",
// 	"KubernetesEndpointResolver",
// 	"KubernetesServiceResolver",
// 	"LogService",
// 	"Mapping",
// 	"Module",
// 	"RateLimitService",
// 	"TCPMapping",
// 	"TLSContext",
// 	"TracingService",
// }
)

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
		PreserveUnknownFields    *bool                                    `json:"preserveUnknownFields,omitempty"`
	} `json:"spec"`
}

func FixCRD(args Args, crd *CRD) error {
	// sanity check
	if crd.Kind != "CustomResourceDefinition" || !strings.HasPrefix(crd.APIVersion, "apiextensions.k8s.io/") {
		return errors.Errorf("not a CRD: %#v", crd)
	}

	// hack around limitations in `controller-gen`; see the comments in
	// `pkg/api/getambassdor.io/v2/common.go`.
	if crd.Spec.Validation != nil {
		VisitAllSchemaProps(crd.Spec.Validation.OpenAPIV3Schema, func(node *apiext.JSONSchemaProps) {
			if strings.HasPrefix(node.Type, "d6e-union:") {
				types := strings.Split(strings.TrimPrefix(node.Type, "d6e-union:"), ",")
				node.Type = ""
				node.OneOf = nil
				for _, typ := range types {
					node.OneOf = append(node.OneOf, apiext.JSONSchemaProps{
						Type: typ,
					})
				}
			}
		})
	}

	// fix labels
	if crd.Metadata.Labels == nil {
		crd.Metadata.Labels = make(map[string]string)
	}
	crd.Metadata.Labels["product"] = "aes"
	crd.Metadata.Labels["app.kubernetes.io/name"] = "ambassador"

	// fix annotations
	if crd.Metadata.Annotations == nil {
		crd.Metadata.Annotations = make(map[string]string)
	}
	if args.Product == ProductHelm {
		crd.Metadata.Annotations["helm.sh/hook"] = "crd-install"
	} else {
		delete(crd.Metadata.Annotations, "helm.sh/hook")
	}

	// fix categories
	if !inArray("ambassador-crds", crd.Spec.Names.Categories) {
		//fmt.Fprintf(os.Stderr, "CRD %q missing ambassador-crds category\n", crd.Metadata.Name)
		crd.Spec.Names.Categories = append(crd.Spec.Names.Categories, "ambassador-crds")
	}

	return nil
}
