#!/usr/bin/env bash
set -o nounset
set -o errexit

TRAVIS_TAG="${TRAVIS_TAG:?TRAVIS_TAG not set or empty}"
printf ${TRAVIS_TAG} | tr '[:upper:]' '[:lower:]' | sed -e 's|-.*||g';
