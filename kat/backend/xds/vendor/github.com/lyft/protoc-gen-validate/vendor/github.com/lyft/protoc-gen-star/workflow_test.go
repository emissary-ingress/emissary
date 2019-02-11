package pgs

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/assert"
)

func TestStandardWorkflow_Init(t *testing.T) {
	t.Parallel()

	req := &plugin_go.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	b, err := proto.Marshal(req)
	assert.NoError(t, err)

	mutated := false

	g := Init(ProtocInput(bytes.NewReader(b)), MutateParams(func(p Parameters) { mutated = true }))
	g.workflow.Init(g)

	assert.True(t, proto.Equal(req, g.pgg.request()))
	assert.True(t, mutated)
}

func TestStandardWorkflow_Go(t *testing.T) {
	t.Parallel()

	g := Init()
	g.workflow = &standardWorkflow{Generator: g}
	g.pgg = mockGeneratorPGG{}
	g.params = Parameters{"foo": "bar"}

	g.workflow.Go()
	assert.Equal(t, g.params, g.gatherer.BuildContext.Parameters())
}

func TestStandardWorkflow_Star(t *testing.T) {
	t.Parallel()

	g := Init()
	g.workflow = &standardWorkflow{Generator: g}
	g.params = Parameters{}
	g.gatherer.targets = map[string]Package{"baz": dummyPkg()}

	m := newMockModule()
	m.name = "foo"

	mm := newMultiMockModule()
	mm.name = "bar"

	g.RegisterModule(m, mm)

	g.workflow.Star()

	assert.True(t, m.executed)
	assert.True(t, mm.multiExecuted)
}

func TestStandardWorkflow_Persist(t *testing.T) {
	t.Parallel()

	g := Init(ProtocOutput(ioutil.Discard))
	g.workflow = &standardWorkflow{Generator: g}
	g.persister = dummyPersister(g.Debugger)

	assert.NotPanics(t, g.workflow.Persist)
}

func TestOnceWorkflow(t *testing.T) {
	t.Parallel()

	d := &dummyWorkflow{}
	wf := &onceWorkflow{workflow: d}

	wf.Init(nil)
	wf.Go()
	wf.Star()
	wf.Persist()

	assert.True(t, d.initted)
	assert.True(t, d.goed)
	assert.True(t, d.starred)
	assert.True(t, d.persisted)

	d = &dummyWorkflow{}
	wf.workflow = d

	wf.Init(nil)
	wf.Go()
	wf.Star()
	wf.Persist()

	assert.False(t, d.initted)
	assert.False(t, d.goed)
	assert.False(t, d.starred)
	assert.False(t, d.persisted)
}

func TestExcludeGoWorkflow_Go(t *testing.T) {
	t.Parallel()

	g := &Generator{
		Debugger: newMockDebugger(t),
		pgg: Wrap(&generator.Generator{Response: &plugin_go.CodeGeneratorResponse{
			File: []*plugin_go.CodeGeneratorResponse_File{
				{Name: proto.String("fizz/buzz.pb.go")},
				{Name: proto.String("foo/bar.pb.go")},
				{Name: proto.String("foo/baz.pb.go")},
			},
		}}),
		gatherer: &gatherer{
			targets: map[string]Package{"quux": &pkg{
				files: []File{
					&file{buildTarget: true, outputPath: "foo/bar.pb.go"},
					&file{buildTarget: false, outputPath: "fizz/buzz.pb.go"},
				},
			}},
		},
	}

	wf := &excludeGoWorkflow{Generator: g, workflow: &dummyWorkflow{}}
	wf.Go()

	resp := g.pgg.response()
	assert.Len(t, resp.File, 2)
	assert.Equal(t, "fizz/buzz.pb.go", resp.File[0].GetName())
	assert.Equal(t, "foo/baz.pb.go", resp.File[1].GetName())
}

type dummyWorkflow struct {
	initted, goed, starred, persisted bool
}

func (wf *dummyWorkflow) Init(g *Generator) { wf.initted = true }
func (wf *dummyWorkflow) Go()               { wf.goed = true }
func (wf *dummyWorkflow) Star()             { wf.starred = true }
func (wf *dummyWorkflow) Persist()          { wf.persisted = true }
