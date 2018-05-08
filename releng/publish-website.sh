#!/usr/bin/env bash
set -o errexit
set -o nounset

RELEASE_TYPE=${RELEASE_TYPE:?RELEASE_TYPE not set or empty}

NETLIFY_TOKEN=${NETLIFY_TOKEN:?NETLIFY_TOKEN not set or empty}
NETLIFY_SITE=${NETLIFY_SITE:?NETLIFY_SITE not set or empty}
NETLIFY_OPTS=${NETLIFY_OPTS:-"--draft"}
if [[ "$RELEASE_TYPE" == "stable" ]]; then
    NETLIFY_OPTS=
fi

docs/node_modules/.bin/netlify \
	--access-token ${NETLIFY_TOKEN} \
	deploy \
	${NETLIFY_OPTS} \
	--path docs/_book \
	--site-id ${NETLIFY_SITE}
