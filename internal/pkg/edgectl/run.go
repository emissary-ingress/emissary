package edgectl

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/sirupsen/logrus"
)

var RunHelp = `edgectl run is a shorthand command for starting the daemon, connecting to the traffic
manager, adding an intercept, running a command, and then removing the intercept,
disconnecting, and quitting the daemon.

The command ensures that only those resources that were acquired are cleaned up. This
means that the daemon will not quit if it was already started, no disconnect will take
place if the connection was already established, and the intercept will not be removed
if it was already added.

Unless the daemon is already started, an attempt will be made to start it. This will
involve a call to sudo unless this command is run as root (not recommended).

Run a command:
    edgectl run hello -n example-url -t 9000 -- <command> arguments...
`

type RunInfo struct {
	InterceptInfo
	Self       string
	DNS        string
	Fallback   string
	Preview    bool
	PreviewSet bool
}

func (ri *RunInfo) RunCommand(cmd *cobra.Command, args []string) error {
	logrus.SetLevel(logrus.DebugLevel)

	ri.Self = os.Args[0]
	ri.PreviewSet = cmd.Flags().Changed("preview")
	return ri.withIntercept(func() error { return run(args[0], args[1:], true) })
}

// channelWriter writes everything it receives on a channel and provides a waitFor
// function that can wait for specific input using a timeout.
type channelWriter chan []byte

func (w channelWriter) Write(p []byte) (n int, err error) {
	w <- p
	return os.Stdout.Write(p)
}

// waitReturn constants enumerate the valid return values for the function passed to waitFor.
type waitReturn int

const (
	waitEndOk = waitReturn(iota)
	waitEndBad
	waitContinue
)

// waitFor reads the channel and passes the contents to the given function. If the given
// function returns waitEndOk or waitEndBad, then a go routine is started to drain the
// channel and true or false is returned immediately. Otherwise, waitFor continues to
// read the channel and call the function. If no waitEndOk or waitEndBad is returned
// until the given duration has expired, then this function returns false.
func (w channelWriter) waitFor(t time.Duration, f func(data []byte) waitReturn) bool {
	drain := func() {
		// ensure writer accepts remaining output without blocking
		for range w {
		}
	}

	timeout := time.NewTimer(t) // timeout waiting for ssh tunnel create
	for {
		select {
		case <-timeout.C:
			return false
		case bts, ok := <-w:
			if !ok {
				return false
			}
			switch f(bts) {
			case waitEndOk:
				timeout.Stop()
				go drain()
				return true
			case waitEndBad:
				go drain()
				return false
			}
		}
	}
}

// withIntercept runs the given function after asserting that an intercept is in place. The intercept
// is removed when the function ends if it was added.
func (ri *RunInfo) withIntercept(f func() error) error {
	return ri.withConnection(func() error {
		args := []string{ri.Self, "intercept", "add",
			ri.Deployment, "--name", ri.Name, "--target", ri.TargetHost}
		if ri.PreviewSet {
			args = append(args, "--preview", fmt.Sprintf("%t", ri.Preview))
		}
		for h, rx := range ri.Patterns {
			args = append(args, "--match", h+"="+rx)
		}
		if ri.Namespace != "" {
			args = append(args, "--namespace", ri.Namespace)
		}
		if ri.Prefix != "/" {
			args = append(args, "--prefix", ri.Prefix)
		}
		if ri.GRPC {
			args = append(args, "--grpc")
		}

		alreadyExists := false

		// create an io.Writer that writes a message on a channel when the desired message has been received
		ready := make(chan bool, 1)
		out := channelWriter(make(chan []byte, 5))
		go func() {
			ready <- out.waitFor(30*time.Second, func(bts []byte) waitReturn {
				line := string(bts)
				switch {
				case strings.Contains(line, "starting SSH"):
					return waitEndOk
				case strings.Contains(line, "already exists"):
					alreadyExists = true
					return waitEndOk
				}
				return waitContinue
			})
		}()

		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			logrus.Debug(strings.Join(args, " "))
		}
		err, exitCode := CommandViaDaemon(args, out)
		if err != nil {
			close(out) // terminates the above go routine
			return err
		}

		ok := <-ready
		if !ok {
			return fmt.Errorf("timeout waiting for intercept add")
		}
		if exitCode == 1 && alreadyExists {
			// The intercept was already added. This is not a bad thing in context of the run command
			exitCode = 0
		}
		if exitCode != 0 {
			return fmt.Errorf("%s intercept add exited with %d", ri.Self, exitCode)
		}
		if !alreadyExists {
			// ensure that added intercept is removed
			defer func() {
				logrus.Debugf("Removing intercept %s", ri.Name)
				_, _ = CommandViaDaemon([]string{ri.Self, "intercept", "remove", ri.Name}, os.Stdout)
			}()
		}
		return f()
	})
}

// withConnection runs the given function after asserting that a connection is active. A disconnect
// will take place when the function ends unless a connection was already established.
func (ri *RunInfo) withConnection(f func() error) error {
	return ri.withDaemonRunning(func() error {
		logrus.Debug("Connecting to daemon")
		wasConnected := false
		connected := false
		var err error

		var exitCode int
		for i := 0; i < 10; i++ {
			ready := make(chan bool, 1)
			out := channelWriter(make(chan []byte, 5))
			go func() {
				ready <- out.waitFor(20*time.Second, func(bts []byte) waitReturn {
					line := string(bts)
					switch {
					case strings.HasPrefix(line, "Already connected"):
						wasConnected = true
						return waitEndOk
					case strings.HasPrefix(line, "Connected"):
						return waitEndOk
					case strings.HasPrefix(line, "Not ready"):
						return waitEndBad
					default:
						return waitContinue
					}
				})
			}()

			err, exitCode = CommandViaDaemon([]string{ri.Self, "connect"}, out)
			if err == nil && exitCode != 0 {
				err = fmt.Errorf("%s connect exited with %d", ri.Self, exitCode)
				break
			}
			if <-ready {
				connected = true
				break
			}
			logrus.Debug("Connection not ready. Retrying...")
			time.Sleep(2 * time.Second)
		}
		if !connected {
			return fmt.Errorf("timeout trying to connect")
		}
		if !wasConnected {
			defer func() {
				logrus.Debug("Disconnecting from daemon")
				_, _ = CommandViaDaemon([]string{ri.Self, "disconnect"}, os.Stdout)
			}()
		}
		// Allow time for traffic manager to start
		time.Sleep(3 * time.Second)
		return f()
	})
}

// withDaemonRunning runs the given function after asserting that the daemon is started. The
// daemon will quit when the function returns unless it was already started.
func (ri *RunInfo) withDaemonRunning(f func() error) error {
	if IsServerRunning() {
		return f()
	}

	daemonError := atomic.Value{}
	go func() {
		logrus.Debug("Starting daemon")
		if err := ri.startDaemon(); err != nil {
			daemonError.Store(err)
		}
	}()

	defer func() {
		logrus.Debug("Quitting daemon")
		if err := ri.quitDaemon(); err != nil {
			logrus.Error(err.Error())
		}
	}()
	for {
		time.Sleep(50 * time.Millisecond)
		if err, ok := daemonError.Load().(error); ok {
			return err
		}
		if IsServerRunning() {
			return f()
		}
	}
}

func (ri *RunInfo) startDaemon() error {
	/* #nosec */
	exe := ri.Self
	args := []string{"daemon"}
	if ri.DNS != "" {
		args = append(args, "--dns", ri.DNS)
	}
	if ri.Fallback != "" {
		args = append(args, "--fallback", ri.Fallback)
	}
	return runAsRoot(exe, args)
}

func (ri *RunInfo) quitDaemon() error {
	return run(ri.Self, []string{"quit"}, false)
}

func runAsRoot(exe string, args []string) error {
	if os.Geteuid() != 0 {
		args = append([]string{"-E", exe}, args...)
		exe = "sudo"
	}
	return run(exe, args, false)
}

func run(exe string, args []string, trapSignals bool) error {
	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logrus.Debugf("executing %s %s\n", exe, strings.Join(args, " "))
	var err error
	if err = cmd.Start(); err != nil {
		logrus.Debugf("starting %s %s returned error: %s\n", exe, strings.Join(args, " "), err)
		return fmt.Errorf("%s %s: %v\n", exe, strings.Join(args, " "), err)
	}
	proc := cmd.Process
	var s *os.ProcessState
	if trapSignals {
		// Ensure that SIGINT and SIGTERM are propagated to the child process
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			sig := <-sigCh
			if sig == syscall.SIGUSR1 {
				return
			}
			_ = proc.Signal(sig)
		}()
		s, err = proc.Wait()
		sigCh <- syscall.SIGUSR1
	} else {
		s, err = proc.Wait()
	}
	if err != nil {
		logrus.Debugf("running %s %s returned error: %s\n", exe, strings.Join(args, " "), err)
		return fmt.Errorf("%s %s: %v\n", exe, strings.Join(args, " "), err)
	}
	exitCode := s.ExitCode()
	if exitCode != 0 {
		logrus.Debugf("executing %s %s returned exit code: %d\n", exe, strings.Join(args, " "), exitCode)
		return fmt.Errorf("%s %s: exited with %d\n", exe, strings.Join(args, " "), exitCode)
	}
	return nil
}
