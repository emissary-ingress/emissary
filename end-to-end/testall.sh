#!/bin/sh

set -e
set -o pipefail

HERE=$(cd $(dirname $0); pwd)
BUILD_ALL=${BUILD_ALL:-false}

cd "$HERE"

if [ "$BUILD_ALL" = true ]; then
  sh buildall.sh
fi

# For linify
export MACHINE_READABLE=yes

for dir in 0*; do
    echo
    echo "================================================================"
    echo "${dir}..."

    if sh $dir/test.sh 2>&1 | python linify.py test.log; then
        echo "${dir} PASSED"
    else
        echo "${dir} FAILED"

        echo "================ start captured output"
        cat test.log
        echo "================ end captured output"
    fi
done
