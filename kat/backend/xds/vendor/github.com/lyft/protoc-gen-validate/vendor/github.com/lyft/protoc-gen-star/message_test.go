package pgs

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/stretchr/testify/assert"
)

func TestMsg_Name(t *testing.T) {
	t.Parallel()

	m := &msg{rawDesc: &descriptor.DescriptorProto{Name: proto.String("msg")}}

	assert.Equal(t, "msg", m.Name().String())
}

func TestMsg_TypeName(t *testing.T) {
	t.Parallel()

	m := dummyMsg()
	assert.Equal(t, m.Name().String(), m.TypeName().String())
}

func TestMsg_Syntax(t *testing.T) {
	t.Parallel()

	m := &msg{}
	f := dummyFile()
	f.addMessage(m)

	assert.Equal(t, f.Syntax(), m.Syntax())
}

func TestMsg_Package(t *testing.T) {
	t.Parallel()

	m := &msg{}
	f := dummyFile()
	f.addMessage(m)

	assert.NotNil(t, m.Package())
	assert.Equal(t, f.Package(), m.Package())
}

func TestMsg_File(t *testing.T) {
	t.Parallel()

	m := &msg{}
	pm := dummyMsg()
	pm.addMessage(m)

	assert.NotNil(t, m.File())
	assert.Equal(t, pm.File(), m.File())
}

func TestMsg_BuildTarget(t *testing.T) {
	t.Parallel()

	m := &msg{}
	f := dummyFile()
	f.addMessage(m)

	assert.False(t, m.BuildTarget())
	f.buildTarget = true
	assert.True(t, m.BuildTarget())
}

func TestMsg_Descriptor(t *testing.T) {
	t.Parallel()

	m := &msg{genDesc: &generator.Descriptor{}}
	assert.Equal(t, m.genDesc, m.Descriptor())
}

func TestMsg_Parent(t *testing.T) {
	t.Parallel()

	m := &msg{}
	pm := dummyMsg()
	pm.addMessage(m)

	assert.Equal(t, pm, m.Parent())
}

func TestMsg_IsMapEntry(t *testing.T) {
	t.Parallel()

	m := &msg{rawDesc: &descriptor.DescriptorProto{}}
	assert.False(t, m.IsMapEntry())

	m.rawDesc.Options = &descriptor.MessageOptions{
		MapEntry: proto.Bool(true),
	}
	assert.True(t, m.IsMapEntry())
}

func TestMsg_Enums(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.Enums())

	sm := &msg{}
	sm.addEnum(&enum{})
	m.addMessage(sm)

	m.addEnum(&enum{})
	assert.Len(t, m.Enums(), 1)
}

func TestMsg_AllEnums(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.AllEnums())

	sm := &msg{}
	sm.addEnum(&enum{})
	m.addMessage(sm)

	m.addEnum(&enum{})
	assert.Len(t, m.AllEnums(), 2)
}

func TestMsg_Messages(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.Messages())

	sm := &msg{}
	sm.addMessage(&msg{})
	m.addMessage(sm)

	assert.Len(t, m.Messages(), 1)
}

func TestMsg_AllMessages(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.AllMessages())

	sm := &msg{}
	sm.addMessage(&msg{})
	m.addMessage(sm)

	assert.Len(t, m.AllMessages(), 2)
}

func TestMsg_MapEntries(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.MapEntries())

	m.addMapEntry(&msg{})
	assert.Len(t, m.MapEntries(), 1)
}

func TestMsg_Fields(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.Fields())

	m.addField(&field{})
	m.addField(&field{oneof: &oneof{}})
	assert.Len(t, m.Fields(), 2)
}

func TestMsg_NonOneOfFields(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.NonOneOfFields())

	m.addField(&field{})
	m.addField(&field{oneof: &oneof{}})
	m.addField(&field{})
	assert.Len(t, m.NonOneOfFields(), 2)
}

func TestMsg_OneOfFields(t *testing.T) {
	t.Parallel()

	o := &oneof{}
	o.addField(&field{})

	m := &msg{}
	m.addField(&field{})
	m.addField(&field{})

	assert.Empty(t, m.OneOfFields())
	m.addOneOf(o)
	assert.Len(t, m.OneOfFields(), 1)
}

func TestMsg_OneOfs(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.OneOfs())

	m.addOneOf(&oneof{})
	assert.Len(t, m.OneOfs(), 1)
}

func TestMsg_Extension(t *testing.T) {
	// cannot be parallel
	m := &msg{rawDesc: &descriptor.DescriptorProto{}}
	assert.NotPanics(t, func() { m.Extension(nil, nil) })
}

func TestMsg_Accept(t *testing.T) {
	t.Parallel()

	m := &msg{}
	m.addMessage(&msg{})
	m.addEnum(&enum{})
	m.addField(&field{})
	m.addOneOf(&oneof{})

	assert.NoError(t, m.accept(nil))

	v := &mockVisitor{}
	assert.NoError(t, m.accept(v))
	assert.Equal(t, 1, v.message)
	assert.Zero(t, v.enum)
	assert.Zero(t, v.field)
	assert.Zero(t, v.oneof)

	v.Reset()
	v.v = v
	v.err = errors.New("")
	assert.Error(t, m.accept(v))
	assert.Equal(t, 1, v.message)
	assert.Zero(t, v.enum)
	assert.Zero(t, v.field)
	assert.Zero(t, v.oneof)

	v.Reset()
	assert.NoError(t, m.accept(v))
	assert.Equal(t, 2, v.message)
	assert.Equal(t, 1, v.enum)
	assert.Equal(t, 1, v.field)
	assert.Equal(t, 1, v.oneof)

	v.Reset()
	m.addOneOf(&mockOneOf{err: errors.New("")})
	assert.Error(t, m.accept(v))
	assert.Equal(t, 2, v.message)
	assert.Equal(t, 1, v.enum)
	assert.Equal(t, 1, v.field)
	assert.Equal(t, 2, v.oneof)

	v.Reset()
	m.addField(&mockField{err: errors.New("")})
	assert.Error(t, m.accept(v))
	assert.Equal(t, 2, v.message)
	assert.Equal(t, 1, v.enum)
	assert.Equal(t, 2, v.field)
	assert.Zero(t, v.oneof)

	v.Reset()
	m.addMessage(&mockMessage{err: errors.New("")})
	assert.Error(t, m.accept(v))
	assert.Equal(t, 3, v.message)
	assert.Equal(t, 1, v.enum)
	assert.Zero(t, v.field)
	assert.Zero(t, v.oneof)

	v.Reset()
	m.addEnum(&mockEnum{err: errors.New("")})
	assert.Error(t, m.accept(v))
	assert.Equal(t, 2, v.enum)
	assert.Equal(t, 1, v.message)
	assert.Zero(t, v.field)
	assert.Zero(t, v.oneof)
}

func TestMsg_Imports(t *testing.T) {
	t.Parallel()

	m := &msg{}
	assert.Empty(t, m.Imports())

	m.addField(&mockField{i: []Package{&pkg{}, &pkg{}}})
	assert.Len(t, m.Imports(), 2)
}

func TestMsg_LookupName(t *testing.T) {
	t.Parallel()

	m := &msg{rawDesc: &descriptor.DescriptorProto{Name: proto.String("msg")}}
	f := dummyFile()
	f.addMessage(m)

	assert.Equal(t, f.lookupName()+".msg", m.lookupName())
}

type mockMessage struct {
	Message
	i   []Package
	p   ParentEntity
	err error
}

func (m *mockMessage) Imports() []Package { return m.i }

func (m *mockMessage) setParent(p ParentEntity) { m.p = p }

func (m *mockMessage) accept(v Visitor) error {
	_, err := v.VisitMessage(m)
	if m.err != nil {
		return m.err
	}
	return err
}

func dummyMsg() *msg {
	f := dummyFile()

	m := &msg{
		rawDesc: &descriptor.DescriptorProto{Name: proto.String("message")},
	}
	m.genDesc = &generator.Descriptor{DescriptorProto: m.rawDesc}

	f.addMessage(m)
	return m
}
