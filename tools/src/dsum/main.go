// Command dsum (short for d-summarize) helps keep good developer UX while running a command with
// potentially noisy output.
//
// Running `dsum SHORTNAME TIMEOUT CMD...` is functionally equivalent to invoking `CMD...` directly,
// however it provides an enhanced UX.
//
// The motivating command is `docker build`.  Building any given Dockerfile can range from really
// fast (<1s if it is cached) to really really slow (e.g. 15 minutes) if entirely uncached.  When
// the build is really fast, it can be tempting to use the `-q` (quiet) option for `docker build` in
// order to reduce noise.  The trouble is, if you change something early in the Dockerfile or you
// run the build on a new machine, that `-q` option suddenly appears as the build hanging entirely
// since it will happily run for the full uncached build time (e.g. 15 minutes) generating no
// output.  Running `dsum 'my build' 3s docker build ...` fixes this problem by behaving like
// `docker build -q` for any build that is faster than 3 seconds, but behaving like a normal docker
// build for any build that takes longer than 3 seconds.
package main

import (
	"container/list"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type BufferedPipe struct {
	sync   sync.Cond
	buff   list.List
	closed bool
}

func NewPipe() (io.Reader, io.WriteCloser) {
	ret := &BufferedPipe{}
	ret.sync.L = new(sync.Mutex)
	return ret, ret
}

func (b *BufferedPipe) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	val := append([]byte(nil), p...)
	b.sync.L.Lock()
	b.buff.PushBack(val)
	b.sync.Signal()
	b.sync.L.Unlock()
	return len(p), nil
}

func (b *BufferedPipe) Close() error {
	b.sync.L.Lock()
	b.closed = true
	b.sync.Signal()
	b.sync.L.Unlock()
	return nil
}

func (b *BufferedPipe) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.sync.L.Lock()
	for b.buff.Len() == 0 && !b.closed {
		b.sync.Wait()
	}
	if b.buff.Len() == 0 {
		b.sync.L.Unlock()
		return 0, io.EOF
	}
	el := b.buff.Front()
	avail := el.Value.([]byte)
	ret := avail
	if len(avail) > len(p) {
		ret = avail[:len(p)]
		el.Value = avail[len(p):]
	} else {
		b.buff.Remove(el)
	}
	b.sync.L.Unlock()
	copy(p, ret)
	return len(ret), nil
}

func errUsage(err error) {
	fmt.Fprintf(os.Stderr, "%[1]s: error: %[1]v\nUsage: %[1]s SHORTNAME TIMEOUT_DURATION CMD...\n", os.Args[0], err)
	os.Exit(2)
}

func main() {
	if len(os.Args) < 4 {
		errUsage(fmt.Errorf("not expected at least 3 arguments, got %d", len(os.Args)-1))
	}
	shortname := os.Args[1]
	timeout, err := time.ParseDuration(os.Args[2])
	if err != nil {
		errUsage(fmt.Errorf("could not parse timeout duration %q: %v", os.Args[2], err))
	}
	args := os.Args[3:]

	pipeR, pipeW := NewPipe()

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = pipeW
	cmd.Stderr = pipeW

	fmt.Fprintf(os.Stderr, "[dsum: %s] Running command: %q\n", shortname, args)
	fmt.Fprintf(os.Stderr, "[dsum: %s] But hiding output because it is likely to be verbose and boring...\n", shortname)

	waitCh := make(chan error)
	start := time.Now()
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[dsum: %s] could not run command: %v\n", shortname, err)
		os.Exit(1)
	}
	go func() {
		waitCh <- cmd.Wait()
		close(waitCh)
		pipeW.Close()
	}()

	var waitErr error
	var ioCh chan error
	startIO := func() {
		ioCh = make(chan error)
		go func() {
			_, err := io.Copy(os.Stderr, pipeR)
			ioCh <- err
			close(ioCh)
		}()
	}
	select {
	case waitErr = <-waitCh:
		if waitErr != nil {
			fmt.Fprintf(os.Stderr, "[dsum: %s] ... nevermind, the command errored, here's the output:\n", shortname)
			startIO()
		}
	case <-time.After(timeout):
		fmt.Fprintf(os.Stderr, "[dsum: %s] ... nevermind, the command is taking longer than expected (%v), here's the output:\n", shortname, timeout)
		startIO()
		waitErr = <-waitCh
	}

	if ioCh != nil {
		_ = <-ioCh
	}

	fmt.Fprintf(os.Stderr, "[dsum: %s] command finished in %s\n", shortname, time.Since(start))
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			if ee.Exited() {
				os.Exit(ee.ProcessState.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "[dsum: %s] command terminated abnormally: %v\n", shortname, waitErr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "[dsum: %s] I/O error: %v\n", shortname, waitErr)
		os.Exit(1)
	}
}
