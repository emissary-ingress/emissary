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

set -e
set -o pipefail
set -x

# We only want to pull images if they are not present locally. This impacts local test runs.
if [[ "$(docker images -q $AMBASSADOR_DOCKER_IMAGE 2> /dev/null)" == "" ]]; then
    if ! docker pull $AMBASSADOR_DOCKER_IMAGE; then
        echo "could not pull $AMBASSADOR_DOCKER_IMAGE" >&2
        exit 1
    fi
fi

if [[ "$USE_KUBERNAUT" != "true" ]]; then
    ( cd "$ROOT"; bash "$HERE/test-warn.sh" )
fi

TEST_ARGS="--tb=short -s"

seq=(
    'Plain'
    'not Plain and (A or C)'
    'not Plain and not (A or C)'
)

if [[ -n "${TEST_NAME}" ]]; then
    case "${TEST_NAME}" in
    group1) seq=('Plain') ;;
    group2) seq=('not Plain and (A or C)') ;;
    group3) seq=('not Plain and not (A or C)') ;;
    *) seq=("$TEST_NAME") ;;
    esac
fi

( cd "$ROOT" ; make cluster-and-teleproxy )

echo "==== [$(date)] ==== STARTING TESTS"

failed=()

for el in "${seq[@]}"; do
    echo "==== [$(date)] ==== running $el"

#    kubectl delete namespaces -l scope=AmbassadorTest
#    kubectl delete all -l scope=AmbassadorTest

    if ! pytest ${TEST_ARGS} -k "$el"; then
        failed+=("$el")

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

    if [ -f /tmp/k8s-AmbassadorTest ]; then
        mv /tmp/k8s-AmbassadorTest.yaml /tmp/k8s-$el-AmbassadorTest.yaml
    fi
done

if (( ${#failed[@]} == 0 )); then
    echo "==== [$(date)] ==== FINISHED TESTS (passed)"
    exit 0
else
    echo "==== [$(date)] ==== FINISHED TESTS (failed: ${failed[*]})"
    exit 1
fi

