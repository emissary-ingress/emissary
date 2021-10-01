package k8sresourceparts

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	TypeMeta   = metav1.TypeMeta
	ObjectMeta = metav1.ObjectMeta

	ObjectReference      = corev1.ObjectReference
	LocalObjectReference = corev1.LocalObjectReference
)
