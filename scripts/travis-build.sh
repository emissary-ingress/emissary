#!/bin/sh

set -ex

env | sort 

# Do we have any non-doc changes?
change_count=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep -v '^docs/' | wc -l)

if [ $change_count -eq 0 ]; then
    echo "No non-doc changes"
    exit 0
fi

# Are we on master?
if [ "$TRAVIS_BRANCH" == "master" ]; then
    DOCKER_REGISTRY="datawire"
else
    DOCKER_REGISTRY=-
fi

TYPE=$(python scripts/bumptype.py)

make new-$TYPE

if [ "$TRAVIS_BRANCH" == "master" ]; then
    echo "would make tag"
fi

