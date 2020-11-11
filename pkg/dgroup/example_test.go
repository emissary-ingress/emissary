package dgroup_test

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

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
