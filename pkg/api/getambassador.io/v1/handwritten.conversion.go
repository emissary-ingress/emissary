// This file is ultimately authored by a human, you can ask the build system to generate the
// necessary signatures for you by running (in the project root)
//
//    make $PWD/pkg/api/getambassador.io/v1/handwritten.conversion.scaffold.go
//
// You can then diff `handwritten.conversion.go` and `handwritten.conversion.scaffold.go` to make
// sure you have all of the functions that conversion-gen thinks you need.

package v1

import (
	k8sRuntime "k8s.io/apimachinery/pkg/runtime"
	k8sRuntimeUtil "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v2"
	"github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v3alpha1"
)

// These first few functions are written of our own human initiative.

var (
	conversionScheme = func() *k8sRuntime.Scheme {
		scheme := k8sRuntime.NewScheme()
		k8sRuntimeUtil.Must(AddToScheme(scheme))
		k8sRuntimeUtil.Must(v2.AddToScheme(scheme))
		k8sRuntimeUtil.Must(v3alpha1.AddToScheme(scheme))
		return scheme
	}
	conversionIntermediates = []k8sRuntime.GroupVersioner{
		// v1 (spoke)
		v2.GroupVersion,
		// v3alpha1 (hub)
	}
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// The remaining functions are all filled out from `handwritten.conversion.scaffold.go` (see the
// comment at the top of the file).  I like to leave in the "WARNING" and "INFO" comments that
// `handwritten.conversion.scaffold.go` has, so that I can (1) compare the comments and the code,
// and make sure the code does everything the comments mention, and (2) compare this file against
// `handwritten.conversion.scaffold.go` to make sure all the comments are there.
