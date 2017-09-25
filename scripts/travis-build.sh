#!/bin/bash

set -ex

# Don't build on version-tag pushes.
if [ $(echo "$TRAVIS_BRANCH" | egrep -c '^v[0-9][0-9\.]*$') -gt 0 ]; then
    echo "No need to build $TRAVIS_BRANCH"
    exit 0
fi

env | grep TRAVIS | sort
npm version
aws --version

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
nondoc_changes=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep -v '^docs/' | wc -l | tr -d ' ')
doc_changes=$(git diff --name-only "$TRAVIS_COMMIT_RANGE" | grep -e '^docs/' | wc -l | tr -d ' ')

# Default a VERSION
VERSION=$(python scripts/versioner.py --only-if-changes --scout-json=app.json)

if [ \( -z "$TRAVIS_COMMIT_RANGE" \) -o \( $nondoc_changes -gt 0 \) ]; then
    if onmaster; then
        git checkout ${TRAVIS_BRANCH}

        DOCKER_REGISTRY="datawire"

        set +x
        echo "+docker login..."
        docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}"
        set -x
    else
        DOCKER_REGISTRY=-

        # Override the VERSION for a non-master build.
        VERSION=$(python scripts/versioner.py --only-if-changes --magic-pre)
    fi

    echo "==== BUILDING IMAGES FOR $VERSION"

    make VERSION=${VERSION} EXTRA_DOCKER_ARGS=-q travis-images

    if [ $doc_changes -eq 0 ]; then
        doc_changes=1
    fi

    # Assume we'll push app.json to ...app.json
    SCOUT_KEY=app.json

    if onmaster; then
        make VERSION=${VERSION} tag

        # Push everything to GitHub
        git push --tags https://d6e-automation:${GH_TOKEN}@github.com/datawire/ambassador.git master
    else
        echo "not on master; not tagging"

        SCOUT_KEY=testapp.json
    fi

    # Push new info to AWS
    export AWS_DEFAULT_REGION=us-east-1
    aws s3api put-object \
        --bucket scout-datawire-io \
        --key ambassador/$SCOUT_KEY \
        --body app.json
fi

if [ $doc_changes -gt 0 ]; then
    # Always build the docs assuming the Datawire registry.
    DOCKER_REGISTRY=datawire

    if onmaster; then
        NETLIFY_DRAFT=
        HRDRAFT=
    else
        NETLIFY_DRAFT=--draft
        HRDRAFT=" (draft)"
        VERSION="${VERSION}-draft"
    fi

    echo "==== BUILDING DOCS FOR ${VERSION}${HRDRAFT}"

    make VERSION=${VERSION} travis-website

    docs/node_modules/.bin/netlify --access-token ${NETLIFY_TOKEN} \
        deploy --path docs/_book \
               --site-id datawire-ambassador \
               ${NETLIFY_DRAFT}
fi
