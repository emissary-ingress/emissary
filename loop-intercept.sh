#!/bin/bash

set -e

cd "${BASH_SOURCE%/*}/" || exit 1  # Good enough for debugging, right?

# Launch and check local container

container=intercept-local-$$
docker run --name=$container --rm -d -h container_on_laptop -p 8080 jmalloc/echo-server
local_port=$(docker inspect --format='{{(index (index .NetworkSettings.Ports "8080/tcp") 0).HostPort}}' $container)

if ! curl -s "localhost:$local_port" | grep "Request served by" | tee /dev/stderr | grep -q -e 'Request served by container_on_laptop'; then
    echo "curl of the local container failed or yielded the wrong output."
    echo "This script attempted to launch the container but failed somehow."
    exit 1
fi

APICTL=./bin_$(go env GOOS)_$(go env GOARCH)/apictl
REMOTE_SVC=echo
NUM_LOOPS=2

APICTL_COMMAND="$APICTL traffic intercept -n :path -m /$$ -t localhost:$local_port $REMOTE_SVC"
CURL_COMMAND="curl -s $REMOTE_SVC/$$"

do_curl() {
    echo "+ $CURL_COMMAND"
    $CURL_COMMAND | grep 'Request served by'
}

do_apictl() {
    echo "+ $APICTL_COMMAND"
    $APICTL_COMMAND &
}

test_curl_was_intercepted() {
    do_curl | tee /dev/stderr | grep -q -e 'Request served by container_on_laptop'
}

# Check cluster environment

if ! do_curl; then
    echo "curl of the remote service ($REMOTE_SVC) failed."
    echo "Is kubectl configured correctly?"
    echo "  make shell"
    echo "Is telepresence outbound running?"
    echo "  make proxy"
    echo "Is the example application ($REMOTE_SVC) running in the cluster?"
    echo "  make deploy"
    exit 1
fi

# Make sure apictl is okay
$APICTL help > /dev/null
echo "+ $APICTL version"
$APICTL version

# Set up for cleanup

cleanup() {
    kill $pid || true
    docker kill $container || true
}

trap 'cleanup' INT EXIT

# Do the test

for idx in $(seq $NUM_LOOPS); do
    echo
    echo $idx "--------------------------"
    echo "Before intercept"
    if test_curl_was_intercepted; then false; fi
    echo
    do_apictl
    pid=$!
    for trial in {1..3}; do
        sleep $((3 + RANDOM % 4))
        echo "Trial $trial for loop $idx"
        test_curl_was_intercepted
    done
    echo "Success. Killing intercept..."
    kill $pid
    wait
    echo
    echo "After intercept"
    if test_curl_was_intercepted; then false; fi
done

echo "Done."
