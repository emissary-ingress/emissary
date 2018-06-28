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

cat /dev/null > master.log

run_and_log () {
    if bash testone.sh "$1"; then
        echo "$1 PASS" >> master.log
    else
        echo "$1 FAIL" >> master.log
    fi
}

if [ -n "$E2E_TEST_NAME" ]; then
    run_and_log "$E2E_TEST_NAME"
else
    for dir in 0-serial/[0-9]*; do
        run_and_log "$dir"
    done

    for dir in 1-parallel/[0-9]*; do
        run_and_log "$dir" &
    done
endif

wait

failures=$(grep -c 'FAIL' master.log)

exit $failures
