package pgs

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type enumPrinter struct {
	Visitor
}

func EnumPrinter() Visitor { return enumPrinter{NilVisitor()} }

func (p enumPrinter) VisitMessage(m Message) (Visitor, error) { return p, nil }

func (p enumPrinter) VisitEnum(e Enum) (Visitor, error) {
	fmt.Println(e.Name())
	return nil, nil
}

func ExampleNilVisitor() {
	n := enumNode()
	p := EnumPrinter()

	if err := Walk(p, n); err != nil {
		panic(err)
	}

	// Output:
	// Bar
	// Foo
}

func enumNode() Node {
	// simulating the following proto file:
	//
	// syntax="proto3";
	//
	// package fizz;
	//
	// message Gadget {
	//
	//   enum Bar {
	//     // ...
	//   }
	//
	//   message Gizmo {
	//     enum Foo {
	//       // ...
	//     }
	//   }
	// }

	sm := &msg{}
	sm.addEnum(&enum{rawDesc: &descriptor.EnumDescriptorProto{Name: proto.String("Foo")}})

	m := &msg{}
	m.addMessage(sm)
	m.addEnum(&enum{rawDesc: &descriptor.EnumDescriptorProto{Name: proto.String("Bar")}})

	return m
}
