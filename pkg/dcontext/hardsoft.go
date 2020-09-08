// Package dcontext provides tools for dealing with separate hard/soft
// cancellation of Contexts.
package dcontext

import (
	"context"
	"time"
)

type parentHardContextKey struct{}

// WithSoftness returns a copy of the parent "hard" Context with a way
// of getting the parent's Done channel.  This allows the child to
// have an earlier cancellation, triggering a "soft" shutdown, while
// allowing hard/soft-aware functions to use HardContext() to get the
// parent's Done channel, for a "hard" shutdown.
func WithSoftness(hardCtx context.Context) (softCtx context.Context) {
	return context.WithValue(hardCtx, parentHardContextKey{}, hardCtx)
}

type childHardContext struct {
	hardCtx context.Context
	softCtx context.Context
}

func (c childHardContext) Deadline() (deadline time.Time, ok bool) { return c.hardCtx.Deadline() }
func (c childHardContext) Done() <-chan struct{}                   { return c.hardCtx.Done() }
func (c childHardContext) Err() error                              { return c.hardCtx.Err() }
func (c childHardContext) Value(key interface{}) interface{}       { return c.softCtx.Value(key) }

// HardContext takes a child Context that is canceled sooner (a
// "soft" cancellation) and returns a Context with the same values, but
// with the cancellation of a parent Context that is canceled later
// (a "hard" cancellation).
//
// Such a "soft" cancellation Context is created by
// WithSoftness(hardCtx).  If the passed-in Context dones't have
// softness (WithSoftness isn't somewhere in its ancestry), then it is
// returned unmodified, because it is already hard.
func HardContext(softCtx context.Context) context.Context {
	parentHardCtx := softCtx.Value(parentHardContextKey{})
	if parentHardCtx == nil {
		return softCtx
	}
	return childHardContext{
		hardCtx: parentHardCtx.(context.Context),
		softCtx: softCtx,
	}
}
