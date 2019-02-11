package pgs

import (
	"testing"

	"text/template"

	"github.com/stretchr/testify/assert"
)

type mockModule struct {
	*ModuleBase
	name     string
	executed bool
}

func newMockModule() *mockModule { return &mockModule{ModuleBase: &ModuleBase{}} }

func (m *mockModule) Name() string { return m.name }

func (m *mockModule) Execute(pkg Package, pkgs map[string]Package) []Artifact {
	m.executed = true
	return nil
}

type multiMockModule struct {
	*mockModule
	multiExecuted bool
}

func newMultiMockModule() *multiMockModule { return &multiMockModule{mockModule: newMockModule()} }

func (m *multiMockModule) MultiExecute(targets map[string]Package, packages map[string]Package) []Artifact {
	m.multiExecuted = true
	return nil
}

func TestModuleBase_InitContext(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	assert.Nil(t, m.BuildContext)
	bc := Context(newMockDebugger(t), Parameters{}, ".")
	m.InitContext(bc)
	assert.NotNil(t, m.BuildContext)
}

func TestModuleBase_Name(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	assert.Panics(t, func() { m.Name() })
}

func TestModuleBase_Execute(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	d := newMockDebugger(t)
	m.InitContext(Context(d, Parameters{}, "."))

	assert.NotPanics(t, func() { m.Execute(nil, nil) })
	assert.True(t, d.failed)
}

func TestModuleBase_PushPop(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.InitContext(Context(newMockDebugger(t), Parameters{}, "."))
	m.Push("foo")
	m.Pop()
}

func TestModuleBase_PushPopDir(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.InitContext(Context(newMockDebugger(t), Parameters{}, "foo"))
	m.PushDir("bar")
	assert.Equal(t, "foo/bar", m.OutputPath())
	m.PopDir()
	assert.Equal(t, "foo", m.OutputPath())
}

func TestModuleBase_Artifacts(t *testing.T) {
	t.Parallel()

	arts := []Artifact{nil, nil, nil}
	m := &ModuleBase{artifacts: arts}
	assert.Equal(t, arts, m.Artifacts())
	assert.Empty(t, m.Artifacts())
}

func TestModuleBase_AddArtifact(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	assert.Empty(t, m.Artifacts())
	m.AddArtifact(nil, nil)
	assert.Len(t, m.Artifacts(), 2)
}

func TestModuleBase_AddGeneratorFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddGeneratorFile("foo", "bar")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorFile{
		Name:     "foo",
		Contents: "bar",
	}, arts[0])
}

func TestModuleBase_OverwriteGeneratorFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.OverwriteGeneratorFile("foo", "bar")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorFile{
		Name:      "foo",
		Contents:  "bar",
		Overwrite: true,
	}, arts[0])
}

func TestModuleBase_AddGeneratorTemplateFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddGeneratorTemplateFile("foo", template.New("bar"), "baz")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorTemplateFile{
		Name: "foo",
		TemplateArtifact: TemplateArtifact{
			Template: template.New("bar"),
			Data:     "baz",
		},
	}, arts[0])
}

func TestModuleBase_OverwriteGeneratorTemplateFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.OverwriteGeneratorTemplateFile("foo", template.New("bar"), "baz")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorTemplateFile{
		Name:      "foo",
		Overwrite: true,
		TemplateArtifact: TemplateArtifact{
			Template: template.New("bar"),
			Data:     "baz",
		},
	}, arts[0])
}

func TestModuleBase_AddGeneratorAppend(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddGeneratorAppend("foo", "bar")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorAppend{
		FileName: "foo",
		Contents: "bar",
	}, arts[0])
}

func TestModuleBase_AddGeneratorTemplateAppend(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddGeneratorTemplateAppend("foo", template.New("bar"), "baz")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorTemplateAppend{
		FileName: "foo",
		TemplateArtifact: TemplateArtifact{
			Template: template.New("bar"),
			Data:     "baz",
		},
	}, arts[0])
}

func TestModuleBase_AddGeneratorInjection(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddGeneratorInjection("foo", "bar", "baz")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorInjection{
		FileName:       "foo",
		InsertionPoint: "bar",
		Contents:       "baz",
	}, arts[0])
}

func TestModuleBase_AddGeneratorTemplateInjection(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddGeneratorTemplateInjection("foo", "bar", template.New("fizz"), "buzz")
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, GeneratorTemplateInjection{
		FileName:       "foo",
		InsertionPoint: "bar",
		TemplateArtifact: TemplateArtifact{
			Template: template.New("fizz"),
			Data:     "buzz",
		},
	}, arts[0])
}

func TestModuleBase_AddCustomFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddCustomFile("foo", "bar", 0765)
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, CustomFile{
		Name:     "foo",
		Contents: "bar",
		Perms:    0765,
	}, arts[0])
}

func TestModuleBase_OverwriteCustomFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.OverwriteCustomFile("foo", "bar", 0765)
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, CustomFile{
		Name:      "foo",
		Contents:  "bar",
		Overwrite: true,
		Perms:     0765,
	}, arts[0])
}

func TestModuleBase_AddCustomTemplateFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.AddCustomTemplateFile("foo", template.New("bar"), "baz", 0765)
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, CustomTemplateFile{
		Name:  "foo",
		Perms: 0765,
		TemplateArtifact: TemplateArtifact{
			Template: template.New("bar"),
			Data:     "baz",
		},
	}, arts[0])
}

func TestModuleBase_OverwriteCustomTemplateFile(t *testing.T) {
	t.Parallel()

	m := new(ModuleBase)
	m.OverwriteCustomTemplateFile("foo", template.New("bar"), "baz", 0765)
	arts := m.Artifacts()
	assert.Len(t, arts, 1)
	assert.Equal(t, CustomTemplateFile{
		Name:      "foo",
		Overwrite: true,
		Perms:     0765,
		TemplateArtifact: TemplateArtifact{
			Template: template.New("bar"),
			Data:     "baz",
		},
	}, arts[0])
}
