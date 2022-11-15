// -*- fill-column: 70 -*-

// Copyright 2020-2021 Datawire.  All rights reserved
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License.  You
// may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.  See the License for the specific language governing
// permissions and limitations under the License.

//////////////////////////////////////////////////////////////////////
// 0. Table of Contents //////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////
//
// This file deals with common package/apiVersion-level things not
// specific to any individual CRD:
//
//  1. Magic markers: Document the various magic "+marker" comments
//     that are used in this package, and set up the package-level
//     markers.
//
//  2. Package documentation: The `godoc` package-wide documentation.
//
//  3. Scheme: Set up the Group/Version/SchemeBuilder/AddToScheme for
//     this apiVersion.
//
// Things that are shared between multiple CRDs, but are for
// individual CRDs rather than the package/apiVersion as a whole, do
// not belong in this file; they belong in `common.go`.

//////////////////////////////////////////////////////////////////////
// 1. Magic markers //////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////
//
// We use a bunch of magic comments called "+markers" that serve as
// input to `controller-gen` and `conversion-gen`.  Note that while
// `controller-gen` doesn't care about what file these are in, the
// older `k8s.io/gengo`-based `conversion-gen` specifically looks for
// `doc.go`.  Mostly they annotate a type, or a field within a struct.
// Just below here, we do the "global" package-level markers; these
// package-level markers need to come before the "package" line.
//
// The type markers of interest are:
//
//  - "+kubebuilder:object:generate=bool" whether to generate
//    `DeepCopy` and `DeepCopyInto` methods for this type; but we
//    don't actually set this on types, since we can set it to true
//    for all types at the package-level.
//
//  - "+kubebuilder:object:root=bool" whether to *also* generate a
//    `DeepCopyObject` method.  It upsets me that controller-gen
//    doesn't infer this based on the presence of metav1.TypeMeta
//    inside of the type.
//
//  - "+kubebuilder:subresource:status" whether to add "status" as a
//    subresource for that type.  It upsets me that controller-gen
//    doesn't infer this based on the presence of a `status` field
//    inside of the type.
//
// The field markers of interest are:
//
//  - The "+kubebuilder:validation:*" markers control the OpenAPI v3
//    validation schema that is generated for this field.  ":Optional"
//    or ":Required" may be applied at the package-level in order to
//    set the default for all fields.  Most of the others can also be
//    set at the type level.
//
// Package-level markers:
//
// The group name to use for the CRDs in the generated YAML:
// +groupName=getambassador.io
// +versionName=v1
//
// By default, mark all types in this package to have DeepCopy methods
// generated (so we don't need to specify this for every type):
// +kubebuilder:object:generate=true
//
// By default, mark all fields as optional (so we don't need to
// specify this for every optional field, since most fields are
// optional; and also because controller-gen's "required-by-default"
// mode is broken and always makes everything optional, even if it's
// explicitly marked as required):
// +kubebuilder:validation:Optional
//
// Have conversion-gen help write the code to convert to and from
// newer CRD versions.
// +k8s:conversion-gen=github.com/emissary-ingress/emissary/v3/pkg/api/getambassador.io/v2

//////////////////////////////////////////////////////////////////////
// 2. Package documentation //////////////////////////////////////////
//////////////////////////////////////////////////////////////////////

// Package v1 contains API Schema definitions for the getambassador.io
// v1 API group
package v1

//////////////////////////////////////////////////////////////////////
// 3. Scheme /////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "getambassador.io", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme

	// This is so the generated conversion code will compile.
	localSchemeBuilder = &SchemeBuilder.SchemeBuilder
)
