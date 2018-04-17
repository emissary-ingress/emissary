#!/usr/bin/env bash

export DOCKER_USERNAME="${DOCKER_USERNAME:?DOCKER_USERNAME is not set or empty}"
export DOCKER_PASSWORD="${DOCKER_PASSWORD:?DOCKER_PASSWORD is not set or empty}"
export DOCKER_REGISTRY="${DOCKER_REGISTRY:?quay.io}"

DOCKER_REPO_NAME="${DOCKER_REPO_NAME:?datawire/ambassador-gh369}"
if [[ "$DOCKER_REGISTRY" != "-" ]]; then
    DOCKER_REPO_NAME="${DOCKER_REGISTRY}/${DOCKER_REPO_NAME}"
fi

export DOCKER_REPO_NAME

export GIT_COMMIT="$(git rev-parse)"
export GIT_COMMIT_SHORT="$(git rev-parse --short HEAD)"

export NETLIFY_SITE="datawire/ambassador"
export NETLIFY_TOKEN="${NETLIFY_TOKEN:?NETLIFY_TOKEN is not set or empty}"

export SCOUT_DISABLE="1"
