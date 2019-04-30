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
fi

pytest ${TEST_ARGS}
RESULT=$?

if [ $RESULT -ne 0 ]; then
    kubectl get pods --all-namespaces
    kubectl get svc --all-namespaces

    if [ -n "${AMBASSADOR_DEV}" ]; then
        docker ps -a
    fi

    exit 1
fi

exit 0
