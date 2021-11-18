package agent_test

import (
	"testing"

	"github.com/datawire/ambassador/v2/pkg/agent"
	agentTypes "github.com/datawire/ambassador/v2/pkg/api/agent"
	"github.com/datawire/dlib/dlog"
)

func TestHandleDirective(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	a := &agent.Agent{}
	dh := &agent.BasicDirectiveHandler{}

	d := &agentTypes.Directive{ID: "one"}

	dh.HandleDirective(ctx, a, d)
}
