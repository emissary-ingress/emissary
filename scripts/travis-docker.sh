#!/usr/bin/env bash

set -o errexit
set -o nounset

docker build . -t $(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_COMMIT)
docker push $(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_COMMIT)