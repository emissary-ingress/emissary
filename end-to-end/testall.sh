#!/usr/bin/env bash

set -e
set -o pipefail

HERE=$(cd $(dirname $0); pwd)
BUILD_ALL=${BUILD_ALL:-false}

cd "$HERE"
source "$HERE/kubernaut_utils.sh"

if [ "$BUILD_ALL" = true ]; then
  sh buildall.sh
fi

if [ -z "$SKIP_KUBERNAUT" ]; then
    get_kubernaut_cluster
fi

# For linify
export MACHINE_READABLE=yes
export SKIP_CHECK_CONTEXT=yes

for dir in 0*; do
    attempt=0

    while [ $attempt -lt 2 ]; do
        echo
        echo "================================================================"
        echo "${attempt}: ${dir}..."

        attempt=$(( $attempt + 1 ))

        if bash $dir/test.sh 2>&1 | python linify.py test.log; then
            echo "${dir} PASSED"
            break
        else
            echo "${dir} FAILED"

            echo "================ start captured output"
            cat test.log
            echo "================ end captured output"
        fi
    done
done
