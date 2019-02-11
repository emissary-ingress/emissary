package pgs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockNode struct {
	Node

	a func(Visitor) error
}

func (n mockNode) accept(v Visitor) error { return n.a(v) }

func TestWalk(t *testing.T) {
	t.Parallel()

	e := errors.New("TestWalk")
	type mockVisitor struct{ Visitor }
	mv := mockVisitor{}
	n := mockNode{}

	n.a = func(v Visitor) error {
		assert.Equal(t, mv, v)
		return e
	}

	assert.Equal(t, e, Walk(mv, n))
}

func TestNilVisitor(t *testing.T) {
	t.Parallel()

	nv := NilVisitor()

	v, err := nv.VisitPackage(&pkg{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitFile(&file{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitService(&service{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitMethod(&method{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitEnum(&enum{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitEnumValue(&enumVal{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitMessage(&msg{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitField(&field{})
	assert.Nil(t, v)
	assert.NoError(t, err)

	v, err = nv.VisitOneOf(&oneof{})
	assert.Nil(t, v)
	assert.NoError(t, err)
}

func TestPassThroughVisitor(t *testing.T) {
	t.Parallel()

	nv := NilVisitor()
	pv := PassThroughVisitor(nv)

	v, err := pv.VisitPackage(&pkg{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitFile(&file{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitService(&service{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitMethod(&method{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitEnum(&enum{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitEnumValue(&enumVal{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitMessage(&msg{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitField(&field{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)

	v, err = pv.VisitOneOf(&oneof{})
	assert.Equal(t, nv, v)
	assert.NoError(t, err)
}

type mockVisitor struct {
	v   Visitor
	err error

	pkg, file, message, enum, enumvalue, field, oneof, service, method int
}

func (v *mockVisitor) VisitPackage(p Package) (w Visitor, err error) {
	v.pkg++
	return v.v, v.err
}

func (v *mockVisitor) VisitFile(f File) (w Visitor, err error) {
	v.file++
	return v.v, v.err
}

func (v *mockVisitor) VisitMessage(m Message) (w Visitor, err error) {
	v.message++
	return v.v, v.err
}

func (v *mockVisitor) VisitEnum(e Enum) (w Visitor, err error) {
	v.enum++
	return v.v, v.err
}

func (v *mockVisitor) VisitEnumValue(ev EnumValue) (w Visitor, err error) {
	v.enumvalue++
	return v.v, v.err
}

func (v *mockVisitor) VisitField(f Field) (w Visitor, err error) {
	v.field++
	return v.v, v.err
}

func (v *mockVisitor) VisitOneOf(o OneOf) (w Visitor, err error) {
	v.oneof++
	return v.v, v.err
}

func (v *mockVisitor) VisitService(s Service) (w Visitor, err error) {
	v.service++
	return v.v, v.err
}

func (v *mockVisitor) VisitMethod(m Method) (w Visitor, err error) {
	v.method++
	return v.v, v.err
}

func (v *mockVisitor) Reset() {
	*v = mockVisitor{v: v.v}
}
