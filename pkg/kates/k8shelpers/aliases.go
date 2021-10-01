package k8shelpers

import (
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Server metadata ///////////////////////////////////////////////////

type (
	APIResource     = metav1.APIResource
	APIResourceList = metav1.APIResourceList
	VersionInfo     = version.Info
)

// Verb options //////////////////////////////////////////////////////

type (
	GetOptions    = metav1.GetOptions
	ListOptions   = metav1.ListOptions
	CreateOptions = metav1.CreateOptions
	UpdateOptions = metav1.UpdateOptions
	PatchOptions  = metav1.PatchOptions
	DeleteOptions = metav1.DeleteOptions

	PodLogOptions = corev1.PodLogOptions
)

type PatchType = types.PatchType

var (
	JSONPatchType           = types.JSONPatchType
	MergePatchType          = types.MergePatchType
	StrategicMergePatchType = types.StrategicMergePatchType
	ApplyPatchType          = types.ApplyPatchType
)

// Misc //////////////////////////////////////////////////////////////

type (
	LabelSelector = labels.Selector
	LabelSet      = labels.Set
)

func ParseLabelSelector(selector string) (LabelSelector, error) {
	return labels.Parse(selector)
}

type CLIConfigFlags = genericclioptions.ConfigFlags

func NewCLIConfigFlags(usePersistentConfig bool) *CLIConfigFlags {
	return genericclioptions.NewConfigFlags(usePersistentConfig)
}

func IsNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}
func IsConflict(err error) bool {
	return apierrors.IsConflict(err)
}

const (
	NamespaceAll  = metav1.NamespaceAll
	NamespaceNone = metav1.NamespaceNone
)
