package daemon

import (
	"context"
	"net"
	"os"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/datawire/ambassador/internal/pkg/edgectl"
	"github.com/datawire/ambassador/pkg/api/edgectl/rpc"
	"github.com/datawire/ambassador/pkg/supervisor"
	"github.com/datawire/dlib/dhttp"
)

var Help = `The Edge Control Daemon is a long-lived background component that manages
connections and network state.

Launch the Edge Control Daemon:
    sudo edgectl daemon

Examine the Daemon's log output in
    ` + edgectl.Logfile + `
to troubleshoot problems.
`

// daemon represents the state of the Edge Control Daemon
type daemon struct {
	network    Resource
	cluster    *KCluster
	bridge     Resource
	trafficMgr *TrafficManager
	intercepts []*Intercept
	dns        string
	fallback   string
}

// RunAsDaemon is the main function when executing as the daemon
func RunAsDaemon(dns, fallback string) error {
	if os.Geteuid() != 0 {
		return errors.New("edgectl daemon must run as root")
	}

	d := &daemon{dns: dns, fallback: fallback}

	ctx := SetUpLogging(context.Background())
	sup := supervisor.WithContext(ctx)
	sup.Supervise(&supervisor.Worker{
		Name: "daemon",
		Work: d.runGRPCService,
	})
	sup.Supervise(&supervisor.Worker{
		Name:     "signal",
		Requires: []string{"daemon"},
		Work:     WaitForSignal,
	})
	sup.Supervise(&supervisor.Worker{
		Name:     "setup",
		Requires: []string{"daemon"},
		Work: func(p *supervisor.Process) error {
			if err := d.MakeNetOverride(p); err != nil {
				return err
			}
			p.Ready()
			return nil
		},
	})

	sup.Logger(ctx, "---")
	sup.Logger(ctx, "Edge Control daemon %s starting...", edgectl.DisplayVersion())
	sup.Logger(ctx, "PID is %d", os.Getpid())
	runErrors := sup.Run()

	sup.Logger(ctx, "")
	if len(runErrors) > 0 {
		sup.Logger(ctx, "daemon has exited with %d error(s):", len(runErrors))
		for _, err := range runErrors {
			sup.Logger(ctx, "- %v", err)
		}
	}
	sup.Logger(ctx, "Edge Control daemon %s is done.", edgectl.DisplayVersion())
	return errors.New("edgectl daemon has exited")
}

type grpcService struct {
	s *grpc.Server
	d *daemon
	p *supervisor.Process
}

func (s *grpcService) Version(_ context.Context, _ *rpc.Empty) (*rpc.VersionResponse, error) {
	return &rpc.VersionResponse{
		APIVersion: edgectl.ApiVersion,
		Version:    edgectl.Version,
	}, nil
}

func (s *grpcService) Status(_ context.Context, _ *rpc.Empty) (*rpc.StatusResponse, error) {
	return s.d.status(s.p), nil
}

func (s *grpcService) Connect(_ context.Context, cr *rpc.ConnectRequest) (*rpc.ConnectResponse, error) {
	return s.d.connect(s.p, cr), nil
}

func (s *grpcService) Disconnect(_ context.Context, _ *rpc.Empty) (*rpc.DisconnectResponse, error) {
	return s.d.disconnect(s.p), nil
}

func (s *grpcService) AddIntercept(_ context.Context, ir *rpc.InterceptRequest) (*rpc.InterceptResponse, error) {
	return s.d.addIntercept(s.p, ir), nil
}

func (s *grpcService) RemoveIntercept(_ context.Context, rr *rpc.RemoveInterceptRequest) (*rpc.InterceptResponse, error) {
	return s.d.removeIntercept(s.p, rr.Name), nil
}

func (s *grpcService) AvailableIntercepts(_ context.Context, _ *rpc.Empty) (*rpc.AvailableInterceptsResponse, error) {
	return s.d.availableIntercepts(s.p), nil
}

func (s *grpcService) ListIntercepts(_ context.Context, _ *rpc.Empty) (*rpc.ListInterceptsResponse, error) {
	return s.d.listIntercepts(s.p), nil
}

func (s *grpcService) Pause(ctx context.Context, empty *rpc.Empty) (*rpc.PauseResponse, error) {
	return s.d.pause(s.p), nil
}

func (s *grpcService) Resume(ctx context.Context, empty *rpc.Empty) (*rpc.ResumeResponse, error) {
	return s.d.resume(s.p), nil
}

func (s *grpcService) Quit(ctx context.Context, empty *rpc.Empty) (*rpc.Empty, error) {
	// GracefulStop() must be called in a separate go routine since it will await the
	// client disconnect. That doesn't happen until this function returns.
	go s.s.GracefulStop()
	s.p.Supervisor().Shutdown()
	return &rpc.Empty{}, nil
}

func (d *daemon) runGRPCService(daemonProc *supervisor.Process) error {
	// Listen on unix domain socket
	unixListener, err := net.Listen("unix", edgectl.DaemonSocketName)
	if err != nil {
		return errors.Wrap(err, "listen")
	}
	err = os.Chmod(edgectl.DaemonSocketName, 0777)
	if err != nil {
		return errors.Wrap(err, "chmod")
	}

	grpcHandler := grpc.NewServer()
	rpc.RegisterDaemonServer(grpcHandler, &grpcService{
		s: grpcHandler,
		d: d,
		p: daemonProc,
	})
	sc := &dhttp.ServerConfig{
		Handler: grpcHandler,
	}

	daemonProc.Ready()
	Notify(daemonProc, "Running")
	defer Notify(daemonProc, "Shutting down...")
	ctx, cancel := context.WithCancel(daemonProc.Context())
	return daemonProc.DoClean(
		func() error { return sc.Serve(ctx, unixListener) },
		func() error { cancel(); return nil })
}

func (d *daemon) pause(p *supervisor.Process) *rpc.PauseResponse {
	r := rpc.PauseResponse{}
	switch {
	case d.network == nil:
		r.Error = rpc.PauseResponse_AlreadyPaused
	case d.cluster != nil:
		r.Error = rpc.PauseResponse_ConnectedToCluster
	default:
		if err := d.network.Close(); err != nil {
			r.Error = rpc.PauseResponse_UnexpectedPauseError
			r.ErrorText = err.Error()
			p.Logf("pause: %v", err)
		}
		d.network = nil
	}
	return &r
}

func (d *daemon) resume(p *supervisor.Process) *rpc.ResumeResponse {
	r := rpc.ResumeResponse{}
	if d.network != nil {
		if d.network.IsOkay() {
			r.Error = rpc.ResumeResponse_NotPaused
		} else {
			r.Error = rpc.ResumeResponse_ReEstablishing
		}
	} else if err := d.MakeNetOverride(p); err != nil {
		r.Error = rpc.ResumeResponse_UnexpectedResumeError
		r.ErrorText = err.Error()
		p.Logf("resume: %v", err)
	}
	return &r
}
