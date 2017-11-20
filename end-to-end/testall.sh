#!/bin/sh

set -e

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

echo "Building images"

set -x
docker build -q -t dwflynn/demo:1.0.0 --build-arg VERSION=1.0.0 demo-service
docker build -q -t dwflynn/demo:2.0.0 --build-arg VERSION=2.0.0 demo-service

# seriously? there's no docker push --quiet???
docker push dwflynn/demo:1.0.0
docker push dwflynn/demo:2.0.0
set +x

for dir in 0*; do
    echo "========"
    echo "${dir}..."
    echo

    if sh $dir/test.sh; then
        echo "${dir} PASSED"
    else
        echo "${dir} FAILED"
    fi
done
