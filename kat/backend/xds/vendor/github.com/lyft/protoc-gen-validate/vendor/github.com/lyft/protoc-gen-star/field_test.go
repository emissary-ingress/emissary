package pgs

import (
	"testing"

	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestField_Name(t *testing.T) {
	t.Parallel()

	f := &field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("foo")}}

	assert.Equal(t, "foo", f.Name().String())
}

func TestField_Syntax(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.addField(f)

	assert.Equal(t, m.Syntax(), f.Syntax())
}

func TestField_Package(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.addField(f)

	assert.NotNil(t, f.Package())
	assert.Equal(t, m.Package(), f.Package())
}

func TestField_File(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.addField(f)

	assert.NotNil(t, f.File())
	assert.Equal(t, m.File(), f.File())
}

func TestField_BuildTarget(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.addField(f)

	assert.False(t, f.BuildTarget())
	m.setParent(&file{buildTarget: true})
	assert.True(t, f.BuildTarget())
}

func TestField_LookupName(t *testing.T) {
	t.Parallel()

	f := &field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("field")}}
	m := dummyMsg()
	m.addField(f)

	assert.Equal(t, m.lookupName()+".field", f.lookupName())
}

func TestField_Descriptor(t *testing.T) {
	t.Parallel()

	f := &field{desc: &descriptor.FieldDescriptorProto{}}
	assert.Equal(t, f.desc, f.Descriptor())
}

func TestField_Message(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.addField(f)

	assert.Equal(t, m, f.Message())
}

func TestField_OneOf(t *testing.T) {
	t.Parallel()

	f := &field{}
	assert.Nil(t, f.OneOf())
	assert.False(t, f.InOneOf())

	o := dummyOneof()
	o.addField(f)

	assert.Equal(t, o, f.OneOf())
	assert.True(t, f.InOneOf())
}

func TestField_Type(t *testing.T) {
	t.Parallel()

	f := &field{}
	f.addType(&scalarT{})

	assert.Equal(t, f.typ, f.Type())
}

func TestField_Extension(t *testing.T) {
	// cannot be parallel

	f := &field{desc: &descriptor.FieldDescriptorProto{}}
	assert.NotPanics(t, func() { f.Extension(nil, nil) })
}

func TestField_Accept(t *testing.T) {
	t.Parallel()

	f := &field{}

	assert.NoError(t, f.accept(nil))

	v := &mockVisitor{err: errors.New("")}
	assert.Error(t, f.accept(v))
	assert.Equal(t, 1, v.field)
}

func TestField_Imports(t *testing.T) {
	t.Parallel()

	f := &field{}
	f.addType(&scalarT{})
	assert.Empty(t, f.Imports())

	f.addType(&mockT{i: []Package{&pkg{}, &pkg{}}})
	assert.Len(t, f.Imports(), 2)
}

type mockField struct {
	Field
	i   []Package
	m   Message
	err error
}

func (f *mockField) Imports() []Package { return f.i }

func (f *mockField) setMessage(m Message) { f.m = m }

func (f *mockField) accept(v Visitor) error {
	_, err := v.VisitField(f)
	if f.err != nil {
		return f.err
	}
	return err
}

func dummyField() *field {
	m := dummyMsg()
	str := descriptor.FieldDescriptorProto_TYPE_STRING
	f := &field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("field"), Type: &str}}
	m.addField(f)
	t := &scalarT{name: "string"}
	f.addType(t)
	return f
}
