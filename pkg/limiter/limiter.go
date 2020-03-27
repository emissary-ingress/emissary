package limiter

import "time"

// A limiter can be used to rate limit and/or coalesce a series of
// time-based events. This interface captures the logic of deciding
// what action should be taken when an event occurs at a specific time
// T. The possible actions are act upon the event, do nothing, or
// check back after a specific delay.
type Limiter interface {
	// The Limit() method works kinda like an 8 ball. You pass it
	// the time of an event, and it returns one of Yes, No, or
	// "Later". A zero return value means "Yes", the event should
	// be acted upon right away. A negative return value means
	// "No", do nothing, and a positive return value means to
	// check back after the returned delay.
	//
	// The order that this is invoked is important, and it is
	// expected that this is invoked with a monotonically
	// increasing set of timestamps. Typically this will be
	// invoked when an event is generated and will be the result
	// of time.Now(), but for testing or other cases, this could
	// be invoked with any set of historic or future timestamps so
	// long as they are invoked in monotonically increasing order.
	//
	// The result of this (when positive) is always relative to
	// the passed in time. In other words to compute a deadline
	// rather than a delay, you should take the result of Limit()
	// and add it to whatever value you passed in:
	//
	//   deadline = now + Limit(now).
	//
	Limit(now time.Time) time.Duration
}

type limiter struct {
	// we should never fire more than one event within this timespan
	interval time.Duration
	// records the last event we fired
	lastAction time.Time
	// records the point in the future to which we delayed an event
	deadline time.Time
}

// Constructs a new limiter that will coalesce any events occurring
// within the specified interval.
func NewInterval(interval time.Duration) Limiter {
	return &limiter{
		interval: interval,
	}
}

func (l *limiter) Limit(now time.Time) time.Duration {
	since := now.Sub(l.lastAction)
	switch {
	case since >= l.interval:
		// we haven't triggered any event recently, so we
		// trigger this event and record that we did so
		l.lastAction = now
		return 0
	case l.deadline.After(now):
		// we are still waiting for an event that we delayed
		// into the future, so we can drop this one
		return -1
	default:
		// we have multiple events within the same interval,
		// and there are no delayed events, so we delay this
		// one and record the point in the future to which we
		// delayed it
		delay := l.interval - since
		l.deadline = now.Add(delay)
		return delay
	}
}

type composite struct {
	first    Limiter
	second   Limiter
	delay    time.Duration
	started  bool
	deadline time.Time
}

func NewComposite(first, second Limiter, delay time.Duration) Limiter {
	return &composite{
		first:  first,
		second: second,
		delay:  delay,
	}
}

func (c *composite) Limit(now time.Time) time.Duration {
	if !c.started {
		c.started = true
		c.deadline = now.Add(c.delay)
	}

	if now.After(c.deadline) {
		return c.second.Limit(now)
	} else {
		return c.first.Limit(now)
	}
}

type unlimited struct{}

func NewUnlimited() Limiter {
	return &unlimited{}
}

func (u *unlimited) Limit(now time.Time) time.Duration { return 0 }
