package pgs

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestEnumVal_Name(t *testing.T) {
	t.Parallel()
	ev := &enumVal{desc: &descriptor.EnumValueDescriptorProto{Name: proto.String("eval")}}
	assert.Equal(t, "eval", ev.Name().String())
}

func TestEnumVal_Syntax(t *testing.T) {
	t.Parallel()
	ev := &enumVal{}
	e := dummyEnum()
	e.addValue(ev)
	assert.Equal(t, e.Syntax(), ev.Syntax())
}

func TestEnumVal_Package(t *testing.T) {
	t.Parallel()
	ev := &enumVal{}
	e := dummyEnum()
	e.addValue(ev)
	assert.NotNil(t, ev.Package())
	assert.Equal(t, e.Package(), ev.Package())
}

func TestEnumVal_File(t *testing.T) {
	t.Parallel()
	ev := &enumVal{}
	e := dummyEnum()
	e.addValue(ev)
	assert.NotNil(t, ev.File())
	assert.Equal(t, e.File(), ev.File())
}

func TestEnumVal_BuildTarget(t *testing.T) {
	t.Parallel()
	ev := &enumVal{}
	e := dummyEnum()
	e.addValue(ev)
	assert.False(t, ev.BuildTarget())
	e.parent = &file{buildTarget: true}
	assert.True(t, ev.BuildTarget())
}

func TestEnumVal_Descriptor(t *testing.T) {
	t.Parallel()
	ev := &enumVal{desc: &descriptor.EnumValueDescriptorProto{}}
	assert.Equal(t, ev.desc, ev.Descriptor())
}

func TestEnumVal_Enum(t *testing.T) {
	t.Parallel()
	ev := &enumVal{}
	e := dummyEnum()
	e.addValue(ev)
	assert.Equal(t, e, ev.Enum())
}

func TestEnumVal_Value(t *testing.T) {
	t.Parallel()
	ev := &enumVal{desc: &descriptor.EnumValueDescriptorProto{Number: proto.Int32(123)}}
	assert.Equal(t, int32(123), ev.Value())
}

func TestEnumVal_Imports(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (&enumVal{}).Imports())
}

func TestEnumVal_Extension(t *testing.T) {
	// cannot be parallel

	ev := &enumVal{desc: &descriptor.EnumValueDescriptorProto{}}
	assert.NotPanics(t, func() { ev.Extension(nil, nil) })
}

func TestEnumVal_Accept(t *testing.T) {
	t.Parallel()

	ev := &enumVal{}
	assert.NoError(t, ev.accept(nil))

	v := &mockVisitor{err: errors.New("")}
	assert.Error(t, ev.accept(v))
	assert.Equal(t, 1, v.enumvalue)
}

func TestEnumVAl_LookupName(t *testing.T) {
	t.Parallel()

	ev := &enumVal{desc: &descriptor.EnumValueDescriptorProto{Name: proto.String("ev")}}
	e := dummyEnum()
	e.addValue(ev)

	assert.Equal(t, e.lookupName()+".ev", ev.lookupName())
}

type mockEnumValue struct {
	EnumValue
	e   Enum
	err error
}

func (ev *mockEnumValue) setEnum(e Enum) { ev.e = e }

func (ev *mockEnumValue) accept(v Visitor) error {
	_, err := v.VisitEnumValue(ev)
	if ev.err != nil {
		return ev.err
	}
	return err
}
