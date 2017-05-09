#!/bin/sh

set -ex

env | grep TRAVIS | sort 

# Do we have any non-doc changes?
change_count=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep -v '^docs/' | wc -l)

if [ $change_count -eq 0 ]; then
    echo "No non-doc changes"
    exit 0
fi

# Are we on master?
ONMASTER=

if [ \( "$TRAVIS_BRANCH" = "master" \) -a \( "$TRAVIS_PULL_REQUEST" = "false" \) ]; then
    ONMASTER=yes
fi

# Syntactic sugar really...
onmaster () {
    test -n "$ONMASTER"
}

if onmaster; then
    git checkout ${TRAVIS_BRANCH}

    DOCKER_REGISTRY="datawire"

    set +x
    echo "+docker login..."
    docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}"
    set -x
else
    DOCKER_REGISTRY=-
fi

TYPE=$(python scripts/bumptype.py --verbose)

make new-$TYPE

git status

if onmaster; then
    make tag

    # Push everything to GitHub
    git push --tags https://d6e-automation:${GH_TOKEN}@github.com/datawire/ambassador.git master
else
    echo "not on master; not tagging"
fi
