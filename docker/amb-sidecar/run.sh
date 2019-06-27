#!/bin/bash

export APRO_HTTP_PORT=${APRO_HTTP_PORT:-8500}

if test -z "$REDIS_URL"; then
	echo 'Error: ${REDIS_URL} is not set; not starting'
	exit 1
fi
if test -z "$REDIS_SOCKET_TYPE"; then
	echo 'Error: ${REDIS_SOCKET_TYPE} is not set; not starting'
	exit 1
fi

run_dir=/tmp/amb
mkdir -p ${run_dir}

export RLS_RUNTIME_DIR=${run_dir}/config
export USE_STATSD=false
export RUNTIME_ROOT=${run_dir}/config
export RUNTIME_SUBDIRECTORY=config
exec "${BASH_SOURCE[0]%/*}/amb-sidecar" main
