package agent

import (
	"context"
	"time"

	"github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/dlib/dlog"
)

type DirectiveHandler interface {
	HandleDirective(context.Context, *Agent, *agent.Directive)
}

type BasicDirectiveHandler struct {
	DefaultMinReportPeriod time.Duration
}

func (dh *BasicDirectiveHandler) HandleDirective(ctx context.Context, a *Agent, directive *agent.Directive) {
	if directive == nil {
		dlog.Warn(ctx, "Received empty directive, ignoring.")
		return
	}
	ctx = dlog.WithField(ctx, "directive", directive.ID)

	dlog.Debug(ctx, "Directive received")

	if directive.StopReporting {
		// The Director wants us to stop reporting
		a.StopReporting(ctx)
	}

	if directive.MinReportPeriod != nil {
		// The Director wants to adjust the minimum time we wait between reports
		protoDur := directive.MinReportPeriod
		// Note: This conversion ignores potential overflow. In practice this
		// shouldn't be a problem, as the server will be constructing this
		// durationpb.Duration from a valid time.Duration.
		dur := time.Duration(protoDur.Seconds)*time.Second + time.Duration(protoDur.Nanos)*time.Nanosecond
		dur = MaxDuration(dur, dh.DefaultMinReportPeriod) // respect configured minimum
		a.SetMinReportPeriod(ctx, dur)
	}

	for _, command := range directive.Commands {
		if command.Message != "" {
			dlog.Info(ctx, command.Message)
		}
	}

	a.SetLastDirectiveID(ctx, directive.ID)
}
