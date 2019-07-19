#!bash

# Coverage checks are totally broken right now. I suspect that it's
# probably the result of all the Ambassador stuff actually happen in
# Docker containers. To restore it, first add
# 
# --cov=ambassador --cov=ambassador_diag --cov-report term-missing
#
# to the pytest line, and, uh, I guess recover and merge all the .coverage 
# files from the containers??

HERE=$(cd $(dirname $0); pwd)
ROOT=$(cd .. ; pwd)

echo "HERE: $HERE"
echo "ROOT: $ROOT"

set -e
set -o pipefail

if [[ "$USE_KUBERNAUT" != "true" ]]; then
    ( cd "$ROOT"; bash "$HERE/test-warn.sh" )
fi

TEST_ARGS="--tb=short"

seq=('Plain' 'not Plain')

if [[ -n "${TEST_NAME}" ]]; then
    seq=("$TEST_NAME")
fi

FULL_RESULT=0

for el in "${seq[@]}"; do
    echo "==== running $el"

    ( cd "$ROOT" ; make clean-test cluster-and-teleproxy )

    pytest ${TEST_ARGS} -k "$el"
    true
    RESULT=$?

    if [ $RESULT -ne 0 ]; then
        FULL_RESULT=1

        kubectl get pods --all-namespaces
        kubectl get svc --all-namespaces

        if [ -n "${AMBASSADOR_DEV}" ]; then
            docker ps -a
        fi

        for pod in $(kubectl get pods -o jsonpath='{range .items[?(@.status.phase != "Running")]}{.metadata.name}:{.status.phase}{"\n"}{end}'); do
            # WTFO.
            echo "==== logs for $pod"
            podname=$(echo $pod | cut -d: -f1)
            kubectl logs $podname
        done
    fi
done

exit $FULL_RESULT
