package pgs

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/stretchr/testify/assert"
)

func TestPluginBase_Name(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { new(PluginBase).Name() })
}

func TestPluginBase_InitDebugger(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	g := Init()
	pb.InitContext(Context(g.Debugger, Parameters{}, "."))

	assert.NotNil(t, pb.BuildContext)
}

func TestPluginBase_Init(t *testing.T) {
	t.Parallel()

	g := generator.New()
	pb := new(PluginBase)
	pb.Init(g)
	assert.Equal(t, g, pb.Generator.Unwrap())
}

func TestPluginBase_Generate(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { new(PluginBase).Generate(new(generator.FileDescriptor)) })
}

func TestPluginBase_Imports(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	pb.Init(generator.New())

	f1 := pb.AddImport("foo", "bar", nil)
	f2 := pb.AddImport("foo", "bar", nil)
	f3 := pb.AddImport("foo", "baz", nil)

	assert.Equal(t, pb.seenImports, pb.Imports)
	assert.Len(t, pb.Imports, 2)

	assert.Equal(t, f1, f2)
	assert.NotEqual(t, f2, f3)
	assert.Equal(t, f1, pb.Imports["bar"])
	assert.Equal(t, f3, pb.Imports["baz"])

	assert.NotPanics(t, func() { pb.GenerateImports(nil) })
	assert.Empty(t, pb.Imports)
	assert.Len(t, pb.seenImports, 2)

	assert.NotPanics(t, func() { pb.GenerateImports(nil) })
}

func TestPluginBase_P(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	pb.Init(generator.New())
	pgg := &pluginProtocGenGo{ProtocGenGo: pb.Generator}
	pb.Generator = pgg

	pb.P("foo", 123)
	assert.Len(t, pgg.p, 2)
	assert.Equal(t, "foo", pgg.p[0])
	assert.Equal(t, 123, pgg.p[1])
}

func TestPluginBase_In(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	pb.Init(generator.New())
	pgg := &pluginProtocGenGo{ProtocGenGo: pb.Generator}
	pb.Generator = pgg

	pb.In()
	assert.Equal(t, 1, pgg.in)
	pb.Out()
	assert.Equal(t, 0, pgg.in)
}

func TestPluginBase_C(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in []interface{}
		ex []interface{}
	}{
		{
			[]interface{}{"foo", " bar", " baz"},
			[]interface{}{"// ", "foo bar baz"},
		},
		{
			in: []interface{}{"the quick brown fox jumps over the lazy dog"},
			ex: []interface{}{"// ", "the quick brown", "// ", "fox jumps over", "// ", "the lazy dog"},
		},
		{
			in: []interface{}{"supercalifragilisticexpialidocious"},
			ex: []interface{}{"// ", "supercalifragilisticexpialidocious"},
		},
		{
			in: []interface{}{"1234567890123456789012345 foo"},
			ex: []interface{}{"// ", "1234567890123456789012345", "// ", "foo"},
		},
	}

	pb := new(PluginBase)
	pb.Init(generator.New())
	pgg := &pluginProtocGenGo{ProtocGenGo: pb.Generator}
	pb.Generator = pgg

	for i, test := range tests {
		tc := test
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pgg.p = pgg.p[:0]
			pb.C(20, tc.in...)
			assert.Equal(t, tc.ex, pgg.p)
		})
	}
}

func TestPluginBase_C80(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	pb.Init(generator.New())
	pgg := &pluginProtocGenGo{ProtocGenGo: pb.Generator}
	pb.Generator = pgg

	pb.C80(strings.Repeat("foo ", 20))
	assert.Equal(t, []interface{}{
		"// ", strings.TrimSpace(strings.Repeat("foo ", 19)),
		"// ", "foo",
	}, pgg.p)
}

func TestPluginBase_T(t *testing.T) {
	t.Parallel()

	tpl := template.Must(template.New("tpl").Parse(`foo{{ . }}`))

	pb := new(PluginBase)
	g := generator.New()
	g.Buffer = new(bytes.Buffer)
	pb.Init(g)

	assert.NotPanics(t, func() { pb.T(tpl, "bar") })
	assert.Contains(t, g.Buffer.String(), "foobar")
}

func TestPluginBase_PushPop(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	pb.Init(generator.New())

	pb.Push("foo")
	pb.Pop()
}

func TestPluginBase_PushPopDir(t *testing.T) {
	t.Parallel()

	pb := new(PluginBase)
	pb.Init(generator.New())

	pb.PushDir("foo/bar")
	assert.Equal(t, "foo/bar", pb.OutputPath())
	pb.PopDir()
	assert.Equal(t, ".", pb.OutputPath())
}

func TestPluginBase_BuildTarget(t *testing.T) {
	t.Parallel()

	g := generator.New()
	g.Request.FileToGenerate = []string{"foo"}

	pb := new(PluginBase)
	pb.Init(g)

	o := mockGeneratorObj{f: &descriptor.FileDescriptorProto{Name: proto.String("foo")}}

	assert.True(t, pb.BuildTarget("foo"))
	assert.True(t, pb.BuildTargetObj(o))

	o.f.Name = proto.String("bar")
	assert.False(t, pb.BuildTargetObj(o))
}

type mockPlugin struct {
	*PluginBase
	name string
}

func (p mockPlugin) Name() string { return p.name }

type pluginProtocGenGo struct {
	ProtocGenGo
	p  []interface{}
	in int
}

func (p *pluginProtocGenGo) Name() string          { return "pluginProtocGenGo" }
func (p *pluginProtocGenGo) P(args ...interface{}) { p.p = append(p.p, args...) }
func (p *pluginProtocGenGo) In()                   { p.in++ }
func (p *pluginProtocGenGo) Out()                  { p.in-- }

type mockGeneratorObj struct {
	generator.Object
	f *descriptor.FileDescriptorProto
}

func (o mockGeneratorObj) File() *descriptor.FileDescriptorProto { return o.f }
