#!/bin/sh

if ! pytest --tb=short --cov=ambassador --cov=ambassador_diag --cov-report term-missing  ${TEST_NAME}; then
    kubectl get pods
    kubectl get svc

    if [ -n "${AMBASSADOR_DEV}" ]; then
        docker ps -a
    fi

    exit 1
fi

exit 0
