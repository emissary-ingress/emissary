package k8sresourcetypes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type (
	Unstructured = unstructured.Unstructured

	// core/v1
	Namespace      = corev1.Namespace
	Event          = corev1.Event
	ConfigMap      = corev1.ConfigMap
	Secret         = corev1.Secret
	Service        = corev1.Service
	Endpoints      = corev1.Endpoints
	ServiceAccount = corev1.ServiceAccount
	Pod            = corev1.Pod
	Node           = corev1.Node

	// rbac/v1
	Role               = rbacv1.Role
	RoleBinding        = rbacv1.RoleBinding
	ClusterRole        = rbacv1.ClusterRole
	ClusterRoleBinding = rbacv1.ClusterRoleBinding

	// apps/v1
	Deployment  = appsv1.Deployment
	ReplicaSet  = appsv1.ReplicaSet
	StatefulSet = appsv1.StatefulSet

	// apiextensions/v1
	CustomResourceDefinition = apiextv1.CustomResourceDefinition
)
