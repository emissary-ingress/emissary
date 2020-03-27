#!/usr/bin/env bash

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

set -e
set -o pipefail

HERE=$(cd $(dirname $0); pwd)
ROOT=$HERE
BUILD_ALL=${BUILD_ALL:-false}
PARALLEL_TESTS=7

cd "$HERE"
source "$HERE/kubernaut_utils.sh"
source "$HERE/forge_utils.sh"
source "$HERE/utils.sh"

if [ "$BUILD_ALL" = true ]; then
  bash buildall.sh
fi

if [ -z "$SKIP_KUBERNAUT" ]; then
    get_kubernaut_cluster
else
    echo "WARNING: your current kubernetes context will be WIPED OUT"
    echo "by this test. Current context:"
    echo ""
    kubectl config current-context
    echo ""

    while true; do
        read -p 'Is this really OK? (y/N) ' yn

        case $yn in
            [Yy]* ) break;;
            [Nn]* ) exit 1;;
            * ) echo "Please answer yes or no.";;
        esac
    done
fi

get_forge

check_rbac

rm -f *.log
cat /dev/null > master.log
> failures.txt

run_and_log () {
    test=$1
    if bash testone.sh --cleanup "$test"; then
        echo "$1 PASS" >> master.log
    else
        echo "$1 FAIL" >> master.log
        echo ${test} >> failures.txt
    fi
}

if [ -n "$E2E_TEST_NAME" ]; then
    if [ ! -d "$E2E_TEST_NAME" ]; then
        if [ -d "1-parallel/$E2E_TEST_NAME" ]; then
            E2E_TEST_NAME="1-parallel/$E2E_TEST_NAME"
        else
            echo "Test $E2E_TEST_NAME cannot be found" >&2
            exit 1
        fi
    fi

    run_and_log "$E2E_TEST_NAME"
else
    run_and_log "1-parallel/no-base-serial"

    # Clean up everything, non-interactively.
    SKIP_CHECK_CONTEXT=yes initialize_cluster

    export -f run_and_log
    echo 1-parallel/[0-9]* | xargs -n1 | xargs -P ${PARALLEL_TESTS} -I {} bash -c 'run_and_log {}'
fi

wait

echo
echo "The following tests failed:"
cat failures.txt

cp failures.txt old-failures.txt
# Empty the file for re-runs
> failures.txt
while read -r line || [[ -n "$line" ]]; do
    echo
    echo "Re-running $line"
    run_and_log ${line}
done < old-failures.txt

for f in *-fail-*.log; do
    if [ -f ${f} ]; then
        echo "=========================================="
        echo "Error output from $f"
        echo "=========================================="
        cat ${f}
        echo
        # We need this for Travis to not shut us down
        sleep 2
    fi
done

fail_count=$(wc -l < failures.txt)
echo "The following ${fail_count} tests failed:"
cat failures.txt
exit ${fail_count}
