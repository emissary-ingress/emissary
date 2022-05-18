package kates

import (
	"context"
	"os"

	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// IncoherentApply is like `kubectl apply`, and DOES NOT count as a "write" for the purposes of the
// Client's read/write coherence.
func (c *Client) IncoherentApply(ctx context.Context, stdio IOStreams, args ...string) error {
	factory := cmdutil.NewFactory(c.config)

	cmd := apply.NewCmdApply(os.Args[0], factory, stdio)
	cmd.SetIn(stdio.In)
	cmd.SetOut(stdio.Out)
	cmd.SetErr(stdio.ErrOut)

	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}
