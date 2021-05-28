package dgroup

import (
	"context"

	"github.com/datawire/dlib/dlog"
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
// logged by dlog as the "THREAD" field.
//
// If the context already has a name, then the new name is appended to
// it.  This allows a "tree" to be formed.  There are no delimiters
// added between names; you must include the delimiter as part of the
// name passed to WithGoroutineName.
//
// Group.Go calls this for you (using "/" as a delimiter); you
// shouldn't need to call WithGoroutineName for goroutines managed by
// a Group.
func WithGoroutineName(ctx context.Context, newName string) context.Context {
	oldName := getGoroutineName(ctx)
	if oldName != "" {
		newName = oldName + newName
	}
	ctx = dlog.WithField(ctx, "THREAD", newName)
	ctx = context.WithValue(ctx, goroutineNameKey{}, newName)
	return ctx
}
