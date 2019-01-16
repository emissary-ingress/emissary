#!/bin/bash

set -e

cd "${BASH_SOURCE%/*}/" || exit 1  # Good enough for debugging, right?

APICTL=./bin_$(go env GOOS)_$(go env GOARCH)/apictl
REMOTE_SVC=echo
NUM_LOOPS=2

APICTL_COMMAND="$APICTL traffic intercept -n :path -m /$$ -t localhost:8080 $REMOTE_SVC"
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

# Check environment

if ! curl -s localhost:8080 | grep "Request served by" | tee /dev/stderr | grep -q -e 'Request served by container_on_laptop'; then
    echo "curl of the local container failed or yielded the wrong output."
    echo "You must run the container on your machine so the intercepted"
    echo "traffic has somewhere to go."
    echo "  docker run --rm -d -h container_on_laptop -p 8080:8080 jmalloc/echo-server"
    exit 1
fi

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

# Do the test

trap 'kill $pid' INT  # Perform kill if the script is interrupted

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
