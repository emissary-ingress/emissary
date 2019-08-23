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
#set -x

# We only want to pull images if they are not present locally. This impacts local test runs.
echo "==== [$(date)] ==== Verifying $AMBASSADOR_DOCKER_IMAGE..."

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

seq=("")

if [[ -n "${TEST_NAME}" ]]; then
    case "${TEST_NAME}" in
    group0) seq=('Plain' 'not Plain and (A or C)' 'not Plain and not (A or C)') ;;
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
    hr_el="${el:-ALL}"

    echo "==== [$(date)] $hr_el ==== running"

#    kubectl delete namespaces -l scope=AmbassadorTest
#    kubectl delete all -l scope=AmbassadorTest

    k_args=""

    if [ -n "$el" ]; then
        k_args="-k $el"
    fi

    set +e
    set -x

    outdirbase="kat-log-${hr_el}"
    outdir="/tmp/${outdirbase}"
    tmpdir="/tmp/kat-tmplog-${hr_el}"

    rm -rf "$tmpdir"; mkdir "$tmpdir"

    if ! pytest ${TEST_ARGS} $k_args | tee /tmp/pytest.log; then
        echo "==== [$(date)] $hr_el ==== FAILED"

        failed+=("$el")

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
        echo "==== [$(date)] $hr_el ==== SUCCEEDED"
    fi

    set -e
    set +x
done

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

    cp /tmp/teleproxy.log "$tmpdir"
    cp /etc/resolv.conf "$tmpdir"

    mv "$tmpdir" "$outdir"

    ( cd /tmp; tar czf "$outdir.tgz" "$outdirbase" )

    if [ -n "$AWS_ACCESS_KEY_ID" -a -n "$GIT_BRANCH_SANITIZED" ]; then
        now=$(date +"%y-%m-%dT%H:%M:%S")
        branch=${GIT_BRANCH_SANITIZED:-localdev}
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

