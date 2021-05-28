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
//    // graceful shutdown has passed and that the program should
//    // terminate *right now*, not-so-gracefully; a "hard shutdown".
//    // In other words, it means, "If you haven't finished shutting
//    // down yet, then you should hurry it up."
//    <-HardContext(softCtx).Done()
//
// When writing code that makes use of a Context, which Context should you use,
// the soft Context or the hard Context?
//
// - For most normal-operation code, you should use the soft Context (since this
// is most code, I recommend that you name it just `ctx`, not `softCtx`).
//
// - For shutdown/cleanup code, you should use the hard Context
// (`dcontext.HardContext(ctx)`).
//
// - For normal-operation code that explicitly may persist in to the
// post-shutdown-initiated grace-period, it may be appropriate to use the hard
// Context.
//
//
// Design principles
//
// - The lifetimes of the various stages (normal operation, shutdown) should be
// signaled with Contexts, rather than with bare channels.  Because each stage
// may want to call a function that takes a Context, there should be a Context
// whose lifetime is scoped to the lifetime of that stage.  If things were
// signaled with bare channels, things taking a Context might shut down too late
// or too early (depending on whether the Context represented hard shutdown (as
// it does for pkg/supervisor) or soft shutdown).
//
// - A soft graceful shutdown is enough fully signal a shutdown, and if
// everything is well-behaved will perform a full shutdown; analogous to how
// clicking the "X" button in the upper corner of a window *should* be enough to
// quit the program.  The harder not-so-graceful is the fallback for when
// something isn't well-behaved (whether that be local code or a remote network
// service) and isn't shutting down in an acceptable time; analogous to the
// window manager prompting you "the program is not responding, would you like
// to force-kill it?"
//
// - There should only be one thing to pass around.  For much of amb-sidecar's
// life (2019-02 to 2020-08), it used two separate Contexts for hard and soft
// cancellation, both explicitly passed around.  This turned out to be clunky:
// (1) it was far too easy to accidentally use the wrong one; (2) the hard
// Context wouldn't have Values attached to the soft Context or vice-versa, so
// if you cared about both Values and cancellation, there were situations where
// *both* were the wrong choice.
//
// - It should be simple and safe to interoperate with dcontext-unaware code;
// code needn't be dcontext-aware if it doesn't have fancy shutdown logic.  This
// is one of the reasons why (in conjunction with "A soft shutdown is enough to
// fully signal a shutdown") the main Context that gets passed around is the
// soft Context and you need to call `dcontext.HardContext(ctx)` to get the hard
// Context; the hard-shutdown case is opt-in to facilitate code that has
// shutdown logic that might not be instantaneous and might need to be cut short
// if it takes too long (such as a server waiting for client connections to
// drain).  Simple code with simple roughly instantaneous shutdown logic need
// not be concerned about hard Contexts and shutdown getting cut short.
//
//
//
// Interfacing dcontext-aware code with dcontext-unaware code
//
// When dcontext-aware code passes the soft Context to dcontext-unaware code,
// then that callee code will shutdown at the beginning of the shutdown grace
// period.  This is correct, because the beginning of that grace period means
// "start shutting down" (on the above principles); if the callee code is
// dcontext-unaware, then shutting down when told to start shutting down is
// tautologically the right thing.  If it isn't the right thing, then the code
// is code that needs to be made dcontext-aware (or adapted to be
// dcontext-aware, as in the HTTP server example).
//
// When dcontext-unaware code passes a hard (normal) Context to dcontext-aware
// code, then that callee code will observe the <-ctx.Done() and
// <-HardContext(ctx).Done() occurring at the same instant.  This is correct,
// because the caller code doesn't allow any grace period between "start
// shutting down" and "you need to finish shutting down now", so both of those
// are in the same instant.
//
// Because of these two properties, it is the correct thing for...
//
// - dcontext-aware caller code to just always pass the soft Context to things,
// regardless of whether the code being called it is dcontext-aware or not, and
// for
//
// - dcontext-aware callee code to just always assume that the Context it has
// received is a soft Context (if for whatever reason it really cares, it can
// check if `ctx == dcontext.HardContext(ctx)`).
package dcontext

import (
	"context"
	"time"
)

type parentHardContextKey struct{}

// WithSoftness returns a copy of the parent "hard" Context with a way of
// getting the parent's Done channel.  This allows the child to have an earlier
// cancellation, triggering a "soft" shutdown, while allowing hard/soft-aware
// functions to use HardContext() to get the parent's Done channel, for a "hard"
// shutdown.
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
func (c childHardContext) String() string                          { return contextName(c.softCtx) + ".HardContext" }

// HardContext takes a child Context that is canceled sooner (a "soft"
// cancellation) and returns a Context with the same values, but with the
// cancellation of a parent Context that is canceled later (a "hard"
// cancellation).
//
// Such a "soft" cancellation Context is created by WithSoftness(hardCtx).  If
// the passed-in Context doesn't have softness (WithSoftness isn't somewhere in
// its ancestry), then it is returned unmodified, because it is already hard.
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
