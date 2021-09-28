package agent_test

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/datawire/ambassador/v2/pkg/agent"
	agentTypes "github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/dlib/dlog"
)

func getCtxLog() (context.Context, context.CancelFunc) {
	llog := logrus.New()
	llog.SetLevel(logrus.DebugLevel)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = dlog.WithLogger(ctx, dlog.WrapLogrus(llog))

	return ctx, cancel
}

func TestHandleDirective(t *testing.T) {
	ctx, _ := getCtxLog()

	a := &agent.Agent{}
	dh := &agent.BasicDirectiveHandler{}

	d := &agentTypes.Directive{ID: "one"}

	dh.HandleDirective(ctx, a, d)
}
