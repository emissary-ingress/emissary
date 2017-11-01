#!/bin/bash

slashify () {
    thing="$1"

    if [ -n "$thing" ]; then
        case "$thing" in
            */) ;;
            *) thing="$thing/"
        esac
    fi

    echo "$thing"
}

if [ -z "${DOCKER_REGISTRY}" ]; then
  echo "DOCKER_REGISTRY must be set" >&2
  exit 1
fi

if [ "$DOCKER_REGISTRY" = "-" ]; then
    unset DOCKER_REGISTRY
fi

DOCKER_REGISTRY=$(slashify "$DOCKER_REGISTRY")

# Default to using DOCKER_REGISTRY, but allow overriding.
AMREG=$(slashify "${AMBASSADOR_REGISTRY:-$DOCKER_REGISTRY}")

# Default to using DOCKER_REGISTRY, but allow overriding.
STREG=$(slashify "${STATSD_REGISTRY:-$DOCKER_REGISTRY}")

cat <<EOF
export DOCKER_REGISTRY="$DOCKER_REGISTRY"
export AMREG="$AMREG"
export STREG="$STREG"
EOF
