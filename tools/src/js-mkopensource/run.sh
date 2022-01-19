#!/bin/env bash
set -ex

DOCKERFILE=$(pwd)/docker

pushd /home/andres/source/production/emissary/docker/test-ratelimit
DOCKER_BUILDKIT=1 docker build -f "${DOCKERFILE}/Dockerfile" --output out .
popd

go build .
cat /home/andres/source/production/emissary/tools/src/js-mkopensource/testdata/dependencies.json | ./js-mkopensource
