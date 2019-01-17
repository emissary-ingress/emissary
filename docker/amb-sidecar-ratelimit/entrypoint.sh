#!/bin/bash
set -x

trap 'jobs -p | xargs -r kill --' INT

./ratelimit &
./apictl rls watch -o config &

while test -n "$(jobs -p)"; do
	wait -n
	jobs -p | xargs -r kill --
done
