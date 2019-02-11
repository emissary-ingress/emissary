package pgs

import (
	"testing"

	"errors"

	"github.com/stretchr/testify/assert"
)

func TestPkg_ProtoName(t *testing.T) {
	t.Parallel()

	p := &pkg{fd: mockPackageFD{gp: "foobar"}}
	assert.Equal(t, Name("foobar"), p.ProtoName())
}

func TestPkg_GoName(t *testing.T) {
	t.Parallel()

	p := &pkg{name: "foobar"}
	assert.Equal(t, Name("foobar"), p.GoName())
}

func TestPkg_ImportPath(t *testing.T) {
	t.Parallel()

	p := &pkg{importPath: "fizz/buzz"}
	assert.Equal(t, "fizz/buzz", p.ImportPath())
}

func TestPkg_Files(t *testing.T) {
	t.Parallel()

	p := &pkg{}
	assert.Empty(t, p.Files())

	p.addFile(&file{})
	p.addFile(&file{})
	p.addFile(&file{})

	assert.Len(t, p.Files(), 3)
}

func TestPkg_AddFile(t *testing.T) {
	t.Parallel()

	p := &pkg{}
	f := &file{}
	p.addFile(f)
	assert.Len(t, p.files, 1)
	assert.EqualValues(t, f, p.files[0])
}

func TestPkg_Accept(t *testing.T) {
	t.Parallel()

	p := &pkg{
		files: []File{&mockFile{}},
	}
	assert.Nil(t, p.accept(nil))

	v := &mockVisitor{}
	assert.NoError(t, p.accept(v))
	assert.Equal(t, 1, v.pkg)
	assert.Zero(t, v.file)

	v.Reset()
	v.err = errors.New("foobar")
	assert.EqualError(t, p.accept(v), "foobar")
	assert.Equal(t, 1, v.pkg)
	assert.Zero(t, v.file)

	v.Reset()
	v.v = v
	assert.NoError(t, p.accept(v))
	assert.Equal(t, 1, v.pkg)
	assert.Equal(t, 1, v.file)

	v.Reset()
	p.addFile(&mockFile{err: errors.New("fizzbuzz")})
	assert.EqualError(t, p.accept(v), "fizzbuzz")
	assert.Equal(t, 1, v.pkg)
	assert.Equal(t, 2, v.file)
}

type mockPackageFD struct {
	packageFD
	pn string
	gp string
}

func (mp mockPackageFD) PackageName() string { return mp.pn }
func (mp mockPackageFD) GetPackage() string  { return mp.gp }

func dummyPkg() *pkg {
	return &pkg{
		fd: &mockPackageFD{
			pn: "pkg_name",
			gp: "get_pkg",
		},
		importPath: "import/path",
	}
}
