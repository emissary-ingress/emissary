// Package dexec is a logging variant of os/exec.
//
// dexec is *almost* a drop-in replacement for os/exec.  Differences
// are:
//
// - The "Command" function is missing, because a context is always
// required; use CommandContext.
//
// - It is not valid to create a "Cmd" entirely by hand; you must
// create it using CommandContext.  After it has been created, you may
// adjust the fields as you would with an os/exec.Cmd.
//
// The logger used is configured in the context.Context passed to
// CommandContext by calling
// github.com/datawire/dlib/dlog.WithLogger.
//
// A Cmd logs when it starts, its exit status, and everything read
// from or written to .Stdin, .Stdout, and .Stderr if they aren't an
// *os.File.  If one of those is an *os.File (as it is following a
// call to .StdinPipe, .StdoutPipe, or .StderrPipe), then that stream
// won't be logged (but it will print a message at process-start
// noting that it isn't being logged).
//
// For example:
//
//     ctx := dlog.WithLogger(context.Background(), myLogger)
//     cmd := dexec.CommandContext(ctx, "printf", "%s\n", "foo bar", "baz")
//     cmd.Stdin = os.Stdin
//     err := cmd.Run()
//
// will log the lines (assuming the default dlog configuration):
//
//     time="2021-05-18T17:18:35-06:00" level=info dexec.pid=24272 msg="started command [\"printf\" \"%s\\n\" \"foo bar\" \"baz\"]"
//     time="2021-05-18T17:18:35-06:00" level=info dexec.pid=24272 dexec.stream=stdin msg="not logging input read from file \"/dev/stdin\""
//     time="2021-05-18T17:18:35-06:00" level=info dexec.pid=24272 dexec.stream=stdout+stderr dexec.data="foo bar\n"
//     time="2021-05-18T17:18:35-06:00" level=info dexec.pid=24272 dexec.stream=stdout+stderr dexec.data="baz\n"
//     time="2021-05-18T17:18:35-06:00" level=info dexec.pid=24272 msg="finished successfully: exit status 0"
//
// If you would like a "pipe" to be logged, use an io.Pipe instead of
// calling .StdinPipe, .StdoutPipe, or .StderrPipe.
//
// See the os/exec documentation for more information.
package dexec

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dlog"
)

// Error is returned by LookPath when it fails to classify a file as an
// executable.
type Error = exec.Error

// An ExitError reports an unsuccessful exit by a command.
type ExitError = exec.ExitError

// ErrNotFound is the os/exec.ErrNotFound value.
var ErrNotFound = exec.ErrNotFound

// LookPath is the os/exe.LookPath function.
var LookPath = exec.LookPath

// Cmd represents an external command being prepared or run.
//
// A Cmd cannot be reused after calling its Run, Output or CombinedOutput
// methods.
//
// See the os/exec.Cmd documentation for information on the fields
// within it.
//
// Unlike an os/exec.Cmd, you MUST NOT construct a Cmd by hand, it
// must be created with CommandContext.
type Cmd struct {
	*exec.Cmd
	DisableLogging bool

	ctx context.Context

	pidlock sync.RWMutex

	waitDone chan struct{}
	waitOnce sync.Once
}

// CommandContext returns the Cmd struct to execute the named program with
// the given arguments.
//
// The provided context is used for two purposes:
//
//  1. To kill the process (by calling os.Process.Kill) if the context
//     becomes done before the command completes on its own.
//  2. For logging (see github.com/datawire/dlib/dlog).
//
// See the os/exec.Command and os/exec.CommandContext documentation
// for more information.
func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	ret := &Cmd{
		Cmd: exec.CommandContext(dcontext.HardContext(ctx), name, arg...),
		ctx: ctx,
	}
	ret.pidlock.Lock()
	return ret
}

func (c *Cmd) logiofn(stream string) func(error, []byte) {
	return func(err error, msg []byte) {
		if c.DisableLogging {
			return
		}

		c.pidlock.RLock()
		defer c.pidlock.RUnlock()
		pid := -1
		if c.Process != nil {
			pid = c.Process.Pid
		}
		ctx := dlog.WithField(c.ctx, "dexec.pid", pid)
		ctx = dlog.WithField(ctx, "dexec.stream", stream)
		if msg != nil {
			ctx = dlog.WithField(ctx, "dexec.data", string(msg))
		}
		if err != nil {
			ctx = dlog.WithField(ctx, "dexec.err", err)
		}
		// We don't have an additional message to log; all of the info that we want to log
		// is provided via dlog.WithField.
		dlog.Print(ctx)
	}
}

// Start starts the specified command but does not wait for it to complete.
//
// See the os/exec.Cmd.Start documenaton for more information.
func (c *Cmd) Start() error {
	c.Stdin = fixupReader(c.Stdin, c.logiofn("stdin"))
	if interfaceEqual(c.Stdout, c.Stderr) {
		c.Stdout = fixupWriter(c.Stdout, c.logiofn("stdout+stderr"))
		c.Stderr = c.Stdout
	} else {
		c.Stdout = fixupWriter(c.Stdout, c.logiofn("stdout"))
		c.Stderr = fixupWriter(c.Stderr, c.logiofn("stderr"))
	}

	err := c.Cmd.Start()
	if err == nil {
		if !c.DisableLogging {
			ctx := dlog.WithField(c.ctx, "dexec.pid", c.Process.Pid)
			dlog.Printf(ctx, "started command %q", c.Args)
			if stdin, isFile := c.Stdin.(*os.File); isFile {
				dlog.Printf(dlog.WithField(ctx, "dexec.stream", "stdin"), "not logging input read from file %q", stdin.Name())
			}
			if stdout, isFile := c.Stdout.(*os.File); isFile {
				dlog.Printf(dlog.WithField(ctx, "dexec.stream", "stdout"), "not logging output written to file %q", stdout.Name())
			}
			if stderr, isFile := c.Stderr.(*os.File); isFile {
				dlog.Printf(dlog.WithField(ctx, "dexec.stream", "stderr"), "not logging output written to file %q", stderr.Name())
			}
		}
		if c.ctx != dcontext.HardContext(c.ctx) {
			c.waitDone = make(chan struct{})
			go func() {
				select {
				case <-dcontext.HardContext(c.ctx).Done(): // hard shutdown
					// let os/exec send SIGKILL
				case <-c.ctx.Done(): // soft shutdown
					_ = c.Cmd.Process.Signal(os.Interrupt) // send SIGINT
				case <-c.waitDone:
					// it exited on its own
				}
			}()
		}
	}
	c.pidlock.Unlock()
	return err
}

// Wait waits for the command to exit and waits for any copying to
// stdin or copying from stdout or stderr to complete.
//
// See the os/exec.Cmd.Wait documenaton for more information.
func (c *Cmd) Wait() error {
	err := c.Cmd.Wait()

	if c.waitDone != nil {
		c.waitOnce.Do(func() { close(c.waitDone) })
	}

	pid := -1
	if c.Process != nil {
		pid = c.Process.Pid
	}

	if !c.DisableLogging {
		ctx := dlog.WithField(c.ctx, "dexec.pid", pid)
		if err == nil {
			dlog.Printf(ctx, "finished successfully: %v", c.ProcessState)
		} else {
			dlog.Printf(ctx, "finished with error: %v", err)
		}
	}

	return err
}

// StdinPipe returns a pipe that will be connected to the command's
// standard input when the command starts.
//
// This sets .Stdin to an *os.File, causing what you write to the pipe
// to not be logged.
//
// See the os/exec.Cmd.StdinPipe documenaton for more information.
func (c *Cmd) StdinPipe() (io.WriteCloser, error) { return c.Cmd.StdinPipe() }

// StdoutPipe returns a pipe that will be connected to the command's
// standard output when the command starts.
//
// This sets .Stdout to an *os.File, causing what you read from the
// pipe to not be logged.
//
// See the os/exec.Cmd.StdoutPipe documenaton for more information.
func (c *Cmd) StdoutPipe() (io.ReadCloser, error) { return c.Cmd.StdoutPipe() }

// StderrPipe returns a pipe that will be connected to the command's
// standard error when the command starts.
//
// This sets .Stderr to an *os.File, causing what you read from the
// pipe to not be logged.
//
// See the os/exec.Cmd.StderrPipe documenaton for more information.
func (c *Cmd) StderrPipe() (io.ReadCloser, error) { return c.Cmd.StderrPipe() }

// Higher-level methods around these implemented in borrowed_cmd.go
