#!/bin/sh

set -ex

env | grep TRAVIS | sort

# Do we have any non-doc changes?
change_count=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep -v '^docs/' | wc -l)

if [ -n "$TRAVIS_COMMIT_RANGE" ] && [ $change_count -eq 0 ]; then
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

    VERSION=v$(python scripts/versioner.py --verbose)
    NETLIFY_ENVIRONMENT=production
else
    DOCKER_REGISTRY=-
    VERSION=v$(python scripts/versioner.py --verbose --magic-pre)
    NETLIFY_ENVIRONMENT=branch-deploy
fi

make VERSION=${VERSION}

netlify --access-token ${NETLIFY_TOKEN} \
    deploy --path docs/_book \
           --site-id datawire-ambassador
           --env ${NETLIFY_ENVIRONMENT}

git status

if onmaster; then
    make VERSION=$(VERSION) tag
else
    echo "not on master; not tagging"
fi
