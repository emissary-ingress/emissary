package k8s_resource_types

import (
	"fmt"

	types_ext_v1beta1 "k8s.io/api/extensions/v1beta1"
	types_net_v1 "k8s.io/api/networking/v1"
	types_net_v1beta1 "k8s.io/api/networking/v1beta1"
	types_net_internal "k8s.io/kubernetes/pkg/apis/networking"

	conv_ext_v1beta1 "k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	conv_net_v1 "k8s.io/kubernetes/pkg/apis/networking/v1"
	conv_net_v1beta1 "k8s.io/kubernetes/pkg/apis/networking/v1beta1"

	kates_internal "github.com/datawire/ambassador/v2/pkg/kates/internal"
	k8s_metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_runtime "k8s.io/apimachinery/pkg/runtime"
)

// TODO: Update the consumers (mostly Python, unfortunately... weaker typechecking makes this hard)
// of this to understand types_net_v1.Ingress (or even types_net_internal.Ingress) instead.
type Ingress = types_ext_v1beta1.Ingress

var IngressTypeMeta = k8s_metav1.TypeMeta{
	APIVersion: types_ext_v1beta1.SchemeGroupVersion.String(),
	Kind:       "Ingress",
}

func NewIngress(untyped k8s_runtime.Object) (*Ingress, error) {
	var internal types_net_internal.Ingress
	switch untyped.GetObjectKind().GroupVersionKind() {
	case types_ext_v1beta1.SchemeGroupVersion.WithKind("Ingress"):
		var typed types_ext_v1beta1.Ingress
		if err := kates_internal.Convert(untyped, &typed); err != nil {
			return nil, err
		}
		if err := conv_ext_v1beta1.Convert_v1beta1_Ingress_To_networking_Ingress(&typed, &internal, nil); err != nil {
			return nil, err
		}
	case types_net_v1beta1.SchemeGroupVersion.WithKind("Ingress"):
		var typed types_net_v1beta1.Ingress
		if err := kates_internal.Convert(untyped, &typed); err != nil {
			return nil, err
		}
		if err := conv_net_v1beta1.Convert_v1beta1_Ingress_To_networking_Ingress(&typed, &internal, nil); err != nil {
			return nil, err
		}
	case types_net_v1.SchemeGroupVersion.WithKind("Ingress"):
		var typed types_net_v1.Ingress
		if err := kates_internal.Convert(untyped, &typed); err != nil {
			return nil, err
		}
		if err := conv_net_v1.Convert_v1_Ingress_To_networking_Ingress(&typed, &internal, nil); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognized Ingress GroupVersionKind: %v", untyped.GetObjectKind().GroupVersionKind())
	}

	var ret Ingress
	ret.TypeMeta = IngressTypeMeta
	if err := conv_ext_v1beta1.Convert_networking_Ingress_To_v1beta1_Ingress(&internal, &ret, nil); err != nil {
		return nil, err
	}
	return &ret, nil
}

// TODO: Consider migrating the consumers (mostly Python, unfortunately... weaker typechecking makes
// this hard) of this to understand types_net_internal.IngressClass instead?
type IngressClass = types_net_v1.IngressClass

var IngressClassTypeMeta = k8s_metav1.TypeMeta{
	APIVersion: types_net_v1.SchemeGroupVersion.String(),
	Kind:       "IngressClass",
}

func NewIngressClass(untyped k8s_runtime.Object) (*IngressClass, error) {
	var internal types_net_internal.IngressClass
	switch untyped.GetObjectKind().GroupVersionKind() {
	case types_net_v1beta1.SchemeGroupVersion.WithKind("IngressClass"):
		var typed types_net_v1beta1.IngressClass
		if err := kates_internal.Convert(untyped, &typed); err != nil {
			return nil, err
		}
		if err := conv_net_v1beta1.Convert_v1beta1_IngressClass_To_networking_IngressClass(&typed, &internal, nil); err != nil {
			return nil, err
		}
	case types_net_v1.SchemeGroupVersion.WithKind("IngressClass"):
		var typed types_net_v1.IngressClass
		if err := kates_internal.Convert(untyped, &typed); err != nil {
			return nil, err
		}
		if err := conv_net_v1.Convert_v1_IngressClass_To_networking_IngressClass(&typed, &internal, nil); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognized IngressClass GroupVersionKind: %v", untyped.GetObjectKind().GroupVersionKind())
	}

	var ret IngressClass
	ret.TypeMeta = IngressClassTypeMeta
	if err := conv_net_v1.Convert_networking_IngressClass_To_v1_IngressClass(&internal, &ret, nil); err != nil {
		return nil, err
	}
	return &ret, nil
}
