package gateway_test

import (
	"github.com/emissary-ingress/emissary/v3/pkg/kates"
	"k8s.io/apimachinery/pkg/runtime"
)

// makeFoo, Foo, and FooSpec are all helpers for quickly/easily creating dummy resources for testing
// purposes.
func makeFoo(namespace, name, value string) *Foo {
	return &Foo{
		TypeMeta:   kates.TypeMeta{Kind: "Foo"},
		ObjectMeta: kates.ObjectMeta{Namespace: namespace, Name: name},
		Spec: FooSpec{
			Value: value,
		},
	}
}

type Foo struct {
	kates.TypeMeta
	kates.ObjectMeta
	Spec FooSpec
}

type FooSpec struct {
	Value    string
	PanicArg error
}

func (f *Foo) DeepCopyObject() runtime.Object {
	return nil
}
