package pgs

import (
	"testing"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestScalarE_ParentType(t *testing.T) {
	t.Parallel()

	s := &scalarE{}
	s.setType(&scalarT{})
	assert.Equal(t, s.typ, s.ParentType())
}

func TestScalarE_ProtoType(t *testing.T) {
	t.Parallel()
	s := &scalarE{ptype: ProtoType(descriptor.FieldDescriptorProto_TYPE_BYTES)}
	assert.Equal(t, s.ptype, s.ProtoType())
}

func TestScalarE_IsEmbed(t *testing.T) {
	t.Parallel()
	assert.False(t, (&scalarE{}).IsEmbed())
}

func TestScalarE_IsEnum(t *testing.T) {
	t.Parallel()
	assert.False(t, (&scalarE{}).IsEnum())
}

func TestScalarE_Name(t *testing.T) {
	t.Parallel()
	s := &scalarE{name: TypeName("foobar")}
	assert.Equal(t, s.name, s.Name())
}

func TestScalarE_Imports(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (&scalarE{}).Imports())
}

func TestScalarE_Embed(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (&scalarE{}).Embed())
}

func TestScalarE_Enum(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (&scalarE{}).Enum())
}

func TestEnumE_IsEnum(t *testing.T) {
	t.Parallel()
	assert.True(t, (&enumE{}).IsEnum())
}

func TestEnumE_Enum(t *testing.T) {
	t.Parallel()
	e := &enumE{enum: dummyEnum()}
	assert.Equal(t, e.enum, e.Enum())
}

func TestEnumE_Imports(t *testing.T) {
	t.Parallel()

	e := &enumE{scalarE: &scalarE{}, enum: dummyEnum()}
	f := dummyField()
	e.typ = f.typ

	assert.Empty(t, e.Imports())

	e.enum.File().setPackage(&pkg{name: "not_the_same"})
	assert.Len(t, e.Imports(), 1)
	assert.Equal(t, e.Enum().Package(), e.Imports()[0])
}

func TestEmbedE_IsEmbed(t *testing.T) {
	t.Parallel()
	assert.True(t, (&embedE{}).IsEmbed())
}

func TestEmbedE_Embed(t *testing.T) {
	t.Parallel()
	e := &embedE{msg: dummyMsg()}
	assert.Equal(t, e.msg, e.Embed())
}

func TestEmbedE_Imports(t *testing.T) {
	t.Parallel()

	e := &embedE{scalarE: &scalarE{}, msg: dummyMsg()}
	f := dummyField()
	e.typ = f.typ

	assert.Empty(t, e.Imports())

	e.Embed().File().setPackage(&pkg{name: "not_the_same"})
	assert.Len(t, e.Imports(), 1)
	assert.Equal(t, e.Embed().Package(), e.Imports()[0])
}
