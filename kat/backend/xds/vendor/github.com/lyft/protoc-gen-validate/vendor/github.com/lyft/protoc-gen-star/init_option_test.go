package pgs

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"bytes"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIncludeGo(t *testing.T) {
	t.Parallel()

	g := &Generator{}
	assert.False(t, g.includeGo)

	IncludeGo()(g)
	assert.True(t, g.includeGo)
}

func TestDebugMode(t *testing.T) {
	t.Parallel()

	g := &Generator{}
	assert.False(t, g.debug)

	DebugMode()(g)
	assert.True(t, g.debug)
}

func TestDebugEnv(t *testing.T) {
	t.Parallel()

	g := &Generator{}
	assert.False(t, g.debug)

	e := strconv.Itoa(rand.Int())

	DebugEnv(e)(g)
	assert.False(t, g.debug)

	assert.NoError(t, os.Setenv(e, "1"))
	DebugEnv(e)(g)
	assert.True(t, g.debug)
}

func TestFileSystem(t *testing.T) {
	t.Parallel()

	p := dummyPersister(newMockDebugger(t))
	g := &Generator{persister: p}

	fs := afero.NewMemMapFs()
	FileSystem(fs)(g)

	assert.Equal(t, fs, p.fs)
}

func TestProtocInput(t *testing.T) {
	t.Parallel()

	g := &Generator{}
	assert.Nil(t, g.in)

	b := &bytes.Buffer{}
	ProtocInput(b)(g)
	assert.Equal(t, b, g.in)
}

func TestProtocOutput(t *testing.T) {
	t.Parallel()

	g := &Generator{}
	assert.Nil(t, g.out)

	b := &bytes.Buffer{}
	ProtocOutput(b)(g)
	assert.Equal(t, b, g.out)
}

func TestMultiPackage(t *testing.T) {
	t.Parallel()

	g := &Generator{workflow: &dummyWorkflow{}}

	MultiPackage()(g)
	_, ok := g.workflow.(*multiPackageWorkflow)
	assert.True(t, ok)
}

func TestRequirePlugin(t *testing.T) {
	t.Parallel()

	g := Init(RequirePlugin("foo", "bar"))

	p := Parameters{}
	for _, pm := range g.paramMutators {
		pm(p)
	}

	assert.Equal(t, "plugins=foo+bar", p.String())
}
