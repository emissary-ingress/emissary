#!/bin/sh

set -e

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

sh buildall.sh

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
