#!/usr/bin/env bash

set -o errexit
set -o nounset

if [ -z $(DOCKER_USERNAME) ]; then echo 'DOCKER_USERNAME not defined'; exit 1; fi
if [ -z $(DOCKER_PASSWORD) ]; then echo 'DOCKER_PASSWORD not defined'; exit 1; fi

printf $(DOCKER_PASSWORD) | docker login -u=$(DOCKER_USERNAME) --password-stdin $(DOCKER_REGISTRY)

docker build . -t $(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_COMMIT)
docker push $(DOCKER_REGISTRY)/$(DOCKER_REPO):$(TRAVIS_COMMIT)