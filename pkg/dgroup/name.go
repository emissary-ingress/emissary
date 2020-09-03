package dgroup

import (
	"context"

	"github.com/datawire/ambassador/pkg/dlog"
)

type goroutineNameKey struct{}

func getGoroutineName(ctx context.Context) string {
	name := ctx.Value(goroutineNameKey{})
	if name == nil {
		return ""
	}
	return name.(string)
}

// WithGoroutineName associates a name with the context, which gets
// logged by dlog as the "goroutine" field.
//
// If the context already has a name, then the new name is appended to
// it.  This allows a "tree" to be formed.
func WithGoroutineName(ctx context.Context, newName string) context.Context {
	oldName := getGoroutineName(ctx)
	if oldName != "" {
		newName = oldName + newName
	}
	ctx = dlog.WithField(ctx, "goroutine", newName)
	ctx = context.WithValue(ctx, goroutineNameKey{}, newName)
	return ctx
}
