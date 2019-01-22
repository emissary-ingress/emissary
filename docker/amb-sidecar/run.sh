#!/bin/bash
set -x

dir="${BASH_SOURCE[0]%/*}"
mkdir -p /run/amb
trap 'jobs -p | xargs -r kill --' INT

# Launch each of the worker processes
if test -z "$REDIS_URL"; then
	echo 'Warning: ${REDIS_URL} is not set; not starting ratelimit service'
else
	# Setting the PORT is important only because the default PORT
	# is 8080, which would clash with auth.
	USE_STATSD=false RUNTIME_ROOT=/run/amb/config RUNTIME_SUBDIRECTORY=config PORT=7000 "$dir/ratelimit" &
	"$dir/apictl" rls watch -o /run/amb/config &
fi
if test -z "$AUTH_PROVIDER_URL"; then
	echo 'Warning: ${AUTH_PROVIDER_URL} is not set; not starting auth service'
else
	"$dir/ambassador-oauth" &
fi

# Wait for one of them to quit, then kill the others
while test -n "$(jobs -p)"; do
	wait -n
	jobs -p | xargs -r kill --
done
