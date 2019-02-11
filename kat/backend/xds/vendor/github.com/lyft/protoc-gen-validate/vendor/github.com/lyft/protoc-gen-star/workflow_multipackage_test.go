package pgs

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"context"

	"os"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	protoc "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func multiPackageReq() *protoc.CodeGeneratorRequest {
	return &protoc.CodeGeneratorRequest{
		FileToGenerate: []string{
			"foo",
			"bar",
			"bar/quux",
			"bar/baz",
			"bar/fizz",
			"bar/fizz/buzz",
		},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{Name: proto.String("bar/fizz/buzz")},
			{Name: proto.String("bar/fizz"), Dependency: []string{"bar/fizz/buzz"}},
			{Name: proto.String("bar/baz"), Dependency: []string{"bar/fizz"}},
			{Name: proto.String("bar/quux")},
			{Name: proto.String("bar"), Dependency: []string{"bar/baz"}},
			{Name: proto.String("foo"), Dependency: []string{"bar"}},
		},
	}
}

func TestMultiPackageWorkflow_Init(t *testing.T) {
	wf := &multiPackageWorkflow{workflow: &dummyWorkflow{}}

	g := Init()
	wf.Init(g)

	assert.Equal(t, g, wf.Generator)
	assert.Equal(t, os.Stdout, wf.stdout)
}

func TestMultiPackageWorkflow_Go(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	g := Init()
	g.Debugger = d
	g.pgg = mockGeneratorPGG{g.pgg}

	req := multiPackageReq()
	res := &protoc.CodeGeneratorResponse{Error: proto.String("foo")}

	g.pgg.setRequest(req)

	dwf := &dummyWorkflow{}
	wf := &multiPackageWorkflow{workflow: dwf, spoofFanout: res}
	wf.Init(g)
	wf.Go()

	assert.Equal(t, req, g.pgg.request())
	assert.Equal(t, res, g.pgg.response())
	assert.False(t, dwf.goed)
}

func TestMultiPackageWorkflow_Go_SinglePackage(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	g := Init()
	g.Debugger = d
	g.pgg = mockGeneratorPGG{g.pgg}

	req := &protoc.CodeGeneratorRequest{
		FileToGenerate: []string{"foo", "bar"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{Name: proto.String("bar")},
			{Name: proto.String("foo")},
		},
	}

	g.pgg.setRequest(req)

	dwf := &dummyWorkflow{}
	wf := &multiPackageWorkflow{workflow: dwf}
	wf.Init(g)
	wf.Go()

	assert.True(t, dwf.goed)
}

func TestMultiPackageWorkflow_Go_SubProcess(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	g := Init()
	g.Debugger = d
	g.pgg = mockGeneratorPGG{g.pgg}
	g.params = Parameters{multiPackageSubProcessParam: "true"}

	req := &protoc.CodeGeneratorRequest{
		FileToGenerate: []string{"foo", "bar"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			{Name: proto.String("bar")},
			{Name: proto.String("foo")},
		},
	}

	g.pgg.setRequest(req)

	dwf := &dummyWorkflow{}
	wf := &multiPackageWorkflow{workflow: dwf}
	wf.Init(g)
	wf.stdout = &bytes.Buffer{}
	wf.Go()

	assert.True(t, dwf.goed)
	assert.NoError(t, d.err)
	assert.True(t, d.exited)
}

func TestMultiPackageWorkflow_SplitRequests(t *testing.T) {
	t.Parallel()

	g := Init()
	g.pgg.setRequest(multiPackageReq())
	wf := &multiPackageWorkflow{Generator: g}

	subreqs := wf.splitRequest()
	assert.Len(t, subreqs, 3)

	assert.Len(t, subreqs[0].FileToGenerate, 2)
	assert.Equal(t, "foo", subreqs[0].FileToGenerate[0])
	assert.Len(t, subreqs[0].ProtoFile, 5, "all proto files except bar/quux")

	assert.Len(t, subreqs[1].FileToGenerate, 3)
	assert.Equal(t, "bar/quux", subreqs[1].FileToGenerate[0])
	assert.Len(t, subreqs[1].ProtoFile, 4, "all files except foo and bar")

	assert.Len(t, subreqs[2].FileToGenerate, 1)
	assert.Equal(t, "bar/fizz/buzz", subreqs[2].FileToGenerate[0])
	assert.Len(t, subreqs[2].ProtoFile, 1, "only the file to gen")

	set, err := ParseParameters(subreqs[0].GetParameter()).BoolDefault(multiPackageSubProcessParam, false)
	assert.NoError(t, err)
	assert.True(t, set)
}

func TestMultiPackageWorkflow(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	assert.NotPanics(t, func() {
		wf.fanoutSubReqs(nil)
	})
}

func TestMultiPackageWorkflow_PrepareProcesses(t *testing.T) {
	t.Parallel()

	wf := &multiPackageWorkflow{}
	procs := wf.prepareProcesses(context.Background(), 3)

	assert.Len(t, procs, 3)
	for _, p := range procs {
		assert.NotNil(t, p)
	}
}

func TestMultiPackageWorkflow_HandleProcesses(t *testing.T) {

	g := Init()
	d := newMockDebugger(t)
	g.Debugger = d
	wf := &multiPackageWorkflow{Generator: g}

	reqs := []*protoc.CodeGeneratorRequest{
		{FileToGenerate: []string{"alpha"}},
		{FileToGenerate: []string{"beta"}},
	}

	res0, _ := proto.Marshal(&protoc.CodeGeneratorResponse{File: []*protoc.CodeGeneratorResponse_File{
		{Name: proto.String("foo"), Content: proto.String("bar")},
	}})

	res1, _ := proto.Marshal(&protoc.CodeGeneratorResponse{File: []*protoc.CodeGeneratorResponse_File{
		{Name: proto.String("fizz"), Content: proto.String("buzz")},
	}})

	procs := []subProcess{
		&mockSubProcess{out: bytes.NewReader(res0)},
		&mockSubProcess{out: bytes.NewReader(res1)},
	}

	res := wf.handleProcesses(&errgroup.Group{}, procs, reqs)

	if !assert.Nil(t, d.err) {
		return
	}

	assert.Len(t, res.File, 2)

	assert.Equal(t, "bar", res.File[0].GetContent())
	assert.Equal(t, "buzz", res.File[1].GetContent())
}

func TestMultiPackageWorkflow_HandleProcess_Success(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	res := &protoc.CodeGeneratorResponse{Error: proto.String("bar")}
	b, _ := proto.Marshal(res)

	sp := &mockSubProcess{
		out: bytes.NewReader(b),
		err: bytes.NewBufferString("some line\n"),
	}

	out := new(protoc.CodeGeneratorResponse)
	assert.NoError(t, wf.handleProcess(sp, req, out))
	assert.True(t, proto.Equal(res, out))

	b, _ = proto.Marshal(req)
	assert.Equal(t, b, sp.in.Bytes())
}

func TestMultiPackageWorkflow_HandleProcess_BrokenIn(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	sp := &mockSubProcess{inErr: errors.New("pipe error")}
	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	assert.Equal(t, sp.inErr, wf.handleProcess(sp, req, new(protoc.CodeGeneratorResponse)))
}

func TestMultiPackageWorkflow_HandleProcess_BrokenOut(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	sp := &mockSubProcess{outErr: errors.New("pipe error")}
	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	assert.Equal(t, sp.outErr, wf.handleProcess(sp, req, new(protoc.CodeGeneratorResponse)))
}

func TestMultiPackageWorkflow_HandleProcess_BrokenErr(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	sp := &mockSubProcess{errErr: errors.New("pipe error")}
	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	assert.Equal(t, sp.errErr, wf.handleProcess(sp, req, new(protoc.CodeGeneratorResponse)))
}

func TestMultiPackageWorkflow_HandleProcess_StartErr(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	sp := &mockSubProcess{startErr: errors.New("start error")}
	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	assert.Equal(t, sp.startErr, wf.handleProcess(sp, req, new(protoc.CodeGeneratorResponse)))
}

func TestMultiPackageWorkflow_HandleProcess_WaitErr(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	sp := &mockSubProcess{waitErr: errors.New("wait error")}
	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}
	assert.Equal(t, sp.waitErr, wf.handleProcess(sp, req, new(protoc.CodeGeneratorResponse)))
}

func TestMultiPackageWorkflow_HandleProcess_UnmarshalErr(t *testing.T) {
	t.Parallel()

	g := Init()
	g.Debugger = newMockDebugger(t)
	wf := &multiPackageWorkflow{Generator: g}

	sp := &mockSubProcess{out: bytes.NewReader([]byte("not a valid proto"))}
	req := &protoc.CodeGeneratorRequest{FileToGenerate: []string{"foo"}}

	assert.Error(t, wf.handleProcess(sp, req, new(protoc.CodeGeneratorResponse)))
}

type mockSubProcess struct {
	startErr, waitErr     error
	inErr, outErr, errErr error
	in                    bytes.Buffer
	out, err              io.Reader
}

func (sp *mockSubProcess) Start() error { return sp.startErr }
func (sp *mockSubProcess) Wait() error  { return sp.waitErr }

func (sp *mockSubProcess) StdinPipe() (io.WriteCloser, error) {
	return NopWriteCloser{&sp.in}, sp.inErr
}

func (sp *mockSubProcess) StdoutPipe() (io.ReadCloser, error) {
	if sp.out == nil {
		sp.out = bytes.NewReader([]byte{})
	}
	return ioutil.NopCloser(sp.out), sp.outErr
}

func (sp *mockSubProcess) StderrPipe() (io.ReadCloser, error) {
	if sp.err == nil {
		sp.err = bytes.NewReader([]byte{})
	}
	return ioutil.NopCloser(sp.err), sp.errErr
}

type NopWriteCloser struct{ io.Writer }

func (w NopWriteCloser) Close() error { return nil }
