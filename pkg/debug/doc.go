// The debug package is intended to aid in live debugging of misbehaving Ambassadors in
// production. Its role is somewhat akin to a debugger and/or a profiler in that it gives you
// visiblity into important internal state and performance charactaristics of Ambassador, however
// unlike a debugger and/or profiler it is intended to function well in a production context.
//
// The debug package is also very complementary to a logging library. To make an analogy to GPS
// Navigation, a logging library is kind of like directions that your nav system produces. The
// information that logging provides is often similar to "turn left", "go straight", etc. This is
// super useful if you are trying to understand how your code got to the wrong destination, but
// often if you are trying to live debug a misbehaving Ambassador in production, you don't
// necessarily care as much about how it got to where it is. You care about *quickly* understanding
// where it is and how to get it to someplace better. The debug library is intended to help with
// that by giving you an understanding of exactly where Ambassador is without requiring the tedius
// exercise of pooring over the logs while also staring at and mentally executing the code in order
// to reconstruct the relevant state. Instead of doing that we just make the code capable of telling
// us exactly where it has ended up.
//
// So how does this work? Well there is a new endpoint at `localhost:8877/debug` and you can run
// `curl localhost:8877/debug` to see some useful information.
//
// There are currently two kinds of debug information that it exposes:
//
// 1. Timers
//
// We've learned that the timing of various actions that ambassador takes is actually very important
// to its production behavior. Things like:
//
//   - how long it takes us to load secrets
//   - how long it takes us to validate envoy configuration
//   - how long it takes us to compute a new envoy configuration
//   - how long it takes us to respond to a probe request
//
// Anywhere in the code can now easily add new timing information to this debug endpoint by doing
// the following:
//
//	dbg := debug.FromContext(ctx)
//	timer := dbg.Timer("myAction")
//
//	timer.Time(func() {
//	  // ... do some work
//	})
//
//	// or
//
//	stop := timer.Start()
//	// ... do some work
//	stop()
//
//	// or
//
//	func() {
//	  defer timer.Start()() // this is really the same as the stop case but using defer to guarantee stop gets called
//	  // ... do some work
//	}()
//
// There are also special purpose convenience middleware for adding timing information to http
// handlers:
//
//	handler = timer.TimedHandler(handler)
//	// or
//	handlerFunc = timer.TimedHandlerFunc(handlerFunc)
//
// A timer tracks a count, minimum, maximum, and average timing info for all actions:
//
//	{
//	  "timers": {
//	    "check_alive": "0, 0s/0s/0s",
//	    "check_ready": "0, 0s/0s/0s",
//	    "consulUpdate": "0, 0s/0s/0s",
//	    "katesUpdate": "615, 29.385µs/297.114µs/95.220222ms",
//	    "notifyWebhook:diagd": "2, 1.206967947s/1.3298432s/1.452718454s",
//	    "notifyWebhooks": "2, 1.207007216s/1.329901037s/1.452794859s",
//	    "parseAnnotations": "2, 21.944µs/22.541µs/23.138µs",
//	    "reconcileConsul": "2, 50.104µs/55.499µs/60.894µs",
//	    "reconcileSecrets": "2, 18.704µs/20.786µs/22.868µs"
//	  },
//	  ...
//	}
//
// 2. Atomic Values
//
// Another tool in the toolkit for externalizing relevant state is atomic values. Anywhere in the
// code can now expose important values in the following way:
//
//	dbg := debug.FromContext(ctx)
//	value := dbg.Value("myValue")
//
//	// ...
//
//	value.Store("blah")
//
//	// or
//
//	v := StructThatCanBeJsonMarshalled{...}
//	value.Store(v)
//
// The debug endpoint will now show the current value for "myValue". The only requirement for the
// stored value is that it can be marshalled as json.
//
// We currently use this to expose how much memory ambassador is using as well as the state of the
// envoy reconfiguration, but there are lots of other relevant state we can/should expose in the
// future as we understand what is important to ambassador's behavior:
//
//	{
//	  ...
//	  "values": {
//	    "envoyReconfigs": {
//	      "times": [
//	        "2020-11-06T13:13:24.218707995-05:00",
//	        "2020-11-06T13:13:27.185754494-05:00",
//	        "2020-11-06T13:13:28.612279777-05:00"
//	      ],
//	      "staleCount": 2,
//	      "staleMax": 0,
//	      "synced": true
//	    },
//	    "memory": "39.68Gi of Unlimited (0%)"
//	  }
//	}
//
// The full output of the debug endpoint now currently looks like this:
//
//	$ curl localhost:8877/debug
//	{
//	  "timers": {
//	    # these two timers track how long it takes to respond to liveness and readiness probes
//	    "check_alive": "7, 45.411495ms/61.85999ms/81.358927ms",
//	    "check_ready": "7, 49.951304ms/61.976205ms/86.279038ms",
//
//	    # these two timers track how long we spend updating our in-memory snapshot when our kubernetes
//	    # watches tell us something has changed
//	    "consulUpdate": "0, 0s/0s/0s",
//	    "katesUpdate": "3382, 28.662µs/102.784µs/95.220222ms",
//
//	    # These timers tell us how long we spend notifying the sidecars if changed input. This
//	    # includes how long the sidecars take to process that input.
//	    "notifyWebhook:diagd": "2, 1.206967947s/1.3298432s/1.452718454s",
//	    "notifyWebhooks": "2, 1.207007216s/1.329901037s/1.452794859s",
//
//	    # This timer tells us how long we spend parsing annotations.
//	    "parseAnnotations": "2, 21.944µs/22.541µs/23.138µs",
//
//	    # This timer tells us how long we spend reconciling changes to consul inputs.
//	    "reconcileConsul": "2, 50.104µs/55.499µs/60.894µs",
//
//	    # This timer tells us how long we spend reconciling secrets related changes to ambassador
//	    # inputs.
//	    "reconcileSecrets": "2, 18.704µs/20.786µs/22.868µs"
//	  },
//	  "values": {
//	    "envoyReconfigs": {
//	      "times": [
//	        "2020-11-06T13:13:24.218707995-05:00",
//	        "2020-11-06T13:13:27.185754494-05:00",
//	        "2020-11-06T13:13:28.612279777-05:00"
//	      ],
//	      "staleCount": 2,
//	      "staleMax": 0,
//	      "synced": true
//	    },
//	    "memory": "39.73Gi of Unlimited (0%)"
//	  }
//	}
//
// TODO:
//
//   - Extend the API to permit a description of a given Timer or Value in the code. This will let us
//     implement `curl localhost:8877/debug/help` or similar instead of manually keeping the above
//     docs up to date.
package debug
