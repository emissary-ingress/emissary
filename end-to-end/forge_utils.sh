#!/bin/sh

if [ -n "$HERE" ]; then
    HERE=$(pwd)
fi

FORGE="$HERE/forge"

retry () {
    label="$1"
    iteration_function="$2"

    attempts=${3:-3}
    delay=${4:-20}
    succeeded=

    while [ $attempts -gt 0 ]; do
        echo "$attempts: $label"
        attempts=$(( $attempts - 1 ))

        if $iteration_function; then
            succeeded=yes
            break
        fi

        sleep $delay
    done

    if [ -n "$succeeded" ]; then
        return 0
    else
        return 1
    fi
}

_get_forge_iteration () {
    forge_version=$(curl -f -s https://s3.amazonaws.com/datawire-static-files/forge/latest.url)
    if [ $? -ne 0 ]; then return 1; fi

    curl -f -s -L -o "$FORGE" https://s3.amazonaws.com/datawire-static-files/forge/$forge_version/forge
    if [ $? -ne 0 ]; then return 1; fi

    chmod +x "$FORGE"
    return 0
}

get_forge () {
    if [ ! -x "$FORGE" ]; then
        retry "Fetching forge" _get_forge_iteration 3 5

        if [ ! -x "$FORGE" ]; then
            return 1
        fi
    fi

    return 0
}
