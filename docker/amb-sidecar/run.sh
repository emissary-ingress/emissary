#!/bin/bash

export APRO_AUTH_PORT=${APRO_AUTH_PORT:-8500} # Auth gRPC
export GRPC_PORT=${GRPC_PORT:-8501} # RLS gRPC
export DEBUG_PORT=${DEBUG_PORT:-8502} # RLS debug (HTTP)
export PORT=${PORT:-8503} # RLS HTTP ???

if test -z "$REDIS_URL"; then
	echo 'Error: ${REDIS_URL} is not set; not starting'
	exit 1
fi
if test -z "$REDIS_SOCKET_TYPE"; then
	echo 'Error: ${REDIS_SOCKET_TYPE} is not set; not starting'
	exit 1
fi

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
launch USE_STATSD=false RUNTIME_ROOT=${RUN_DIR}/config RUNTIME_SUBDIRECTORY=config "$exe" ratelimit
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
