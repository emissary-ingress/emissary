#!/usr/bin/env bash

if [[ "$TRAVIS_TAG" != "" ]]; then
    printf ${TRAVIS_TAG};
elif [[ "${GIT_BRANCH}" =~ ^rc/.* ]]; then
    printf ${GIT_BRANCH} | tr '[:upper:]' '[:lower:]' | sed -e 's|rc/||g';
else
    printf "";
fi