#!/usr/bin/env bash

set -e
set -o pipefail

HERE=$(cd $(dirname $0); pwd)
BUILD_ALL=${BUILD_ALL:-false}

cd "$HERE"
source "$HERE/kubernaut_utils.sh"

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

cat /dev/null > master.log

run_and_log () {
    if bash testone.sh "$1"; then
        echo "$1 PASS" >> master.log
    else
        echo "$1 FAIL" >> master.log
    fi
}

for dir in 0-serial/[0-9]*; do
    run_and_log "$dir"
done

for dir in 1-parallel/[0-9]*; do
    run_and_log "$dir" &
done

wait

failures=$(grep -c 'FAIL' master.log)

exit $failures
