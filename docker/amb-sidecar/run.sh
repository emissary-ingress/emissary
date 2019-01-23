#!/bin/bash

exe="${BASH_SOURCE[0]%/*}/amb-sidecar"
trap 'jobs -p | xargs -r kill --' INT

launch() {
	(
		trap 'echo "Exited with $?: $*"' EXIT
		env "$@"
	) &
}

# Launch each of the worker processes
if test -z "$REDIS_URL"; then
	echo 'Warning: ${REDIS_URL} is not set; not starting ratelimit service'
else
	mkdir -p /run/amb
	# Setting the PORT is important only because the default PORT
	# is 8080, which would clash with auth.
	launch USE_STATSD=false RUNTIME_ROOT=/run/amb/config RUNTIME_SUBDIRECTORY=config PORT=7000 "$exe" ratelimit
	launch "$exe" rls-watch -o /run/amb/config
fi
if test -z "$AUTH_PROVIDER_URL"; then
	echo 'Warning: ${AUTH_PROVIDER_URL} is not set; not starting auth service'
else
	launch "$exe" auth
fi

# Wait for one of them to quit, then kill the others
wait -n
r=$?
echo ' ==> One of the worker processes exited; shutting down the others <=='
while test -n "$(jobs -p)"; do
	jobs -p | xargs -r kill --
	wait -n
done
echo 'Finished shutting down'
exit $r
