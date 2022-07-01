package agent_test

import (
	"testing"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/agent"
	agentTypes "github.com/emissary-ingress/emissary/v3/pkg/api/agent"
)

func TestHandleDirective(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	a := &agent.Agent{}
	dh := &agent.BasicDirectiveHandler{}

	d := &agentTypes.Directive{ID: "one"}

	dh.HandleDirective(ctx, a, d)
}
