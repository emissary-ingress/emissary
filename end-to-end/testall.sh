#!/bin/sh

set -e

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

for dir in 0*; do
    echo "========"
    echo "${dir}..."
    echo

    if sh $dir/test.sh; then
        echo "${dir} PASSED"
    else
        echo "${dir} FAILED"
    fi
done
