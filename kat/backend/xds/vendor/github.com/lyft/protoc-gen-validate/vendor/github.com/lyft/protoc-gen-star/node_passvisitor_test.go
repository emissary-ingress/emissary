package pgs

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type fieldPrinter struct {
	Visitor
}

func FieldPrinter() Visitor {
	p := &fieldPrinter{}
	p.Visitor = PassThroughVisitor(p)
	return p
}

func (p fieldPrinter) VisitField(f Field) (Visitor, error) {
	fmt.Println(f.Name())
	return nil, nil
}

func ExamplePassThroughVisitor() {
	n := fieldNode()
	p := FieldPrinter()

	if err := Walk(p, n); err != nil {
		panic(err)
	}

	// Output:
	// Foo
	// Bar
}

func fieldNode() Node {
	// simulating the following proto file:
	//
	// syntax="proto3";
	//
	// package fizz;
	//
	// message Gadget {
	//   string Bar = 1;
	//
	//   message Gizmo {
	//     int Foo = 1;
	//   }
	// }

	sm := &msg{}
	sm.addField(&field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("Foo")}})

	m := &msg{}
	m.addMessage(sm)
	m.addField(&field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("Bar")}})

	f := &file{}
	f.addMessage(m)

	p := &pkg{}
	p.addFile(f)

	return p
}
