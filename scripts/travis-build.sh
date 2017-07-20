#!/bin/bash

set -ex -o pipefail

env | grep TRAVIS | sort
npm version

# Are we on master?
ONMASTER=

if [ \( "$TRAVIS_BRANCH" = "master" \) -a \( "$TRAVIS_PULL_REQUEST" = "false" \) ]; then
    ONMASTER=yes
fi

# Syntactic sugar really...
onmaster () {
    test -n "$ONMASTER"
}

# Do we have any non-doc changes?
nondoc_changes=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep -v '^docs/' | wc -l)
doc_changes=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep '^docs/' | wc -l)

if [ \( -z "$TRAVIS_COMMIT_RANGE" \) -o \( $nondoc_changes -gt 0 \) ]; then
    if onmaster; then
        git checkout ${TRAVIS_BRANCH}

        DOCKER_REGISTRY="datawire"

        set +x
        echo "+docker login..."
        docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}"
        set -x

        VERSION=v$(python scripts/versioner.py --verbose)
    else
        DOCKER_REGISTRY=-
        VERSION=v$(python scripts/versioner.py --verbose --magic-pre)
    fi

    echo "==== BUILDING IMAGES FOR $VERSION"

    make VERSION=${VERSION} travis-images

    if [ $doc_changes -eq 0 ]; then
        doc_changes=1
    fi

    if onmaster; then
        make VERSION=${VERSION} tag
    else
        echo "not on master; not tagging"
    fi
fi

if [ $doc_changes -gt 0 ]; then
    if onmaster; then
        NETLIFY_DRAFT=
        HRDRAFT=
    else
        NETLIFY_DRAFT=--draft
        HRDRAFT=" (draft)"
    fi

    echo "==== BUILDING DOCS FOR ${VERSION}${HRDRAFT}"

    make VERSION=${VERSION} travis-website

    docs/node_modules/.bin/netlify --access-token ${NETLIFY_TOKEN} \
        deploy --path docs/_book \
               --site-id datawire-ambassador \
               ${NETLIFY_DRAFT}
fi
