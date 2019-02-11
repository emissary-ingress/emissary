package pgs

import (
	"bytes"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	t.Parallel()

	b := &bytes.Buffer{}
	bb := &bytes.Buffer{}
	g := Init(ProtocInput(b), ProtocOutput(bb), func(g *Generator) { /* noop */ })

	assert.NotNil(t, g)
	assert.Equal(t, g.in, b)
	assert.Equal(t, g.out, bb)

	g = Init()
	assert.Equal(t, os.Stdin, g.in)
	assert.Equal(t, os.Stdout, g.out)

	_, ok := g.workflow.(*onceWorkflow)
	assert.True(t, ok)
}

func TestGenerator_RegisterPlugin(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	g := &Generator{Debugger: d}
	p := mockPlugin{PluginBase: &PluginBase{}, name: "foo"}
	g.RegisterPlugin(p)

	assert.False(t, d.failed)
	assert.Len(t, g.plugins, 1)
	assert.Equal(t, p, g.plugins[0])

	assert.Panics(t, func() { g.RegisterPlugin(nil) })
	assert.True(t, d.failed)
}

func TestGenerator_RegisterModule(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	g := &Generator{Debugger: d}

	assert.Empty(t, g.mods)
	g.RegisterModule(&mockModule{name: "foo"})

	assert.False(t, d.failed)
	assert.Len(t, g.mods, 1)

	assert.Panics(t, func() { g.RegisterModule(nil) })
	assert.True(t, d.failed)
}

func TestGenerator_RegisterPostProcessor(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	p := newPersister()
	g := &Generator{Debugger: d, persister: p}

	assert.Empty(t, p.procs)
	g.RegisterPostProcessor(GoFmt())

	assert.False(t, d.failed)
	assert.Len(t, p.procs, 1)

	g.RegisterPostProcessor(nil)
	assert.True(t, d.failed)
}

func TestGenerator_AST(t *testing.T) {
	t.Parallel()

	g := Init()
	g.workflow = &dummyWorkflow{}

	pkg := dummyPkg()
	pkgName := pkg.GoName().String()

	g.gatherer.targets = map[string]Package{pkgName: pkg}
	g.gatherer.pkgs = map[string]Package{"foo": nil}

	targets, pkgs := g.AST()
	assert.Equal(t, g.gatherer.targets[pkgName], targets[pkgName])
	assert.Equal(t, g.gatherer.pkgs, pkgs)
}

func TestGenerator_Render(t *testing.T) {
	// cannot be parallel

	req := &plugin_go.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	b, err := proto.Marshal(req)
	assert.NoError(t, err)

	buf := &bytes.Buffer{}
	g := Init(ProtocInput(bytes.NewReader(b)), ProtocOutput(buf))
	g.pgg = mockGeneratorPGG{ProtocGenGo: g.pgg}
	g.gatherer.targets = map[string]Package{"foo": &pkg{}}
	g.pgg.response().File = []*plugin_go.CodeGeneratorResponse_File{{}}

	assert.NotPanics(t, g.Render)

	var res plugin_go.CodeGeneratorResponse
	assert.NoError(t, proto.Unmarshal(buf.Bytes(), &res))
	assert.True(t, proto.Equal(g.pgg.response(), &res))
}

func TestGenerator_PushPop(t *testing.T) {
	t.Parallel()

	g := Init()
	g.push("foo")

	pd, ok := g.Debugger.(prefixedDebugger)
	assert.True(t, ok)
	assert.Equal(t, "[foo]", pd.prefix)

	g.pop()

	_, ok = g.Debugger.(rootDebugger)
	assert.True(t, ok)
}

type mockGeneratorPGG struct {
	ProtocGenGo
}

func (pgg mockGeneratorPGG) Error(err error, msgs ...string) {}
func (pgg mockGeneratorPGG) Fail(msgs ...string)             {}
func (pgg mockGeneratorPGG) prepare(param Parameters)        {}
func (pgg mockGeneratorPGG) generate()                       {}
