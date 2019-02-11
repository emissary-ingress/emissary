package pgs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixContext_Log(t *testing.T) {
	t.Parallel()

	l := newMockLogger()
	c := initPrefixContext(nil, &rootDebugger{l: l}, "foo")

	c.Log("bar")
	assert.Equal(t, "[foo] bar\n", l.buf.String())
}

func TestPrefixContext_Logf(t *testing.T) {
	t.Parallel()

	l := newMockLogger()
	c := initPrefixContext(nil, &rootDebugger{l: l}, "foo")

	c.Logf("bar %s", "baz")
	assert.Equal(t, "[foo] bar baz\n", l.buf.String())
}

func TestPrefixContext_Debug(t *testing.T) {
	t.Parallel()

	l := newMockLogger()
	c := initPrefixContext(nil, &rootDebugger{l: l, logDebugs: true}, "foo")

	c.Debug("bar")
	assert.Equal(t, "[foo] bar\n", l.buf.String())
}

func TestPrefixContext_Debugf(t *testing.T) {
	t.Parallel()

	l := newMockLogger()
	c := initPrefixContext(nil, &rootDebugger{l: l, logDebugs: true}, "foo")

	c.Debugf("bar %s", "baz")
	assert.Equal(t, "[foo] bar baz\n", l.buf.String())
}

func TestPrefixContext_Fail(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	c := initPrefixContext(nil, d, "foo")

	c.Fail("bar")
	assert.True(t, d.failed)
}

func TestPrefixContext_Failf(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	c := initPrefixContext(nil, d, "foo")

	c.Failf("bar %s", "baz")
	assert.True(t, d.failed)
}

func TestPrefixContext_CheckErr(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	c := initPrefixContext(nil, d, "foo")

	c.CheckErr(nil)
	assert.False(t, d.failed)
	err := errors.New("bar")
	c.CheckErr(err)
	assert.True(t, d.failed)
	assert.Equal(t, d.err, err)
}

func TestPrefixContext_Assert(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	c := initPrefixContext(nil, d, "foo")

	c.Assert(true)
	assert.False(t, d.failed)
	c.Assert(false)
	assert.True(t, d.failed)
}

func TestPrefixContext_OutputPath(t *testing.T) {
	t.Parallel()

	d := Context(newMockDebugger(t), Parameters{}, "foo/bar")
	c := initPrefixContext(d, newMockDebugger(t), "")
	assert.Equal(t, c.OutputPath(), d.OutputPath())
}

func TestPrefixContext_PushPop(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo/bar")
	p := initPrefixContext(r, newMockDebugger(t), "baz")

	c := p.Push("fizz")
	assert.IsType(t, prefixContext{}, c)
	assert.IsType(t, rootContext{}, c.Pop().Pop())
}

func TestPrefixContext_PushPopDir(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo/bar")
	p := initPrefixContext(r, newMockDebugger(t), "fizz")
	c := p.PushDir("baz")

	assert.Equal(t, "foo/bar/baz", c.OutputPath())
	assert.Equal(t, "foo/bar", c.Push("buzz").PopDir().OutputPath())
}

func TestPrefixContext_Parameters(t *testing.T) {
	t.Parallel()

	p := Parameters{"foo": "bar"}
	r := Context(newMockDebugger(t), p, ".")
	c := r.Push("foo")

	assert.Equal(t, p, c.Parameters())
}

func TestDirContext_OutputPath(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo/bar")
	d := initDirContext(r, newMockDebugger(t), "baz")
	assert.Equal(t, "foo/bar/baz", d.OutputPath())
}

func TestDirContext_Push(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo/bar")
	d := initDirContext(r, newMockDebugger(t), "baz")
	c := d.Push("fizz")

	assert.Equal(t, d.OutputPath(), c.OutputPath())
	assert.IsType(t, prefixContext{}, c)
}

func TestDirContext_PushPopDir(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo")
	d := initDirContext(r, newMockDebugger(t), "bar")
	c := d.PushDir("baz")

	assert.Equal(t, "foo/bar/baz", c.OutputPath())
	c = c.PopDir()
	assert.Equal(t, "foo/bar", c.OutputPath())
	c = c.PopDir()
	assert.Equal(t, "foo", c.OutputPath())
}

func TestRootContext_OutputPath(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo")
	assert.Equal(t, "foo", r.OutputPath())
}

func TestRootContext_PushPop(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	r := Context(d, Parameters{}, "foo")

	c := r.Push("bar")
	assert.Equal(t, "foo", c.OutputPath())
	c = c.Pop()

	assert.False(t, d.failed)
	c.Pop()
	assert.True(t, d.failed)
}

func TestRootContext_PushPopDir(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo")
	c := r.PushDir("bar")
	assert.Equal(t, "foo/bar", c.OutputPath())

	c = c.PopDir()
	assert.Equal(t, "foo", c.OutputPath())

	c = c.PopDir()
	assert.Equal(t, "foo", c.OutputPath())
}

func TestRootContext_Parameters(t *testing.T) {
	t.Parallel()

	p := Parameters{"foo": "bar"}
	r := Context(newMockDebugger(t), p, "foo")
	assert.Equal(t, p, r.Parameters())
}

func TestRootContext_JoinPath(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo")
	assert.Equal(t, "foo/bar", r.JoinPath("bar"))
}

func TestDirContext_JoinPath(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo")
	c := r.PushDir("bar")

	assert.Equal(t, "foo/bar/baz", c.JoinPath("baz"))
}

func TestPrefixContext_JoinPath(t *testing.T) {
	t.Parallel()

	r := Context(newMockDebugger(t), Parameters{}, "foo")
	c := r.Push("baz")

	assert.Equal(t, "foo/bar", c.JoinPath("bar"))
}

func TestPrefixContext_Exit(t *testing.T) {
	t.Parallel()

	d := newMockDebugger(t)
	r := Context(d, Parameters{}, "")
	r.Exit(123)

	assert.True(t, d.exited)
	assert.Equal(t, 123, d.exitCode)
}
