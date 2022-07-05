package agent_test

import (
	"testing"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/agent"
	agentTypes "github.com/emissary-ingress/emissary/v3/pkg/api/agent"
	agentapi "github.com/emissary-ingress/emissary/v3/pkg/api/agent"
)

func TestHandleDirective(t *testing.T) {
	ctx := dlog.NewTestContext(t, false)

	a := &agent.Agent{}
	dh := &agent.BasicDirectiveHandler{}

	d := &agentTypes.Directive{ID: "one"}

	dh.HandleDirective(ctx, a, d)
}

func TestHandleSecretSyncDirective(t *testing.T) {
	// given
	ctx := dlog.NewTestContext(t, false)

	a := &agent.Agent{}
	dh := &agent.BasicDirectiveHandler{}

	d := &agentTypes.Directive{
		ID: "one",
		Commands: []*agentapi.Command{
			{
				SecretSyncCommand: &agentapi.SecretSyncCommand{
					Name:      "my-secret",
					Namespace: "my-namespace",
					CommandId: "1234",
					Action:    agentapi.SecretSyncCommand_SET,
					Secret: map[string][]byte{
						"my-key": []byte("abcd"),
					},
				},
			},
		},
	}

	// when
	dh.HandleDirective(ctx, a, d)

	// then
}
