package pgs

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/stretchr/testify/assert"
)

func TestGoPackageOption(t *testing.T) {
	t.Parallel()

	fd := &generator.FileDescriptor{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Options: &descriptor.FileOptions{}}}

	impPath, pkg, ok := goPackageOption(fd)
	assert.Empty(t, impPath)
	assert.Empty(t, pkg)
	assert.False(t, ok)

	fd.Options.GoPackage = proto.String("foobar")
	impPath, pkg, ok = goPackageOption(fd)
	assert.Empty(t, impPath)
	assert.Equal(t, "foobar", pkg)
	assert.True(t, ok)

	fd.Options.GoPackage = proto.String("fizz/buzz")
	impPath, pkg, ok = goPackageOption(fd)
	assert.Equal(t, "fizz/buzz", impPath)
	assert.Equal(t, "buzz", pkg)
	assert.True(t, ok)

	fd.Options.GoPackage = proto.String("foo/bar;baz")
	impPath, pkg, ok = goPackageOption(fd)
	assert.Equal(t, "foo/bar", impPath)
	assert.Equal(t, "baz", pkg)
	assert.True(t, ok)
}

func TestGoFileName(t *testing.T) {
	t.Parallel()

	fd := &generator.FileDescriptor{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("dir/file.proto"),
			Options: &descriptor.FileOptions{},
		},
	}

	assert.Equal(t, "dir/file.pb.go", goFileName(fd))

	fd.FileDescriptorProto.Options.GoPackage = proto.String("other/path")
	assert.Equal(t, "other/path/file.pb.go", goFileName(fd))
}

func TestGoImportPath(t *testing.T) {
	t.Parallel()

	fd := &generator.FileDescriptor{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("dir/file.proto"),
			Options: &descriptor.FileOptions{},
		},
	}

	g := &generator.Generator{ImportMap: map[string]string{}}

	assert.Equal(t, "dir", goImportPath(g, fd))

	g.ImportMap[fd.GetName()] = "other/pkg"
	g.ImportPrefix = "github.com/example"

	assert.Equal(t, "github.com/example/other/pkg", goImportPath(g, fd))
}
