#!/bin/bash

run_dir=/tmp/amb
mkdir -p ${run_dir}

export RLS_RUNTIME_DIR=${run_dir}/config
export USE_STATSD=false
export RUNTIME_ROOT=${run_dir}/config
export RUNTIME_SUBDIRECTORY=config
exec "${BASH_SOURCE[0]%/*}/amb-sidecar" main
