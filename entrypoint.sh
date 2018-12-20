#!/bin/sh

# The trap CHLD doesn't work without this.
set -m

PIDS=""

kill_pids() {
    echo "Killing ${PIDS}"
    kill ${PIDS} > /dev/null 2>&1
}

trap kill_pids INT CHLD

./ratelimit &
PIDS+=$!

./apictl rls watch -o config &
PIDS+=" $!"

echo PIDS=${PIDS}

wait $PIDS
