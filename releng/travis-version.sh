#!/usr/bin/env bash
set -o nounset
set -o errexit

TRAVIS_TAG="${TRAVIS_TAG:-''}"
printf ${TRAVIS_TAG} | tr '[:upper:]' '[:lower:]' | sed -e 's|-.*||g';
