#!/bin/bash

exe="${BASH_SOURCE[0]%/*}/amb-sidecar"
trap 'jobs -p | xargs -r kill --' INT

launch() {
	(
		trap 'echo "Exited with $?: $*"' EXIT
		env "$@"
	) &
}

RUN_DIR=/tmp/amb
mkdir -p ${RUN_DIR}

export RLS_RUNTIME_DIR=${RUN_DIR}/config

# Launch each of the worker processes
if test -z "$REDIS_URL"; then
	echo 'Warning: ${REDIS_URL} is not set; not starting ratelimit service'
else
	# Setting the PORT is important only because the default PORT
	# is 8080, which would clash with non-root-Ambassador when
	# running as a sidecar.
	launch USE_STATSD=false RUNTIME_ROOT=${RUN_DIR}/config RUNTIME_SUBDIRECTORY=config PORT=7000 "$exe" ratelimit
fi
launch "$exe" main

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
