// This file is a lightly modified subset of
// sigs.k8s.io/controller-runtime/pkg/webhook/conversion/conversion.go.

/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

type Webhook struct {
	scheme *runtime.Scheme
}

// convertObject will convert given a src object to dst object.
// Note(droot): couldn't find a way to reduce the cyclomatic complexity under 10
// without compromising readability, so disabling gocyclo linter
func (wh *Webhook) convertObject(src, dst runtime.Object) error {
	srcGVK := src.GetObjectKind().GroupVersionKind()
	dstGVK := dst.GetObjectKind().GroupVersionKind()

	if srcGVK.GroupKind() != dstGVK.GroupKind() {
		return fmt.Errorf("src %T and dst %T does not belong to same API Group", src, dst)
	}

	if srcGVK == dstGVK {
		return fmt.Errorf("conversion is not allowed between same type %T", src)
	}

	srcIsHub, dstIsHub := isHub(src), isHub(dst)
	srcIsConvertible, dstIsConvertible := isConvertible(src), isConvertible(dst)

	switch {
	case srcIsHub && dstIsConvertible:
		return dst.(conversion.Convertible).ConvertFrom(src.(conversion.Hub))
	case dstIsHub && srcIsConvertible:
		return src.(conversion.Convertible).ConvertTo(dst.(conversion.Hub))
	case srcIsConvertible && dstIsConvertible:
		return wh.convertViaHub(src.(conversion.Convertible), dst.(conversion.Convertible))
	default:
		return fmt.Errorf("%T is not convertible to %T", src, dst)
	}
}

func (wh *Webhook) convertViaHub(src, dst conversion.Convertible) error {
	hub, err := wh.getHub(src)
	if err != nil {
		return err
	}

	if hub == nil {
		return fmt.Errorf("%s does not have any Hub defined", src)
	}

	err = src.ConvertTo(hub)
	if err != nil {
		return fmt.Errorf("%T failed to convert to hub version %T : %w", src, hub, err)
	}

	err = dst.ConvertFrom(hub)
	if err != nil {
		return fmt.Errorf("%T failed to convert from hub version %T : %w", dst, hub, err)
	}

	return nil
}

// getHub returns an instance of the Hub for passed-in object's group/kind.
func (wh *Webhook) getHub(obj runtime.Object) (conversion.Hub, error) {
	gvks, err := objectGVKs(wh.scheme, obj)
	if err != nil {
		return nil, err
	}
	if len(gvks) == 0 {
		return nil, fmt.Errorf("error retrieving gvks for object : %v", obj)
	}

	var hub conversion.Hub
	var hubFoundAlready bool
	for _, gvk := range gvks {
		instance, err := wh.scheme.New(gvk)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate an instance for gvk %v: %w", gvk, err)
		}
		if val, isHub := instance.(conversion.Hub); isHub {
			if hubFoundAlready {
				return nil, fmt.Errorf("multiple hub version defined for %T", obj)
			}
			hubFoundAlready = true
			hub = val
		}
	}
	return hub, nil
}

// objectGVKs returns all (Group,Version,Kind) for the Group/Kind of given object.
func objectGVKs(scheme *runtime.Scheme, obj runtime.Object) ([]schema.GroupVersionKind, error) {
	// NB: we should not use `obj.GetObjectKind().GroupVersionKind()` to get the
	// GVK here, since it is parsed from apiVersion and kind fields and it may
	// return empty GVK if obj is an uninitialized object.
	objGVKs, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return nil, err
	}
	if len(objGVKs) != 1 {
		return nil, fmt.Errorf("expect to get only one GVK for %v", obj)
	}
	objGVK := objGVKs[0]
	knownTypes := scheme.AllKnownTypes()

	var gvks []schema.GroupVersionKind
	for gvk := range knownTypes {
		if objGVK.GroupKind() == gvk.GroupKind() {
			gvks = append(gvks, gvk)
		}
	}
	return gvks, nil
}

// isHub determines if passed-in object is a Hub or not.
func isHub(obj runtime.Object) bool {
	_, yes := obj.(conversion.Hub)
	return yes
}

// isConvertible determines if passed-in object is a convertible.
func isConvertible(obj runtime.Object) bool {
	_, yes := obj.(conversion.Convertible)
	return yes
}
