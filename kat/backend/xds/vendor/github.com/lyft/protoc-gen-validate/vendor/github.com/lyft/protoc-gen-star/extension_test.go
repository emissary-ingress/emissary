package pgs

import (
	"bytes"
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

type mockExtractor struct {
	has bool
	get interface{}
	err error
}

func (e *mockExtractor) HasExtension(proto.Message, *proto.ExtensionDesc) bool { return e.has }

func (e *mockExtractor) GetExtension(proto.Message, *proto.ExtensionDesc) (interface{}, error) {
	return e.get, e.err
}

var testExtractor = &mockExtractor{}

func init() { extractor = testExtractor }

func TestExtension(t *testing.T) {
	// cannot be parallel

	defer func() { testExtractor.get = nil }()

	found, err := extension(nil, nil, nil)
	assert.False(t, found)
	assert.NoError(t, err)

	found, err = extension(proto.Message(nil), nil, nil)
	assert.False(t, found)
	assert.NoError(t, err)

	opts := &struct{ proto.Message }{}

	found, err = extension(opts, nil, nil)
	assert.False(t, found)
	assert.Error(t, err)

	desc := &proto.ExtensionDesc{}

	found, err = extension(opts, desc, nil)
	assert.False(t, found)
	assert.Error(t, err)

	type myExt struct{ Name string }

	found, err = extension(opts, desc, &myExt{})
	assert.False(t, found)
	assert.NoError(t, err)

	testExtractor.has = true

	found, err = extension(opts, desc, &myExt{})
	assert.False(t, found)
	assert.NoError(t, err)

	testExtractor.err = errors.New("foo")

	found, err = extension(opts, desc, &myExt{})
	assert.False(t, found)
	assert.Error(t, err)

	testExtractor.err = nil
	testExtractor.get = &myExt{"bar"}

	out := myExt{}

	found, err = extension(opts, desc, out)
	assert.False(t, found)
	assert.Error(t, err)

	found, err = extension(opts, desc, &out)
	assert.True(t, found)
	assert.NoError(t, err)
	assert.Equal(t, "bar", out.Name)

	var ref *myExt
	found, err = extension(opts, desc, &ref)
	assert.True(t, found)
	assert.NoError(t, err)
	assert.Equal(t, "bar", ref.Name)

	found, err = extension(opts, desc, &bytes.Buffer{})
	assert.True(t, found)
	assert.Error(t, err)
}

func TestProtoExtExtractor(t *testing.T) {
	e := protoExtExtractor{}
	assert.NotPanics(t, func() { e.HasExtension(nil, nil) })
	assert.NotPanics(t, func() { e.GetExtension(nil, nil) })
}
