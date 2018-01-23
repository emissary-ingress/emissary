#!/bin/bash

set -e

# Don't build on version-tag pushes.
if [ $(echo "$TRAVIS_BRANCH" | egrep -c '^v[0-9][0-9\.]*$') -gt 0 ]; then
    echo "No need to build $TRAVIS_BRANCH"
    exit 0
fi

env | grep TRAVIS | sort
npm version
aws --version

export DOCKER_REGISTRY
ECHO=echo
DRYRUN=yes

if [ -n "$TRAVIS" ]; then
    ECHO=
    DRYRUN=
fi

# Syntactic sugar really...
dryrun () {
    test -n "$DRYRUN"
}

if dryrun; then
    echo "======== DRYRUN"
else
    echo "======== RUNNING"
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

# Do we have any non-doc changes?
DIFF_RANGE=${TRAVIS_COMMIT_RANGE:-HEAD^}

echo "======== Diff summary ($DIFF_RANGE)"
git diff --stat "$DIFF_RANGE"

nondoc_changes=$(git diff --name-only "$DIFF_RANGE" | grep -v '^docs/' | wc -l | tr -d ' ')
doc_changes=$(git diff --name-only "$DIFF_RANGE" | grep -e '^docs/' | wc -l | tr -d ' ')

# Default VERSION to _the current version of Ambassador._
VERSION=$(python scripts/versioner.py)

echo "========"
echo "Base version ${VERSION}; non-doc changes ${nondoc_changes}, doc changes ${doc_changes}"
echo "========"

# Do we have any non-doc changes?
if [ \( -z "$TRAVIS_COMMIT_RANGE" \) -o \( $nondoc_changes -gt 0 \) ]; then
    # Yes. Are we on master?
    if onmaster; then
        # Yes. This is a Real Official Build(tm) -- make sure git is in a sane state...
        git checkout ${TRAVIS_BRANCH}

        # ...make sure we're interacting with our official Docker repo...
        DOCKER_REGISTRY="quay.io/datawire"

        set +x
        echo "+docker login..."
        $ECHO docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}" quay.io
        set -x

        # We _won't_ try to figure out a magic prebuild number for real builds.
        MAGIC_PRE=""
    else
        # We're not on master, so we're not going to push anywhere...
        DOCKER_REGISTRY=-

        # ...and we _will_ do a magic prebuild number.
        MAGIC_PRE="--magic-pre"
    fi

    # OK. Figure out the correct version number, including updating app.json...
    VERSION=$(python scripts/versioner.py --bump --only-if-changes --scout-json=app.json $MAGIC_PRE)

    # ...then actually build our Docker images.
    echo "==== BUILDING IMAGES FOR $VERSION"

    $ECHO make VERSION=${VERSION} EXTRA_DOCKER_ARGS=-q travis-images

    # Assume we'll push app.json to, uh, app.json...
    SCOUT_KEY=app.json

    if onmaster; then
        # ...and, if we're on master, tag this version...
        $ECHO make VERSION=${VERSION} tag

        # ...and push the tag.
        $ECHO git push --tags https://d6e-automation:${GH_TOKEN}@github.com/datawire/ambassador.git master
    else
        # If not on master, don't tag...
        echo "not on master; not tagging"

        # ...and push app.json to testapp.json for later examination.
        SCOUT_KEY=testapp.json
    fi

    # Push new info to AWS
    export AWS_DEFAULT_REGION=us-east-1
    $ECHO aws s3api put-object \
        --bucket scout-datawire-io \
        --key ambassador/$SCOUT_KEY \
        --body app.json

    # Finally, force a doc build whenever the code changes.
    if [ $doc_changes -eq 0 ]; then
        doc_changes=1
    fi
else
    echo "Not building images for $VERSION; no non-doc changes"
fi

# OK. Any doc changes?
if [ $doc_changes -gt 0 ]; then
    # Yes, so we'll run a doc build, for which we always use the Datawire registry.
    # (why? 'cause there's no way to figure out WTF domain name Netlify will push to
    # at this point)
    DOCKER_REGISTRY=quay.io/datawire

    if onmaster; then
        # If on master, we publish instead of just leaving everything in draft mode.
        NETLIFY_DRAFT=
    else
        # If not on master, we leave all the Netlify stuff in draft mode...
        NETLIFY_DRAFT=--draft

        # ...and, if the version number has no '-' already, we append "-draft" to it
        # so that we can push something real if we want to.
        if [ $(echo ${VERSION} | grep -c -e '-') -eq 0 ]; then
            VERSION="${VERSION}-draft"
        fi
    fi

    echo "==== BUILDING DOCS FOR ${VERSION}"

    $ECHO make VERSION=${VERSION} travis-website

    $ECHO docs/node_modules/.bin/netlify --access-token ${NETLIFY_TOKEN} \
        deploy --path docs/_book \
               --site-id datawire-ambassador \
               ${NETLIFY_DRAFT}
else
    echo "Not building docs for $VERSION; no doc changes"
fi
