package pgs

import (
	"html/template"
	"testing"

	"errors"

	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestPersister_Persist_Unrecognized(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)

	p.Persist(nil)

	assert.True(t, d.failed)
}

func TestPersister_Persist_GeneratorFile(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(GeneratorFile{
		Name:     "foo",
		Contents: "bar",
	})

	assert.Len(t, p.pgg.response().File, 1)
	assert.Equal(t, "foo", p.pgg.response().File[0].GetName())
	assert.Equal(t, "bar", p.pgg.response().File[0].GetContent())

	p.Persist(GeneratorFile{
		Name:     "foo",
		Contents: "baz",
	})

	assert.Len(t, p.pgg.response().File, 2)

	p.Persist(GeneratorFile{
		Name:      "foo",
		Contents:  "fizz",
		Overwrite: true,
	})

	assert.Len(t, p.pgg.response().File, 2)
	assert.Equal(t, "fizz", p.pgg.response().File[0].GetContent())
}

var genTpl = template.Must(template.New("good").Parse("{{ . }}"))

func TestPersister_Persist_GeneratorTemplateFile(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(GeneratorTemplateFile{
		Name: "foo",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "bar",
		},
	})

	assert.Len(t, p.pgg.response().File, 1)
	assert.Equal(t, "foo", p.pgg.response().File[0].GetName())
	assert.Equal(t, "bar", p.pgg.response().File[0].GetContent())

	p.Persist(GeneratorTemplateFile{
		Name: "foo",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "baz",
		},
	})

	assert.Len(t, p.pgg.response().File, 2)

	p.Persist(GeneratorTemplateFile{
		Name: "foo",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "fizz",
		},
		Overwrite: true,
	})

	assert.Len(t, p.pgg.response().File, 2)
	assert.Equal(t, "fizz", p.pgg.response().File[0].GetContent())
}

func TestPersister_Persist_GeneratorAppend(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(
		GeneratorFile{Name: "foo"},
		GeneratorFile{Name: "bar"},
	)

	p.Persist(GeneratorAppend{
		FileName: "foo",
		Contents: "baz",
	})

	assert.Len(t, p.pgg.response().File, 3)
	assert.Equal(t, "", p.pgg.response().File[1].GetName())
	assert.Equal(t, "baz", p.pgg.response().File[1].GetContent())

	p.Persist(GeneratorAppend{
		FileName: "bar",
		Contents: "quux",
	})

	assert.Len(t, p.pgg.response().File, 4)
	assert.Equal(t, "", p.pgg.response().File[3].GetName())
	assert.Equal(t, "quux", p.pgg.response().File[3].GetContent())

	p.Persist(GeneratorAppend{
		FileName: "doesNotExist",
	})

	assert.True(t, d.failed)
}

func TestPersister_Persist_GeneratorTemplateAppend(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(
		GeneratorFile{Name: "foo"},
		GeneratorFile{Name: "bar"},
	)

	p.Persist(GeneratorTemplateAppend{
		FileName: "foo",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "baz",
		},
	})

	assert.Len(t, p.pgg.response().File, 3)
	assert.Equal(t, "", p.pgg.response().File[1].GetName())
	assert.Equal(t, "baz", p.pgg.response().File[1].GetContent())

	p.Persist(GeneratorTemplateAppend{
		FileName: "bar",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "quux",
		},
	})

	assert.Len(t, p.pgg.response().File, 4)
	assert.Equal(t, "", p.pgg.response().File[3].GetName())
	assert.Equal(t, "quux", p.pgg.response().File[3].GetContent())

	p.Persist(GeneratorTemplateAppend{
		FileName: "doesNotExist",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "baz",
		},
	})

	assert.True(t, d.failed)
}

func TestPersister_Persist_GeneratorInjection(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(GeneratorInjection{
		FileName:       "foo",
		InsertionPoint: "bar",
		Contents:       "baz",
	})

	assert.Len(t, p.pgg.response().File, 1)
	assert.Equal(t, "foo", p.pgg.response().File[0].GetName())
	assert.Equal(t, "bar", p.pgg.response().File[0].GetInsertionPoint())
	assert.Equal(t, "baz", p.pgg.response().File[0].GetContent())
}

func TestPersister_Persist_GeneratorTemplateInjection(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(GeneratorTemplateInjection{
		FileName:       "foo",
		InsertionPoint: "bar",
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "baz",
		},
	})

	assert.Len(t, p.pgg.response().File, 1)
	assert.Equal(t, "foo", p.pgg.response().File[0].GetName())
	assert.Equal(t, "bar", p.pgg.response().File[0].GetInsertionPoint())
	assert.Equal(t, "baz", p.pgg.response().File[0].GetContent())
}

func TestPersister_Persist_CustomFile(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(CustomFile{
		Name:     "foo/bar/baz.txt",
		Perms:    0655,
		Contents: "fizz",
	})

	b, err := afero.ReadFile(fs, "foo/bar/baz.txt")
	assert.NoError(t, err)
	assert.Equal(t, "fizz", string(b))

	p.Persist(CustomFile{
		Name:     "foo/bar/baz.txt",
		Perms:    0655,
		Contents: "buzz",
	})

	b, err = afero.ReadFile(fs, "foo/bar/baz.txt")
	assert.NoError(t, err)
	assert.Equal(t, "fizz", string(b))

	p.Persist(CustomFile{
		Name:      "foo/bar/baz.txt",
		Perms:     0655,
		Contents:  "buzz",
		Overwrite: true,
	})

	b, err = afero.ReadFile(fs, "foo/bar/baz.txt")
	assert.NoError(t, err)
	assert.Equal(t, "buzz", string(b))
}

func TestPersister_Persist_CustomTemplateFile(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := dummyPersister(d)
	fs := afero.NewMemMapFs()
	p.SetFS(fs)

	p.Persist(CustomTemplateFile{
		Name:  "foo/bar/baz.txt",
		Perms: 0655,
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "fizz",
		},
	})

	b, err := afero.ReadFile(fs, "foo/bar/baz.txt")
	assert.NoError(t, err)
	assert.Equal(t, "fizz", string(b))

	p.Persist(CustomTemplateFile{
		Name:  "foo/bar/baz.txt",
		Perms: 0655,
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "buzz",
		},
	})

	b, err = afero.ReadFile(fs, "foo/bar/baz.txt")
	assert.NoError(t, err)
	assert.Equal(t, "fizz", string(b))

	p.Persist(CustomTemplateFile{
		Name:  "foo/bar/baz.txt",
		Perms: 0655,
		TemplateArtifact: TemplateArtifact{
			Template: genTpl,
			Data:     "buzz",
		},
		Overwrite: true,
	})

	b, err = afero.ReadFile(fs, "foo/bar/baz.txt")
	assert.NoError(t, err)
	assert.Equal(t, "buzz", string(b))
}

func TestPersister_AddPostProcessor(t *testing.T) {
	t.Parallel()

	p := dummyPersister(newMockDebugger(t))

	good := &mockPP{match: true, out: []byte("good")}
	bad := &mockPP{err: errors.New("should not be called")}

	p.AddPostProcessor(good, bad)
	out := p.postProcess(GeneratorFile{}, "")
	assert.Equal(t, "good", out)
}

func dummyPersister(d Debugger) *stdPersister {
	return &stdPersister{
		Debugger: d,
		pgg:      mockGeneratorPGG{ProtocGenGo: Wrap(generator.New())},
		fs:       afero.NewMemMapFs(),
	}
}
