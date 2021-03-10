#!/usr/bin/env bash

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
#set -x

imgvars=(
    AMBASSADOR_DOCKER_IMAGE
    KAT_SERVER_DOCKER_IMAGE
    KAT_CLIENT_DOCKER_IMAGE
    TEST_SERVICE_AUTH
    TEST_SERVICE_AUTH_TLS
    TEST_SERVICE_RATELIMIT
    TEST_SERVICE_SHADOW
    TEST_SERVICE_STATS
)
for varname in "${imgvars[@]}"; do
    # We only want to pull images if they are not present locally. This impacts local test runs.
    echo "==== [$(date)] ==== Verifying \$${varname}..."
    varval=${!varname}
    if [[ -n "${varval}" ]]; then
        echo "set to '${varval}' in the environment" >&2
    else
        filebase=${varname}
        filebase=${filebase%_DOCKER_IMAGE}
        filebase=${filebase//_/-}
        filebase=${filebase,,}
        filebase=${filebase/#test-service-/test-}
        filename=${ROOT}/${filebase}.docker.push.dev
        if [[ -f "$filename" ]]; then
            varval="$(sed -n 2p "$filename")"
            if [[ -n "${varval}" ]]; then
                eval "export $varname=$varval"
                echo "set to '${varval}' in '${filename}'" >&2
            fi
        fi
    fi
    if [[ -z "${varval}" ]]; then
        echo "variable '${varname}' is not set" >&2
        exit 1
    fi
    if ! docker run --rm --entrypoint=true "$varval"; then
        echo "could not pull $varval" >&2
        exit 1
    fi
done

( cd "$ROOT"; bash "$HERE/test-warn.sh" )

TEST_ARGS_GENERIC=(--tb=short -s --suppress-no-test-exit-code)
TEST_ARGS_WITHOUT_KNATIVE=("${TEST_ARGS_GENERIC[@]}" -k 'not Knative')
TEST_ARGS_WITH_KNATIVE=("${TEST_ARGS_GENERIC[@]}" -k 'Knative')

# We serialize Knative tests because they are resource intensive - so we don't want any interference with the other tests.
SERIALIZE_KNATIVE_TESTS=true

seq=("")

if [[ -n "${TEST_NAME}" ]]; then
    case "${TEST_NAME}" in
    group0) seq=('Plain' 'not Plain and (A or C)' 'not Plain and not (A or C)') ;;
    group1) seq=('Plain') ;;
    group2) seq=('not Plain and (A or C)') ;;
    group3) seq=('not Plain and not (A or C)') ;;
    *) seq=("$TEST_NAME"); SERIALIZE_KNATIVE_TESTS=false;;
    esac
fi

if [[ "$SERIALIZE_KNATIVE_TESTS" = true ]] ; then
        TEST_ARGS=("${TEST_ARGS_WITHOUT_KNATIVE[@]}")
    else
        TEST_ARGS=("${TEST_ARGS_GENERIC[@]}")
fi

# run_test() runs a given test. Takes 2 arguments -
# $1 is the test name. If blank, runs all tests.
# $2 is the test args you need to pass to the tests.
run_test() {
    test_name=$1
    shift
    test_arg=("$@")

    pretty_test_name="${test_name:-ALL}"

    echo "==== [$(date)] $pretty_test_name ==== running"

    k_args=""

    if [[ -n "$test_name" ]]; then
        k_args="-k $test_name"
    fi

    set +e
    set -x

    outdirbase="kat-log-${pretty_test_name}"
    outdir="/tmp/${outdirbase}"
    tmpdir="/tmp/kat-tmplog-${pretty_test_name}"

    rm -rf "$tmpdir"; mkdir "$tmpdir"

    if ! pytest "${test_arg[@]}" "${k_args}" | tee /tmp/pytest.log; then
        echo "==== [$(date)] $pretty_test_name ==== FAILED"

        failed+=("$test_name")

        mv /tmp/pytest.log "$tmpdir"

        kubectl get pods --all-namespaces > "$tmpdir/pods.txt" 2>&1
        kubectl get svc --all-namespaces > "$tmpdir/svc.txt" 2>&1
        kubectl logs -n kube-system -l k8s-app=kube-proxy > "$tmpdir/kube-proxy.txt" 2>&1

        if [ -n "${AMBASSADOR_DEV}" ]; then
            docker ps -a > "$tmpdir/docker.txt" 2>&1
        fi

        for pod in $(kubectl get pods -o jsonpath='{range .items[?(@.status.phase != "Running")]}{.metadata.name}:{.status.phase}{"\n"}{end}'); do
            # WTFO.
            podname=$(echo $pod | cut -d: -f1)
            kubectl logs $podname > "$tmpdir/pod-$podname.log" 2>&1
        done
    else
        echo "==== [$(date)] $pretty_test_name ==== SUCCEEDED"
    fi

    set -e
    set +x
}


( cd "$ROOT" ; ${MAKE:-make} setup-test )

echo "==== [$(date)] ==== STARTING TESTS"

failed=()

for t_name in "${seq[@]}"; do
    run_test "${t_name}" "${TEST_ARGS_WITHOUT_KNATIVE[@]}"
done

if [[ "$SERIALIZE_KNATIVE_TESTS" = true ]] ; then
    echo "==== Running Knative tests ===="
    all_tests=("")
    run_test "${all_tests}" "${TEST_ARGS_WITH_KNATIVE[@]}"
fi

if (( ${#failed[@]} == 0 )); then
    echo "==== [$(date)] ==== FINISHED TESTS (passed)"
    exit 0
else
    echo "==== [$(date)] ==== FINISHED TESTS (failed: ${failed[*]})"

    echo "==== [$(date)] ==== Collecting log tarball"

    if [ -f /tmp/k8s-AmbassadorTest ]; then
        mv /tmp/k8s-AmbassadorTest.yaml "$tmpdir"
    fi

    copy_if_present () {
        pattern="$1"
        dest="$2"
        expanded="$(echo $pattern)"

        if [ "$expanded" != "$pattern" ]; then
            cp $pattern $dest
        fi
    }

    copy_if_present '/tmp/kat-logs-*' "$tmpdir"
    copy_if_present '/tmp/kat-events-*' "$tmpdir"
    copy_if_present '/tmp/kat-client*' "$tmpdir"

    copy_if_present '/tmp/teleproxy.log' "$tmpdir"
    copy_if_present '/etc/resolv.conf' "$tmpdir"

    mv "$tmpdir" "$outdir"

    ( cd /tmp; tar czf "$outdir.tgz" "$outdirbase" )

    if [ -n "$AWS_ACCESS_KEY_ID" -a "$TRAVIS" = true ]; then
        now=$(date +"%y-%m-%dT%H:%M:%S")
        branch="$(printf ${GIT_BRANCH} | tr '[:upper:]' '[:lower:]' | sed -e 's/[^a-zA-Z0-9]/-/g' -e 's/-\{2,\}/-/g')"
        aws_key="kat-${branch}-${now}-${hr_el}-logs.tgz"

        echo "==== [$(date)] ==== Uploading log tarball as $aws_key"

        aws s3api put-object \
            --bucket datawire-static-files \
            --key kat/${aws_key} \
            --body "${outdir}.tgz"

        echo "==== [$(date)] ==== Recover log tarball with"
        echo "aws s3api get-object --bucket datawire-static-files --key kat/${aws_key} ${outdirbase}.tgz"
    else
        echo "==== [$(date)] ==== Upload to S3 not configured; leaving $outdir.tgz in place"
    fi

    exit 1
fi

