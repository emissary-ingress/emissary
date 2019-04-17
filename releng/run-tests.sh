#!/bin/sh

# Coverage checks are totally broken right now. I suspect that it's
# probably the result of all the Ambassador stuff actually happen in
# Docker containers. To restore it, first add
# 
# --cov=ambassador --cov=ambassador_diag --cov-report term-missing
#
# to the pytest line, and, uh, I guess recover and merge all the .coverage 
# files from the containers??

TEST_ARGS="--tb=short"

if [ -n "${TEST_NAME}" ]; then
    TEST_ARGS+=" -k ${TEST_NAME}"
    pytest ${TEST_ARGS}
    RESULT=$?
else
    # Split up the tests so we don't run them all at once. This is an
    # eggregious hack that depends on the "main[...]" syntax of the
    # pytest parameterized test notation.

    # make sure we include non parameterized tests, but they don't
    # seem to exist in CI, so we need to check (running an empty set
    # of tests counts as a failure)
    UNPARAM="not ["
    if pytest -qq --collect-only -k "${UNPARAM}"; then
        PATTERNS=("${UNPARAM}")
    else
        PATTERNS=()
    fi

    # split up the parameterized tests by starting letter
    for T in {A..Z}; do
        # we need to check if the pattern actually matches any tests,
        # because running an empty set of tests counts as a failure
        if pytest -qq --collect-only -k "[${T}"; then
            PATTERNS+=("[${T}")
        fi
    done

    # actually run the tests in each pattern
    for P in "${PATTERNS[@]}"; do
        set -x
        pytest ${TEST_ARGS} -k "$P"
        RESULT=$?
        set +x
        if [ $RESULT -ne 0 ] ; then
            # we could try again on failure, but lets see if we can
            # keep things non-flakey enough that it isn't necessary
            break
        fi
    done
fi

if [ $RESULT -ne 0 ]; then
    kubectl get pods
    kubectl get svc

    if [ -n "${AMBASSADOR_DEV}" ]; then
        docker ps -a
    fi

    exit 1
fi

exit 0
