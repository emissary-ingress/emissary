package ambex

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/datawire/dlib/dlog"
	"github.com/emissary-ingress/emissary/v3/pkg/debug"
)

// An Update encapsulates everything needed to perform an update (of envoy configuration). The
// version string is for logging purposes, the Updator func does the actual work of updating.
type Update struct {
	Version string
	Update  func() error
}

// Function type for fetching memory usage as a percentage.
type MemoryGetter func() int

// The Updator function will run forever (or until the ctx is canceled) and look for updates on the
// incoming channel. If memory usage is constrained as reported by the getUsage function, updates
// will be rate limited to guarantee that there are only so many stale configs in memory at a
// time. The function assumes updates are cumulative and it will drop old queued updates if a new
// update arrives.
func Updater(ctx context.Context, updates <-chan Update, getUsage MemoryGetter) error {
	drainTime := GetAmbassadorDrainTime(ctx)
	ticker := time.NewTicker(drainTime)
	defer ticker.Stop()
	return updaterWithTicker(ctx, updates, getUsage, drainTime, ticker, time.Now)
}

type debugInfo struct {
	Times              []time.Time `json:"times"`
	StaleCount         int         `json:"staleCount"`
	StaleMax           int         `json:"staleMax"`
	Synced             bool        `json:"synced"`
	DisableRatelimiter bool        `json:"disableRatelimiter"`
}

func updaterWithTicker(ctx context.Context, updates <-chan Update, getUsage MemoryGetter,
	drainTime time.Duration, ticker *time.Ticker, clock func() time.Time) error {

	dbg := debug.FromContext(ctx)
	info := dbg.Value("envoyReconfigs")

	// Is the rate-limiter meant to be active at all?
	disableRatelimiter, err := strconv.ParseBool(os.Getenv("AMBASSADOR_AMBEX_NO_RATELIMIT"))

	if err != nil {
		disableRatelimiter = false
	}

	if disableRatelimiter {
		dlog.Info(ctx, "snapshot ratelimiter DISABLED")
	}

	// This slice holds the times of any updates we have made. This lets us compute how many stale
	// configs are being held in memory since we can filter this list down to just those times that
	// are between now - drain-time and now, i.e. we keep only the events that are more recent than
	// drain-time ago.
	updateTimes := []time.Time{}

	// This variable holds the most recent desired configuration.
	var latest Update
	gotFirst := false
	pushed := false
	for {
		// The basic idea here is that we wakeup whenever we either a) get a new snapshot to update,
		// or b) the timer ticks. In case a) we update the "latest" variable so that it always holds
		// the most recent desired Update. In either case, we filter the list of updateTimes so we
		// know exactly how many updates are in memory, and then based on that we decide whether we
		// can do another reconfig or whether we should wait until the next (tick|update) whichever
		// happens first.

		var now time.Time
		tick := false
		select {
		case up := <-updates:
			latest = up
			pushed = false
			gotFirst = true
			now = clock()
		case now = <-ticker.C:
			if pushed {
				continue
			}
			tick = true
		case <-ctx.Done():
			return nil
		}

		// Remove updates that were longer than drain-time ago
		updateTimes = gcUpdateTimes(updateTimes, now, drainTime)

		usagePercent := getUsage()

		if disableRatelimiter {
			usagePercent = 0
		}

		var maxStaleReconfigs int
		switch {
		case usagePercent >= 90:
			// With the default 10 minute drain time this works out to an average of one reconfig
			// every 10 minutes. This will guarantee the minimum possible memory usage due to stale
			// configs.
			maxStaleReconfigs = 1
		case usagePercent >= 80:
			// With the default 10 minute drain time this works out to one reconfig every 40
			// seconds on average within the window. (They could all happen in one burst.)
			maxStaleReconfigs = 15
		case usagePercent >= 70:
			// With the default 10 minute drain time this works out to one reconfig every 20
			// seconds on average within the window. (They could all happen in one burst.)
			maxStaleReconfigs = 30
		case usagePercent >= 60:
			// With the default 10 minute drain time this works out to one reconfig every 10
			// seconds on average within the window. (They could all happen in one burst.)
			maxStaleReconfigs = 60
		case usagePercent >= 50:
			// With the default 10 minute drain time this works out to one reconfig every 5 seconds
			// on average within the window. (They could all happen in one burst.)
			maxStaleReconfigs = 120
		default:
			// Zero means no limit. This is what we want by default when memory usage is in the 0 to
			// 50 range.
			maxStaleReconfigs = 0
		}

		staleReconfigs := len(updateTimes)

		info.Store(debugInfo{updateTimes, staleReconfigs, maxStaleReconfigs, pushed, disableRatelimiter})

		// Decide if we have enough capacity left to perform a reconfig.
		if maxStaleReconfigs > 0 && staleReconfigs >= maxStaleReconfigs {
			if !tick {
				dlog.Warnf(ctx, "Memory Usage: throttling reconfig %+v due to constrained memory with %d stale reconfigs (%d max)",
					latest.Version, staleReconfigs, maxStaleReconfigs)
			}
			continue
		}

		// This is just in case we get a timer tick before the first update actually arrives.
		if !gotFirst {
			continue
		}

		// This is going to do the actual work of pushing an update.
		err := latest.Update()
		if err != nil {
			return err
		}

		// Since we just pushed an update, we add the current time to the set of update times.
		updateTimes = append(updateTimes, now)
		dlog.Infof(ctx, "Pushing snapshot %+v", latest.Version)
		pushed = true

		info.Store(debugInfo{updateTimes, staleReconfigs, maxStaleReconfigs, pushed, disableRatelimiter})
	}
}

// The gcUpdateTimes function filters out timestamps that should have drained by now.
func gcUpdateTimes(updateTimes []time.Time, now time.Time, drainTime time.Duration) []time.Time {
	result := []time.Time{}
	for _, ut := range updateTimes {
		if ut.Add(drainTime).After(now) {
			result = append(result, ut)
		}
	}
	return result
}

// The GetAmbassadorDrainTime function retuns the AMBASSADOR_DRAIN_TIME env var as a time.Duration
func GetAmbassadorDrainTime(ctx context.Context) time.Duration {
	s := os.Getenv("AMBASSADOR_DRAIN_TIME")
	if s == "" {
		s = "600"
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		dlog.Printf(ctx, "Error parsing AMBASSADOR_DRAIN_TIME: %v", err)
		i = 600
	}

	return time.Duration(i) * time.Second
}
