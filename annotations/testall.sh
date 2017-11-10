#!/bin/sh

set -ex

for dir in 0*; do
    sh $dir/test.sh
done
