#!/bin/sh

set -e

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

echo "Building images"

set -x
docker build -q -t dwflynn/demo:1.0.0 --build-arg VERSION=1.0.0 demo-service
docker build -q -t dwflynn/demo:2.0.0 --build-arg VERSION=2.0.0 demo-service
docker build -q -t dwflynn/demo:1.0.0tls --build-arg VERSION=1.0.0 --build-arg TLS=--tls demo-service
docker build -q -t dwflynn/demo:2.0.0tls --build-arg VERSION=2.0.0 --build-arg TLS=--tls demo-service

# seriously? there's no docker push --quiet???
docker push dwflynn/demo:1.0.0
docker push dwflynn/demo:2.0.0
docker push dwflynn/demo:1.0.0tls
docker push dwflynn/demo:2.0.0tls
set +x

# For linify
export MACHINE_READABLE=yes

for dir in 0*; do
    echo
    echo "================================================================"
    echo "${dir}..."

    if sh $dir/test.sh 2>&1 | python linify.py test.log; then
        echo "${dir} PASSED"
    else
        echo "${dir} FAILED"

        echo "================ start captured output"
        cat test.log
        echo "================ end captured output"
    fi
done
