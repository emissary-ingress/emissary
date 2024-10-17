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
//  4. API design guidelines: Guidelines for additions to this
//     package.
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
// +versionName=v4
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
// Other apiVersions say "+k8s:conversion-gen=…/v4" to have
// conversion-gen help write code to convert to/from this apiVersion
// and that one.

//////////////////////////////////////////////////////////////////////
// 2. Package documentation //////////////////////////////////////////
//////////////////////////////////////////////////////////////////////

// package v4alpha1 contains API Schema definitions for the
// getambassador.io v4 API group
package v4alpha1

//////////////////////////////////////////////////////////////////////
// 3. Scheme /////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "getambassador.io", Version: "v4"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

//////////////////////////////////////////////////////////////////////
// 4. API design guidelines //////////////////////////////////////////
//////////////////////////////////////////////////////////////////////
//
// Ambassador's API has inconsistencies because it has historical
// baggage.  Not all of Ambassador's existing API (or even most of
// it!?) follow these guidelines, but new additions to the API should.
// If/when we advance to getambassador.io/v3 and we can break
// compatibility, these are things that we should apply everywhere.
//
// - Prefer `camelCase` to `snake_case`
//   * Exception: Except for consistency with existing fields in the
//     same resource, or symmetry with identical fields in another
//     resource.
//   * Justification: Kubernetes style is to use camelCase. But
//     historically Ambassador used snake_case for everything.
//
// - Give _every_ field a `json:""` struct tag.
//   * Justification: Marshaling and unmarshaling are key to what we
//     do, and it's critical to carefully define how it happens.
//   * Notes: This is not optional. Do it for _every field_. (It's OK
//     if the tag is literally `json:""` for fields that must never be
//     exposed during marshaling.)
//
// - Prefer `*int`, and `*bool`; rather than just `int`, `bool`.
//   * Justification: The Ambassador API is rooted in Python, where it
//     is always possible to tell if a given element was present in in
//     a CRD, or left unset.  This is at odds with Go's `omitempty`
//     specifier, which really means "omit if empty _or if set to the
//     default (zero) value_".  For int in particular, this results in
//     a value of 0 being omitted, and for many Ambassador fields, 0
//     is not the correct default value.
//
//     This resulted in a lot of bugs in the 1.10 timeframe, so be
//     careful going forward.
//
// - Prefer for object references to not support namespacing
//   * Exception: If there's a real use-case for it.
//   * Justification: Most native Kubernetes resources don't support
//     referencing things in a different namespace.  We should be
//     opinionated and not support it either, unless there's a good
//     reason to in a specific case.
//
// - Prefer to use `corev1.LocalObjectReference` or
//   `corev1.SecretReference` references instead of
//   `{name}.{namespace}` strings.
//   * Justification: The `{name}.{namespace}` thing evolved "an
//     opaque DNS name" in the `service` field of Mappings, and that
//     was generalized to other things.  Outside of the context of
//     "this is usable as a DNS name to make a request to", it's just
//     confusing and introduces needless ambiguity.  Nothing other
//     than Ambassador uses that notation.
//   * Notes: For things that don't support cross-namespace references
//     (see above), use LocalObjectReference; if you really must
//     support cross-namespace references, then use SecretReference.
//
// - Prefer to use `metav1.Duration` fields instead of "_s" or "_ms"
//   numeric fields.
//
// - Don't have Ambassador populate anything in the `.spec` or
//   `.metadata` of something a user might edit, only let Ambassador
//   set things in the `.status`.
//   * Exception: If Ambassador 100% owns the resource and a user will
//     never edit it.
//   * Notes: I didn't write "Prefer" on this one.  Don't violate it.
//     Just don't do it.  Ever.  Designing the Host resource in
//     violation of this was a HUGE mistake and one that I regret very
//     much.  Learn from my mistakes.
//   * Justification: Having Ambassador-set things in a subresource
//     from user-set things:
//     1. avoids races between the user updating the spec and us
//        updating the status
//     2. allows watt/whatever to only pay attention to
//        .metadata.generation instead of .metadata.resourceVersion;
//        avoiding pointless reconfigures.
//     3. allows the RBAC to be simpler
//     4. avoids the whole class of bugs where we need to make sure
//        that everything round-trips correctly
//     5. provides clarity on which things a user is expected to know
//        how to fill in
