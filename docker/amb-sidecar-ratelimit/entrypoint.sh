#!/bin/bash
set -x

trap 'jobs -p | xargs -r kill --' INT

mkdir -p /run/amb
USE_STATSD=false RUNTIME_ROOT=/run/amb/config RUNTIME_SUBDIRECTORY=config ./ratelimit &
./apictl rls watch -o /run/amb/config &

while test -n "$(jobs -p)"; do
	wait -n
	jobs -p | xargs -r kill --
done
