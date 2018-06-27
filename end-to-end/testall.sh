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
BUILD_ALL=${BUILD_ALL:-false}

cd "$HERE"
source "$HERE/kubernaut_utils.sh"
source "$HERE/forge_utils.sh"

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

# For linify
export MACHINE_READABLE=yes
export SKIP_CHECK_CONTEXT=yes

failures=0

for dir in 0*; do
    attempt=0
    dir_passed=

    while [ $attempt -lt 2 ]; do
        echo
        echo "================================================================"
        echo "${attempt}: ${dir}..."

        attempt=$(( $attempt + 1 ))

        if bash $dir/test.sh 2>&1 | python linify.py test.log; then
            echo "${dir} PASSED"
            dir_passed=yes
            break
        else
            echo "${dir} FAILED"

            echo "================ k8s info"
            kubectl get svc --all-namespaces
            kubectl get pods --all-namespaces
            echo "================ start captured output"
            cat test.log
            echo "================ end captured output"
        fi
    done

    if [ -z "$dir_passed" ]; then
        failures=$(( $failures + 1 ))
    fi
done

exit $failures
