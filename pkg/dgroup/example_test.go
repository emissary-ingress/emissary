package dgroup_test

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/datawire/ambassador/pkg/dcontext"
	"github.com/datawire/ambassador/pkg/dgroup"
	"github.com/datawire/ambassador/pkg/dlog"
)

func baseContext() context.Context {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	return dlog.WithLogger(context.Background(), dlog.WrapLogrus(logger))
}

func ExampleParentGroup() {
	ctx := dgroup.WithGoroutineName(baseContext(), "parent")
	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{})

	group.Go("a", func(ctx context.Context) error {
		dlog.Infoln(ctx, `I am goroutine "parent/a"`)

		// Use dgroup.ParentGroup to create a sibling goroutine
		dgroup.ParentGroup(ctx).Go("b", func(ctx context.Context) error {
			dlog.Infoln(ctx, `I am goroutine "parent/b"`)
			return nil
		})

		// Use dgroup.NewGroup to create a child goroutine.
		subgroup := dgroup.NewGroup(ctx, dgroup.GroupConfig{})
		subgroup.Go("c", func(ctx context.Context) error {
			dlog.Infoln(ctx, `I am goroutine "parent/a/c"`)
			return nil
		})

		// If you create a sub-group, be sure to wait
		return subgroup.Wait()
	})

	if err := group.Wait(); err != nil {
		dlog.Errorln(ctx, "exiting with error:", err)
	}

	// Unordered output:
	// level=info msg="I am goroutine \"parent/a\"" THREAD=parent/a
	// level=info msg="I am goroutine \"parent/b\"" THREAD=parent/b
	// level=info msg="I am goroutine \"parent/a/c\"" THREAD=parent/a/c
}

// This example shows the signal handler triggering a soft shutdown
// when the user hits Ctrl-C, and what the related logging looks like.
func Example_signalHandling1() {
	exEvents := make(chan struct{})
	go func() {
		// This goroutine isn't "part of" the example, but
		// simulates the user doing things, in order to drive
		// the example.
		self, _ := os.FindProcess(os.Getpid())

		<-exEvents // wait for things to get running

		// Simulate the user hitting Ctrl-C: This will log
		// that a signal was received, and trigger a
		// graceful-shutdown, logging that it's triggered
		// because of a signal.
		self.Signal(os.Interrupt)

		// The worker goroutine will then respond to the
		// graceful-shutdown, and Wait() will then log each of
		// the goroutines' "final" statuses and return.
	}()

	ctx := baseContext()
	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
	})

	group.Go("worker", func(ctx context.Context) error {
		// start up
		exEvents <- struct{}{}

		// wait for shutdown to be triggered
		<-ctx.Done()

		// quit
		return nil
	})

	if err := group.Wait(); err != nil {
		dlog.Errorln(ctx, "exiting with error:", err)
	}

	close(exEvents)
	// Output:
	// level=error msg="goroutine \":signal_handler:0\" exited with error: received signal interrupt (triggering graceful shutdown)" THREAD=":signal_handler:0"
	// level=info msg="shutting down (gracefully)..." THREAD=":shutdown_logger"
	// level=info msg="  final goroutine statuses:" THREAD=":shutdown_status"
	// level=info msg="    /worker          : exited without error" THREAD=":shutdown_status"
	// level=info msg="    :signal_handler:0: exited with error" THREAD=":shutdown_status"
	// level=error msg="exiting with error: received signal interrupt (triggering graceful shutdown)"
}

// This example shows how the signal handler behaves when a worker is
// poorly behaved, and doesn't quit during soft-shutdown when the user
// hits hits Ctrl-C, but does handle hard-shutdown.
func Example_signalHandling2() {
	exEvents := make(chan struct{})
	go func() {
		// This goroutine isn't "part of" the example, but
		// simulates the user doing things, in order to drive
		// the example.
		self, _ := os.FindProcess(os.Getpid())

		<-exEvents // wait for things to get running

		// Simulate the user hitting Ctrl-C: This will log
		// that a signal was received, and trigger a
		// graceful-shutdown, logging that it's triggered
		// because of a signal.  However, the worker goroutine
		// will ignore it and keep running.
		self.Signal(os.Interrupt)

		// wait for the soft-shutdown to be triggered
		<-exEvents

		// Simulate the user hitting Ctrl-C again: This will
		// log that a signal was received, and trigger a
		// not-so-graceful-shutdown, logging that it's
		// triggered because of a second signal; and, because
		// the user being impatient might be a first sign that
		// shutdown might be going wrong, it will log each of
		// the goroutines' statuses, so you can see which task
		// is hanging.
		self.Signal(os.Interrupt)

		// The worker goroutine will then respond to the
		// not-so-graceful-shutdown, and Wait() will then log
		// each of the goroutines' "final" statuses and
		// return.
	}()

	ctx := baseContext()
	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
	})

	group.Go("worker", func(ctx context.Context) error {
		// start up
		exEvents <- struct{}{}

		// wait for soft-shutdown to be triggered
		<-ctx.Done()

		// respond to hard-shutdown: quit, but don't do a good job of it
		exEvents <- struct{}{}

		// wait for hard-shutdown to be triggered
		<-dcontext.HardContext(ctx).Done()

		// respond to hard-shutdown: quit
		time.Sleep(1 * time.Second)
		return nil
	})

	if err := group.Wait(); err != nil {
		dlog.Errorln(ctx, "exiting with error:", err)
	}

	close(exEvents)
	// Output:
	// level=error msg="goroutine \":signal_handler:0\" exited with error: received signal interrupt (triggering graceful shutdown)" THREAD=":signal_handler:0"
	// level=info msg="shutting down (gracefully)..." THREAD=":shutdown_logger"
	// level=error msg="received signal interrupt (graceful shutdown already triggered; triggering not-so-graceful shutdown)" THREAD=":signal_handler:1"
	// level=error msg="  goroutine statuses:" THREAD=":signal_handler:1"
	// level=error msg="    /worker          : running" THREAD=":signal_handler:1"
	// level=error msg="    :signal_handler:0: exited with error" THREAD=":signal_handler:1"
	// level=info msg="shutting down (not-so-gracefully)..." THREAD=":shutdown_logger"
	// level=info msg="  final goroutine statuses:" THREAD=":shutdown_status"
	// level=info msg="    /worker          : exited without error" THREAD=":shutdown_status"
	// level=info msg="    :signal_handler:0: exited with error" THREAD=":shutdown_status"
	// level=error msg="exiting with error: received signal interrupt (triggering graceful shutdown)"
}

const exampleStackTrace = `
goroutine 1405 [running]:
runtime/pprof.writeGoroutineStacks(0x6575e0, 0xc0003803c0, 0x30, 0x7f56be200788)
	/usr/lib/go/src/runtime/pprof/pprof.go:693 +0x9f
runtime/pprof.writeGoroutine(0x6575e0, 0xc0003803c0, 0x2, 0x203000, 0xc)
	/usr/lib/go/src/runtime/pprof/pprof.go:682 +0x45
runtime/pprof.(*Profile).WriteTo(0x7770c0, 0x6575e0, 0xc0003803c0, 0x2, 0xc000380340, 0xc00038a4d0)
	/usr/lib/go/src/runtime/pprof/pprof.go:331 +0x3f2
github.com/datawire/ambassador/pkg/dgroup.logGoroutineTraces(0x659e80, 0xc000392300, 0x61b619, 0x16, 0x629ca0)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:109 +0xba
github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func4(0x659e80, 0xc00017c330, 0xc0001586b0, 0xc0000357c0)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:305 +0x4e5
github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c330, 0xc0000b7920)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0
created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88

goroutine 1 [select]:
github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait(0xc000153e30, 0x616a5c, 0x6)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:418 +0x139
github.com/datawire/ambassador/pkg/dgroup_test.Example_signalHandling3()
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/example_test.go:233 +0x125
testing.runExample(0x61ba2a, 0x17, 0x629c90, 0x6245c4, 0x491, 0x0, 0x0)
	/usr/lib/go/src/testing/run_example.go:62 +0x209
testing.runExamples(0xc0001fbe98, 0x77b360, 0x5, 0x5, 0xbfe34159434d384f)
	/usr/lib/go/src/testing/example.go:44 +0x1af
testing.(*M).Run(0xc0000fc000, 0x0)
	/usr/lib/go/src/testing/testing.go:1346 +0x273
main.main()
	_testmain.go:105 +0x1c5

goroutine 1400 [IO wait]:
internal/poll.runtime_pollWait(0x7f56be1fb738, 0x72, 0x657920)
	/usr/lib/go/src/runtime/netpoll.go:220 +0x55
internal/poll.(*pollDesc).wait(0xc00012fa58, 0x72, 0x657901, 0x749470, 0x0)
	/usr/lib/go/src/internal/poll/fd_poll_runtime.go:87 +0x45
internal/poll.(*pollDesc).waitRead(...)
	/usr/lib/go/src/internal/poll/fd_poll_runtime.go:92
internal/poll.(*FD).Read(0xc00012fa40, 0xc0003b6000, 0x8000, 0x8000, 0x0, 0x0, 0x0)
	/usr/lib/go/src/internal/poll/fd_unix.go:159 +0x1a5
os.(*File).read(...)
	/usr/lib/go/src/os/file_posix.go:31
os.(*File).Read(0xc0000a8028, 0xc0003b6000, 0x8000, 0x8000, 0x56, 0x0, 0x0)
	/usr/lib/go/src/os/file.go:116 +0x71
io.copyBuffer(0x6575e0, 0xc000380140, 0x657500, 0xc0000a8028, 0xc0003b6000, 0x8000, 0x8000, 0x491, 0x0, 0x0)
	/usr/lib/go/src/io/io.go:409 +0x12c
io.Copy(...)
	/usr/lib/go/src/io/io.go:368
testing.runExample.func1(0xc0000a8028, 0xc00015e660)
	/usr/lib/go/src/testing/run_example.go:37 +0x85
created by testing.runExample
	/usr/lib/go/src/testing/run_example.go:35 +0x176

goroutine 1404 [chan receive]:
github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func3(0x659e80, 0xc00017c270, 0xc0000a5d70, 0xd)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:271 +0x4a
github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c270, 0xc0000b7900)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0
created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88

goroutine 1403 [select]:
github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func2(0x659e80, 0xc00017c1b0, 0xc0000a5d80, 0x9)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:255 +0x254
github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c1b0, 0xc000158870)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0
created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88

goroutine 1443 [syscall]:
os/signal.signal_recv(0x6585e0)
	/usr/lib/go/src/runtime/sigqueue.go:147 +0x9d
os/signal.loop()
	/usr/lib/go/src/os/signal/signal_unix.go:23 +0x25
created by os/signal.Notify.func1.1
	/usr/lib/go/src/os/signal/signal.go:150 +0x45

goroutine 1406 [select (no cases)]:
github.com/datawire/ambassador/pkg/dgroup_test.Example_signalHandling3.func2(0x659e80, 0xc00017c3f0, 0x0, 0x0)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/example_test.go:230 +0x2ab
github.com/datawire/ambassador/pkg/dgroup.(*Group).goWorkerCtx.func1(0x0, 0x0)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:354 +0x9f
github.com/datawire/ambassador/pkg/derrgroup.(*Group).Go.func2(0xc00017c420, 0xc00012fbc0, 0xc0000a5de5, 0x7)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:131 +0x2b
created by github.com/datawire/ambassador/pkg/derrgroup.(*Group).Go
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:129 +0x12d

goroutine 1407 [semacquire]:
sync.runtime_Semacquire(0xc00012fbcc)
	/usr/lib/go/src/runtime/sema.go:56 +0x45
sync.(*WaitGroup).Wait(0xc00012fbcc)
	/usr/lib/go/src/sync/waitgroup.go:130 +0x65
github.com/datawire/ambassador/pkg/derrgroup.(*Group).Wait(...)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:99
github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait.func1(0xc00015e840, 0xc000153e30)
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:413 +0x48
created by github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait
	/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:412 +0x85
`

// This example shows how the signal handler behaves when a worker is
// poorly behaved, and doesn't quit at all.
func Example_signalHandling3() {
	exEvents := make(chan struct{})
	exFinished := make(chan struct{})
	go func() {
		// This goroutine isn't "part of" the example, but
		// simulates the user doing things, in order to drive
		// the example.
		self, _ := os.FindProcess(os.Getpid())

		<-exEvents // wait for things to get running

		// Simulate the user hitting Ctrl-C: This will log
		// that a signal was received, and trigger a
		// graceful-shutdown, logging that it's triggered
		// because of a signal.  However, the worker goroutine
		// will ignore it and keep running.
		self.Signal(os.Interrupt)

		// wait for the soft-shutdown to be triggered
		<-exEvents

		// Simulate the user hitting Ctrl-C a 2nd time: This
		// will log that a signal was received, and trigger a
		// not-so-graceful-shutdown, logging that it's
		// triggered because of a second signal; and, because
		// the user being impatient might be a first sign that
		// shutdown might be going wrong, it will log each of
		// the goroutines' statuses, so you can see which task
		// is hanging.  However, the worker goroutine will
		// ignore it and keep running.
		self.Signal(os.Interrupt)

		// wait for the hard-shutdown to be triggered
		<-exEvents

		// Simulate the user hitting Ctrl-C a 3rd time: This
		// will log that a 3rd signal was received, and
		// because the user having to hit Ctrl-C this many
		// times before something happens indicates that
		// something is probably wrong, it will not only log
		// each of the goroutines' statuses so you can see
		// which task is hanging, but it will also log a
		// stacktraces of each goroutine.  However, the worker
		// goroutine will ignore it and keep running.
		self.Signal(os.Interrupt)

		// Because the worker goroutine is hanging, and not
		// responding to our shutdown signals, we've set a
		// HardShutdownTimeout that will let Wait() return
		// even though some goroutines are still running.
		// Because something is definitely wrong, once again
		// we log each of the goroutine's statuses (at
		// loglevel info), and stacktraces for each goroutine
		// (at loglevel error).
		close(exFinished)
	}()
	dgroup.SetStacktraceForTesting(exampleStackTrace)

	ctx := baseContext()
	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		EnableSignalHandling: true,
		HardShutdownTimeout:  1 * time.Second,
	})

	group.Go("worker", func(ctx context.Context) error {
		// start up
		exEvents <- struct{}{}

		// wait for soft-shutdown to be triggered
		<-ctx.Done()

		// respond to soft-shutdown: hang
		exEvents <- struct{}{}

		// wait for hard-shutdown to be triggered
		<-dcontext.HardContext(ctx).Done()

		// respond to hard-shutdown: hang
		exEvents <- struct{}{}
		select {}
	})

	if err := group.Wait(); err != nil {
		dlog.Errorln(ctx, "exiting with error:", err)
	}

	<-exFinished
	close(exEvents)
	// Unordered output:
	// level=error msg="goroutine \":signal_handler:0\" exited with error: received signal interrupt (triggering graceful shutdown)" THREAD=":signal_handler:0"
	// level=info msg="shutting down (gracefully)..." THREAD=":shutdown_logger"
	// level=error msg="received signal interrupt (graceful shutdown already triggered; triggering not-so-graceful shutdown)" THREAD=":signal_handler:1"
	// level=error msg="  goroutine statuses:" THREAD=":signal_handler:1"
	// level=error msg="    /worker          : running" THREAD=":signal_handler:1"
	// level=error msg="    :signal_handler:0: exited with error" THREAD=":signal_handler:1"
	// level=info msg="shutting down (not-so-gracefully)..." THREAD=":shutdown_logger"
	// level=error msg="received signal interrupt (not-so-graceful shutdown already triggered)" THREAD=":signal_handler:2"
	// level=error msg="  goroutine statuses:" THREAD=":signal_handler:2"
	// level=error msg="    /worker          : running" THREAD=":signal_handler:2"
	// level=error msg="    :signal_handler:0: exited with error" THREAD=":signal_handler:2"
	// level=error msg="  goroutine stack traces:" THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1405 [running]:" THREAD=":signal_handler:2"
	// level=error msg="    runtime/pprof.writeGoroutineStacks(0x6575e0, 0xc0003803c0, 0x30, 0x7f56be200788)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/runtime/pprof/pprof.go:693 +0x9f" THREAD=":signal_handler:2"
	// level=error msg="    runtime/pprof.writeGoroutine(0x6575e0, 0xc0003803c0, 0x2, 0x203000, 0xc)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/runtime/pprof/pprof.go:682 +0x45" THREAD=":signal_handler:2"
	// level=error msg="    runtime/pprof.(*Profile).WriteTo(0x7770c0, 0x6575e0, 0xc0003803c0, 0x2, 0xc000380340, 0xc00038a4d0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/runtime/pprof/pprof.go:331 +0x3f2" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.logGoroutineTraces(0x659e80, 0xc000392300, 0x61b619, 0x16, 0x629ca0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:109 +0xba" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func4(0x659e80, 0xc00017c330, 0xc0001586b0, 0xc0000357c0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:305 +0x4e5" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c330, 0xc0000b7920)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0" THREAD=":signal_handler:2"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1 [select]:" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait(0xc000153e30, 0x616a5c, 0x6)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:418 +0x139" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup_test.Example_signalHandling3()" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/example_test.go:233 +0x125" THREAD=":signal_handler:2"
	// level=error msg="    testing.runExample(0x61ba2a, 0x17, 0x629c90, 0x6245c4, 0x491, 0x0, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/testing/run_example.go:62 +0x209" THREAD=":signal_handler:2"
	// level=error msg="    testing.runExamples(0xc0001fbe98, 0x77b360, 0x5, 0x5, 0xbfe34159434d384f)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/testing/example.go:44 +0x1af" THREAD=":signal_handler:2"
	// level=error msg="    testing.(*M).Run(0xc0000fc000, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/testing/testing.go:1346 +0x273" THREAD=":signal_handler:2"
	// level=error msg="    main.main()" THREAD=":signal_handler:2"
	// level=error msg="    \t_testmain.go:105 +0x1c5" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1400 [IO wait]:" THREAD=":signal_handler:2"
	// level=error msg="    internal/poll.runtime_pollWait(0x7f56be1fb738, 0x72, 0x657920)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/runtime/netpoll.go:220 +0x55" THREAD=":signal_handler:2"
	// level=error msg="    internal/poll.(*pollDesc).wait(0xc00012fa58, 0x72, 0x657901, 0x749470, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/internal/poll/fd_poll_runtime.go:87 +0x45" THREAD=":signal_handler:2"
	// level=error msg="    internal/poll.(*pollDesc).waitRead(...)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/internal/poll/fd_poll_runtime.go:92" THREAD=":signal_handler:2"
	// level=error msg="    internal/poll.(*FD).Read(0xc00012fa40, 0xc0003b6000, 0x8000, 0x8000, 0x0, 0x0, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/internal/poll/fd_unix.go:159 +0x1a5" THREAD=":signal_handler:2"
	// level=error msg="    os.(*File).read(...)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/os/file_posix.go:31" THREAD=":signal_handler:2"
	// level=error msg="    os.(*File).Read(0xc0000a8028, 0xc0003b6000, 0x8000, 0x8000, 0x56, 0x0, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/os/file.go:116 +0x71" THREAD=":signal_handler:2"
	// level=error msg="    io.copyBuffer(0x6575e0, 0xc000380140, 0x657500, 0xc0000a8028, 0xc0003b6000, 0x8000, 0x8000, 0x491, 0x0, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/io/io.go:409 +0x12c" THREAD=":signal_handler:2"
	// level=error msg="    io.Copy(...)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/io/io.go:368" THREAD=":signal_handler:2"
	// level=error msg="    testing.runExample.func1(0xc0000a8028, 0xc00015e660)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/testing/run_example.go:37 +0x85" THREAD=":signal_handler:2"
	// level=error msg="    created by testing.runExample" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/testing/run_example.go:35 +0x176" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1404 [chan receive]:" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func3(0x659e80, 0xc00017c270, 0xc0000a5d70, 0xd)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:271 +0x4a" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c270, 0xc0000b7900)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0" THREAD=":signal_handler:2"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1403 [select]:" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func2(0x659e80, 0xc00017c1b0, 0xc0000a5d80, 0x9)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:255 +0x254" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c1b0, 0xc000158870)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0" THREAD=":signal_handler:2"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1443 [syscall]:" THREAD=":signal_handler:2"
	// level=error msg="    os/signal.signal_recv(0x6585e0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/runtime/sigqueue.go:147 +0x9d" THREAD=":signal_handler:2"
	// level=error msg="    os/signal.loop()" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/os/signal/signal_unix.go:23 +0x25" THREAD=":signal_handler:2"
	// level=error msg="    created by os/signal.Notify.func1.1" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/os/signal/signal.go:150 +0x45" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1406 [select (no cases)]:" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup_test.Example_signalHandling3.func2(0x659e80, 0xc00017c3f0, 0x0, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/example_test.go:230 +0x2ab" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goWorkerCtx.func1(0x0, 0x0)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:354 +0x9f" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/derrgroup.(*Group).Go.func2(0xc00017c420, 0xc00012fbc0, 0xc0000a5de5, 0x7)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:131 +0x2b" THREAD=":signal_handler:2"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/derrgroup.(*Group).Go" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:129 +0x12d" THREAD=":signal_handler:2"
	// level=error msg="    " THREAD=":signal_handler:2"
	// level=error msg="    goroutine 1407 [semacquire]:" THREAD=":signal_handler:2"
	// level=error msg="    sync.runtime_Semacquire(0xc00012fbcc)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/runtime/sema.go:56 +0x45" THREAD=":signal_handler:2"
	// level=error msg="    sync.(*WaitGroup).Wait(0xc00012fbcc)" THREAD=":signal_handler:2"
	// level=error msg="    \t/usr/lib/go/src/sync/waitgroup.go:130 +0x65" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/derrgroup.(*Group).Wait(...)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:99" THREAD=":signal_handler:2"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait.func1(0xc00015e840, 0xc000153e30)" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:413 +0x48" THREAD=":signal_handler:2"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait" THREAD=":signal_handler:2"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:412 +0x85" THREAD=":signal_handler:2"
	// level=info msg="  final goroutine statuses:" THREAD=":shutdown_status"
	// level=info msg="    /worker          : running" THREAD=":shutdown_status"
	// level=info msg="    :signal_handler:0: exited with error" THREAD=":shutdown_status"
	// level=error msg="  final goroutine stack traces:" THREAD=":shutdown_status"
	// level=error msg="    goroutine 1405 [running]:" THREAD=":shutdown_status"
	// level=error msg="    runtime/pprof.writeGoroutineStacks(0x6575e0, 0xc0003803c0, 0x30, 0x7f56be200788)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/runtime/pprof/pprof.go:693 +0x9f" THREAD=":shutdown_status"
	// level=error msg="    runtime/pprof.writeGoroutine(0x6575e0, 0xc0003803c0, 0x2, 0x203000, 0xc)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/runtime/pprof/pprof.go:682 +0x45" THREAD=":shutdown_status"
	// level=error msg="    runtime/pprof.(*Profile).WriteTo(0x7770c0, 0x6575e0, 0xc0003803c0, 0x2, 0xc000380340, 0xc00038a4d0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/runtime/pprof/pprof.go:331 +0x3f2" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.logGoroutineTraces(0x659e80, 0xc000392300, 0x61b619, 0x16, 0x629ca0)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:109 +0xba" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func4(0x659e80, 0xc00017c330, 0xc0001586b0, 0xc0000357c0)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:305 +0x4e5" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c330, 0xc0000b7920)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0" THREAD=":shutdown_status"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1 [select]:" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait(0xc000153e30, 0x616a5c, 0x6)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:418 +0x139" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup_test.Example_signalHandling3()" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/example_test.go:233 +0x125" THREAD=":shutdown_status"
	// level=error msg="    testing.runExample(0x61ba2a, 0x17, 0x629c90, 0x6245c4, 0x491, 0x0, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/testing/run_example.go:62 +0x209" THREAD=":shutdown_status"
	// level=error msg="    testing.runExamples(0xc0001fbe98, 0x77b360, 0x5, 0x5, 0xbfe34159434d384f)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/testing/example.go:44 +0x1af" THREAD=":shutdown_status"
	// level=error msg="    testing.(*M).Run(0xc0000fc000, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/testing/testing.go:1346 +0x273" THREAD=":shutdown_status"
	// level=error msg="    main.main()" THREAD=":shutdown_status"
	// level=error msg="    \t_testmain.go:105 +0x1c5" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1400 [IO wait]:" THREAD=":shutdown_status"
	// level=error msg="    internal/poll.runtime_pollWait(0x7f56be1fb738, 0x72, 0x657920)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/runtime/netpoll.go:220 +0x55" THREAD=":shutdown_status"
	// level=error msg="    internal/poll.(*pollDesc).wait(0xc00012fa58, 0x72, 0x657901, 0x749470, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/internal/poll/fd_poll_runtime.go:87 +0x45" THREAD=":shutdown_status"
	// level=error msg="    internal/poll.(*pollDesc).waitRead(...)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/internal/poll/fd_poll_runtime.go:92" THREAD=":shutdown_status"
	// level=error msg="    internal/poll.(*FD).Read(0xc00012fa40, 0xc0003b6000, 0x8000, 0x8000, 0x0, 0x0, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/internal/poll/fd_unix.go:159 +0x1a5" THREAD=":shutdown_status"
	// level=error msg="    os.(*File).read(...)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/os/file_posix.go:31" THREAD=":shutdown_status"
	// level=error msg="    os.(*File).Read(0xc0000a8028, 0xc0003b6000, 0x8000, 0x8000, 0x56, 0x0, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/os/file.go:116 +0x71" THREAD=":shutdown_status"
	// level=error msg="    io.copyBuffer(0x6575e0, 0xc000380140, 0x657500, 0xc0000a8028, 0xc0003b6000, 0x8000, 0x8000, 0x491, 0x0, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/io/io.go:409 +0x12c" THREAD=":shutdown_status"
	// level=error msg="    io.Copy(...)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/io/io.go:368" THREAD=":shutdown_status"
	// level=error msg="    testing.runExample.func1(0xc0000a8028, 0xc00015e660)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/testing/run_example.go:37 +0x85" THREAD=":shutdown_status"
	// level=error msg="    created by testing.runExample" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/testing/run_example.go:35 +0x176" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1404 [chan receive]:" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func3(0x659e80, 0xc00017c270, 0xc0000a5d70, 0xd)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:271 +0x4a" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c270, 0xc0000b7900)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0" THREAD=":shutdown_status"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1403 [select]:" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).launchSupervisors.func2(0x659e80, 0xc00017c1b0, 0xc0000a5d80, 0x9)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:255 +0x254" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx.func1(0xc000153e30, 0x659e80, 0xc00017c1b0, 0xc000158870)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:396 +0xb0" THREAD=":shutdown_status"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).goSupervisorCtx" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:384 +0x88" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1443 [syscall]:" THREAD=":shutdown_status"
	// level=error msg="    os/signal.signal_recv(0x6585e0)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/runtime/sigqueue.go:147 +0x9d" THREAD=":shutdown_status"
	// level=error msg="    os/signal.loop()" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/os/signal/signal_unix.go:23 +0x25" THREAD=":shutdown_status"
	// level=error msg="    created by os/signal.Notify.func1.1" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/os/signal/signal.go:150 +0x45" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1406 [select (no cases)]:" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup_test.Example_signalHandling3.func2(0x659e80, 0xc00017c3f0, 0x0, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/example_test.go:230 +0x2ab" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).goWorkerCtx.func1(0x0, 0x0)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:354 +0x9f" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/derrgroup.(*Group).Go.func2(0xc00017c420, 0xc00012fbc0, 0xc0000a5de5, 0x7)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:131 +0x2b" THREAD=":shutdown_status"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/derrgroup.(*Group).Go" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:129 +0x12d" THREAD=":shutdown_status"
	// level=error msg="    " THREAD=":shutdown_status"
	// level=error msg="    goroutine 1407 [semacquire]:" THREAD=":shutdown_status"
	// level=error msg="    sync.runtime_Semacquire(0xc00012fbcc)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/runtime/sema.go:56 +0x45" THREAD=":shutdown_status"
	// level=error msg="    sync.(*WaitGroup).Wait(0xc00012fbcc)" THREAD=":shutdown_status"
	// level=error msg="    \t/usr/lib/go/src/sync/waitgroup.go:130 +0x65" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/derrgroup.(*Group).Wait(...)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/derrgroup/errgroup.go:99" THREAD=":shutdown_status"
	// level=error msg="    github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait.func1(0xc00015e840, 0xc000153e30)" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:413 +0x48" THREAD=":shutdown_status"
	// level=error msg="    created by github.com/datawire/ambassador/pkg/dgroup.(*Group).Wait" THREAD=":shutdown_status"
	// level=error msg="    \t/home/lukeshu/src/github.com/datawire/apro/ambassador/pkg/dgroup/group.go:412 +0x85" THREAD=":shutdown_status"
	// level=error msg="exiting with error: failed to shut down within the 1s shutdown timeout; some goroutines are left running"
}

func Example_timeout() {
	ctx := baseContext()

	group := dgroup.NewGroup(ctx, dgroup.GroupConfig{
		// Once one of the workers quits, the others should quit too.
		ShutdownOnNonError: true,
		// Give those others 1 second to quit after the first one quits.
		SoftShutdownTimeout: 1 * time.Second,
	})

	group.Go("a", func(ctx context.Context) error {
		dlog.Infoln(ctx, "I'm running")
		// now I'm done
		return nil
	})

	group.Go("b", func(ctx context.Context) error {
		dlog.Infoln(ctx, "I'm running")

		<-ctx.Done() // graceful shutdown
		dlog.Infoln(ctx, "I should shutdown now... but give me just a moment more")

		<-dcontext.HardContext(ctx).Done() // not-so-graceful shutdown
		dlog.Infoln(ctx, "oops, I've been a bad boy, I really do need to shut down now")
		return nil
	})

	if err := group.Wait(); err != nil {
		dlog.Errorln(ctx, "exiting with error:", err)
	}

	// Unordered output:
	// level=info msg="I'm running" THREAD=/a
	// level=info msg="I'm running" THREAD=/b
	// level=info msg="I should shutdown now... but give me just a moment more" THREAD=/b
	// level=info msg="shutting down (gracefully)..." THREAD=":shutdown_logger"
	// level=info msg="oops, I've been a bad boy, I really do need to shut down now" THREAD=/b
	// level=info msg="shutting down (not-so-gracefully)..." THREAD=":shutdown_logger"
}
