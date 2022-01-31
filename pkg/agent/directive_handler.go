package agent

import (
	"context"
	"time"

	agentapi "github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/dlib/dlog"
)

type DirectiveHandler interface {
	HandleDirective(context.Context, *Agent, *agentapi.Directive)
}

type BasicDirectiveHandler struct {
	DefaultMinReportPeriod time.Duration
	rolloutsGetterFactory  rolloutsGetterFactory
}

func (dh *BasicDirectiveHandler) HandleDirective(ctx context.Context, a *Agent, directive *agentapi.Directive) {
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

		if command.RolloutCommand != nil {
			dh.handleRolloutCommand(ctx, command.RolloutCommand, a)
		}
	}

	a.SetLastDirectiveID(ctx, directive.ID)
}

func (dh *BasicDirectiveHandler) handleRolloutCommand(ctx context.Context, cmdSchema *agentapi.RolloutCommand, a *Agent) {
	if dh.rolloutsGetterFactory == nil {
		dlog.Warn(ctx, "Received rollout command but does not know how to talk to Argo Rollouts.")
		return
	}

	rolloutName := cmdSchema.GetName()
	namespace := cmdSchema.GetNamespace()
	action := int32(cmdSchema.GetAction())
	commandID := cmdSchema.GetCommandId()

	if rolloutName == "" {
		dlog.Warn(ctx, "Rollout command received without a rollout name.")
		return
	}

	if namespace == "" {
		dlog.Warn(ctx, "Rollout command received without a namespace.")
		return
	}

	if commandID == "" {
		dlog.Warn(ctx, "Rollout command received without a command ID.")
		return
	}

	cmd := &rolloutCommand{
		rolloutName: rolloutName,
		namespace:   namespace,
		action:      rolloutAction(agentapi.RolloutCommand_Action_name[action]),
	}
	err := cmd.RunWithClientFactory(ctx, dh.rolloutsGetterFactory)
	if err != nil {
		dlog.Errorf(ctx, "error running rollout command %s: %s", cmd, err)
	}
	dh.reportCommandResult(ctx, commandID, cmd, err, a)
}

func (dh *BasicDirectiveHandler) reportCommandResult(ctx context.Context, commandID string, cmd *rolloutCommand, cmdError error, a *Agent) {
	result := &agentapi.CommandResult{CommandId: commandID, Success: true}
	if cmdError != nil {
		result.Success = false
		result.Message = cmdError.Error()
	}
	err := a.comm.ReportCommandResult(ctx, result)
	if err != nil {
		dlog.Errorf(ctx, "error reporting result of rollout command %s: %s", cmd, cmdError)
	}
}
