#!/bin/sh

usage () {
    echo "$(basename $0) [--update-gold] [test-patterns]" >&2
    echo "" >&2
    echo "test-patterns: glob patterns for tests to run. Default to all" >&2
    echo "" >&2
    echo "If --update-gold is given, update gold files (if any) for tests." >&2
}

export SCOUT_DISABLE=1

HERE=$(cd $(dirname $0); pwd)

VALIDATOR_IMAGE="dwflynn/ambassador-envoy:v1.4.0-49-g008635a04"
AMBASSADOR="${HERE}/ambassador.py"

ERRORS=0

error () {
    echo "$@" >&2
    ERRORS=$(( $ERRORS + 1 ))
}

cd ${HERE}/tests
TESTDIR=$(pwd)
UPDATE_GOLD=

case "$1" in
    --update-gold)
        shift
        UPDATE_GOLD=yes
        echo "Updating gold files"
        ;;
    -*)
        usage
        exit 1
        ;;
esac

dirs='[0-9]*'

if [ -n "$1" ]; then
    dirs="$@"
fi

for dir in $dirs
do
    test -d "$dir" || continue

    fqdir="${TESTDIR}/$dir"

    # Is there a TEST_DEFAULT_CONFIG file?
    if [ -f "$fqdir/TEST_DEFAULT_CONFIG" ]; then
        # Yes; test the default-config that's a sibling of this script.
        # (We could've use a symlink for this, but I'm paranoid about git,
        # symlinks, and portability.)
        CONFIGDIR="${HERE}/default-config"
    elif [ -d "$fqdir/config" ]; then
        CONFIGDIR="$fqdir/config"
    else
        error "$dir: missing config"
        continue
    fi

    ENVOY_JSON="${fqdir}/envoy.json"
    GOLD_JSON="${fqdir}/gold.json"
    ENVOY_DIFF="${fqdir}/envoy-diff.out"
    AMBASSADOR_OUT="${fqdir}/ambassador.out"
    ENVOY_OUT="${fqdir}/envoy.out"

    NEEDS_GOLD_UPDATE=

    echo "$dir: starting..."

    # Use Ambassador to generate an envoy.json...
    if ! python "$AMBASSADOR" config "$CONFIGDIR" "$ENVOY_JSON" > "$AMBASSADOR_OUT" 2>&1; then
        error "$dir: ambassador could not generate config"
        cat "$AMBASSADOR_OUT" >&2
        continue
    fi

    # Check against the gold file, if there is one.
    if [ -f "$GOLD_JSON" ]; then
        if [ -n "$UPDATE_GOLD" ]; then
            cp "$ENVOY_JSON" "$GOLD_JSON"
            echo "$dir: updated gold file"
        elif ! diff -u "$GOLD_JSON" "$ENVOY_JSON" > "$ENVOY_DIFF"; then
            error "$dir: envoy.json does not match gold.json"
            cat "$ENVOY_DIFF" >&2
            # DO NOT CONTINUE HERE -- let the envoy validation proceed.
            # Do remember that we need a goldfile update though.
            NEEDS_GOLD_UPDATE=yes
        fi
    fi

    # Is it valid? Use Envoy to check.
    if ! docker run -it --rm --volume="$fqdir":/etc/ambassador "$VALIDATOR_IMAGE" \
        /usr/local/bin/envoy --base-id 1 --mode validate -c /etc/ambassador/envoy.json > "$ENVOY_OUT"; then
        error "$dir: envoy could not validate config"
        if [ -f "$ENVOY_OUT" ]; then cat "$ENVOY_OUT" >&2; fi
        continue
    fi

    if ! tail -1 "$ENVOY_OUT" | tr -d '\015\012' | egrep 'OK$' > /dev/null; then
        error "$dir: envoy config was not valid"
        if [ -f "$ENVOY_OUT" ]; then cat "$ENVOY_OUT" >&2; fi
        continue
    fi

    if [ -n "$NEEDS_GOLD_UPDATE" ]; then
        echo "$dir: OK but needs goldfile update"
    else
        echo "$dir: OK"
    fi
done

echo "Errors: $ERRORS"
exit $ERRORS

