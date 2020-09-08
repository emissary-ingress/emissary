// Package dcontext provides tools for dealing with separate hard/soft
// cancellation of Contexts.
//
// Given
//
//    softCtx := WithSoftness(hardCtx)
//
// then
//
//    // The soft Context being done signals the end of "normal
//    // operation", and the program should initiate a graceful
//    // shutdown; a "soft shutdown".  In other words, it means, "You
//    // should start shutting down now."
//    <-softCtx.Done()
//
//    // The hard Context being done signals that the time for a
//    // graceful shutdown has passed and that the programs should
//    // terminate *right now*; a "hard shutdown".  In other words, it
//    // means, "If you haven't finished shutting down yet, then you
//    // should hurry it up."
//    <-HardContext(softCtx).Done()
//
// It should almost always be the soft Context that is passed around
// (I recommend that you name it just "ctx", not "softCtx").
//
//
//
// Interfacing dcontext-aware code with dcontext-unaware code
//
// When dcontext-aware code passes the soft Context to
// dcontext-unaware code, then that callee code will shutdown at the
// beginning of the shutdown grace period.  This is correct, because
// the beginning of that grace period means "start shutting down"; if
// the callee code is dcontext-unaware, then its shutdown is probably
// more-or-less instantaneous, and so it is OK if it doesn't observe
// the hard-shutdown signal as it already shut down from the soft
// shutdown signal.
//
// When dcontext-unaware code passes a hard (normal) Context to
// dcontext-aware code, then that callee code will observe the
// <-ctx.Done() and <-HardContext(ctx).Done() occurring at the same
// instant.  This is correct, because the caller code doesn't allow
// any grace period between "start shutting down" and "you need to
// finish shutting down now", so both of those are in the same
// instant.
//
// Because of these two properties, it is the correct thing for
//  - dcontext-aware callee code to just always assume that the
//    Context it has received is a soft Context, and for
//  - dcontext-aware caller code to just always pass the soft Context
//    to things, regardless of whether the code being called it is
//    dcontext-aware or not.
//
//
//
// Why this is necessary
//
// Why not a separate channel for soft shutdown, instead of a full
// Context; something like `<-dcontext.GracefulShutdownStarted(ctx)`?
// Similar to how github.com/datawire/ambassador/pkg/supervisor does?
// Because, as explained above, the correct thing is for
// dcontext-unaware code to observe shutdown when the graceful/soft
// shutdown is initiated.  Using a stand-alone channel instead of a
// Context means that one has to be very careful about passing the
// normal (hard) Context to functions that are not dcontext-aware, as
// to ensure that they don't shutdown too late.
//
// So, why not do the opposite, and use a separate channel for hard
// shutdown?  Because that means you wouldn't have a Context to pass
// to anything that your shutdown routine calls.
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
// WithSoftness(hardCtx).  If the passed-in Context doesn't have
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
