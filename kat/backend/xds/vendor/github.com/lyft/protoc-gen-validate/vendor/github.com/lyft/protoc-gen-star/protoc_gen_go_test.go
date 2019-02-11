package pgs

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/assert"
)

func TestWrappedPGG_PackageName(t *testing.T) {
	t.Parallel()

	fd := mockPackageFD{pn: "foo"}

	pgg := Wrap(generator.New())
	assert.Equal(t, "foo", pgg.packageName(fd))
}

func TestWrappedPGG_SetRequest(t *testing.T) {
	t.Parallel()

	wrapped := Wrap(&generator.Generator{})

	assert.Nil(t, wrapped.request())

	req := &plugin_go.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	wrapped.setRequest(req)

	assert.Equal(t, req, wrapped.request())
}

func TestWrappedPGG_SetResponse(t *testing.T) {
	t.Parallel()

	wrapped := Wrap(&generator.Generator{})

	assert.Nil(t, wrapped.response())

	res := &plugin_go.CodeGeneratorResponse{Error: proto.String("foo")}
	wrapped.setResponse(res)

	assert.Equal(t, res, wrapped.response())
}
