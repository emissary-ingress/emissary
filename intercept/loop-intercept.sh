#!/bin/bash

set -e

cd "${BASH_SOURCE%/*}/" || exit 1  # Good enough for debugging, right?
APICTL=../bin_$(go env GOOS)_$(go env GOARCH)/apictl
COMMAND="$APICTL traffic intercept -m ".*$$.*" -n :path -t 8080 the-app"

if ! curl -s -o /dev/null localhost:8080; then
    echo "You must run the container on your machine so the intercepted"
    echo "traffic has somewhere to go."
    echo "  docker run --rm -d -h container_on_laptop -p 8080:8080 jmalloc/echo-server"
    exit 1
fi

do_curl() {
    echo "+ curl -vs the-app/$$"
    curl -vs the-app/$$ | egrep 'Request served by'
}

if ! do_curl; then
    echo "Is telepresence outbound running?"
    echo "  telepresence outbound"
    echo "Is the example application running?"
    echo "  kubectl apply -f <(.../apictl traffic inject intercept/example/the-app.yaml -d the-app -s the-app -p 8080)"
    exit 1
fi

do_apictl() {
    echo "+ $COMMAND"
    $COMMAND &
}

test_curl_was_intercepted() {
    curl -s the-app/$$ | grep -q -e 'Request served by container_on_laptop'
}

trap 'kill $pid' INT  # Perform kill if the script is interrupted

for idx in {1..300}; do
    echo
    echo $idx "--------------------------"
    do_curl
    if test_curl_was_intercepted; then false; fi
    do_apictl
    pid=$!
    for trial in {1..5}; do
        sleep $((3 + RANDOM % 4))
        echo "Trial $trial for loop $idx"
        do_curl
        if ! test_curl_was_intercepted; then
            echo "Oh no! Failed! Waiting..."
            sleep 86400
        fi
    done
    echo "Success. Killing apictl..."
    kill $pid
    wait
done

echo "Done."
