package kates

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	xv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// The kubernetes client libraries and core protobufs are split across so many different packages
// that it is extremely difficult to keep them straight. The naming conventions are also very poorly
// chosen resulting in frequent name collisions under even simple uses. Case in point, there are a
// whole lot of v1 packages (core/v1, apiextensions/v1, meta/v1, apps/v1) just to name a few, and
// you need to use at least 3 of these together in order to accomplish almost anything. The types
// within packages are often horribly named as well (e.g. dynamic.Interface, rest.Config,
// version.Info are not super description when you omit the packages).
//
// The aliases in this file are intended to do several things:
//
//   1. Make our kubernetes code easier to read by providing a standard set of aliases instead of
//      requiring developers to make up potentially different aliases at the point of use.
//
//   2. Make for a simpler and easier Quickstart for our kubernetes library by providing a single
//      entry point.
//
//   3. Allow us to build a set of simpler and easier to use APIs on top of client-go while using
//      types that are compatible with client-go so we have a good escape hatch for directly using
//      client-go.
//
//   4. Provide a single file that helps me (rhs@datawire.io) remember where the hell everything is.

// type related aliases

type TypeMeta = metav1.TypeMeta
type ObjectMeta = metav1.ObjectMeta

type Namespace = corev1.Namespace

type ConfigMap = corev1.ConfigMap

type Secret = corev1.Secret

type Service = corev1.Service
type ServiceSpec = corev1.ServiceSpec
type ServicePort = corev1.ServicePort

var ServiceTypeLoadBalancer = corev1.ServiceTypeLoadBalancer

type Pod = corev1.Pod
type PodSpec = corev1.PodSpec
type Container = corev1.Container
type SecurityContext = corev1.SecurityContext

type Volume = corev1.Volume
type VolumeSource = corev1.VolumeSource
type PersistentVolumeClaimVolumeSource = corev1.PersistentVolumeClaimVolumeSource
type VolumeMount = corev1.VolumeMount

type ResourceRequirements = corev1.ResourceRequirements
type ResourceList = corev1.ResourceList

const ResourceCPU = corev1.ResourceCPU
const ResourceMemory = corev1.ResourceMemory

type PersistentVolumeClaim = corev1.PersistentVolumeClaim

type Deployment = appsv1.Deployment

type CustomResourceDefinition = xv1.CustomResourceDefinition

var NamesAccepted = xv1.NamesAccepted
var Established = xv1.Established
var ConditionTrue = xv1.ConditionTrue

type Unstructured = unstructured.Unstructured

var MustParseQuantity = resource.MustParse

type Quantity = resource.Quantity
type IntOrString = intstr.IntOrString

var Int = intstr.Int

// client related aliases

type ConfigFlags = genericclioptions.ConfigFlags

var NewConfigFlags = genericclioptions.NewConfigFlags

type PatchType = types.PatchType

var (
	JSONPatchType           = types.JSONPatchType
	MergePatchType          = types.MergePatchType
	StrategicMergePatchType = types.StrategicMergePatchType
	ApplyPatchType          = types.ApplyPatchType
)

type GetOptions = metav1.GetOptions
type ListOptions = metav1.ListOptions
type CreateOptions = metav1.CreateOptions
type UpdateOptions = metav1.UpdateOptions
type PatchOptions = metav1.PatchOptions
type DeleteOptions = metav1.DeleteOptions

var NamespaceAll = metav1.NamespaceAll
var NamespaceNone = metav1.NamespaceNone

type Selector = labels.Selector
type LabelSet = labels.Set

var ParseSelector = labels.Parse

// error related aliases

var IsNotFound = apierrors.IsNotFound
var IsConflict = apierrors.IsConflict

//

type Object interface {
	runtime.Object
	metav1.Object
}
