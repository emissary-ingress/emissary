#!/bin/bash

# For linify
export MACHINE_READABLE=yes
export SKIP_CHECK_CONTEXT=yes

CLEAN_ON_SUCCESS=

if [ "$1" == "--cleanup" ]; then
    CLEAN_ON_SUCCESS="--cleanup"
    shift
fi

DIR="$1"
LOG="$(basename $DIR).log"

attempt=0
dir_passed=

while [ $attempt -lt 2 ]; do
    echo
    echo "================================================================"
    echo "${attempt}: $DIR..."

    attempt=$(( $attempt + 1 ))

    bash $DIR/test.sh $CLEAN_ON_SUCCESS 2>&1 | python linify.py $LOG

    # I hate shell sometimes.
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo "$DIR PASSED"
        dir_passed=yes
        break
    else
        echo "================ k8s info" >> $LOG
        kubectl get svc --all-namespaces >> $LOG
        kubectl get pods --all-namespaces >> $LOG

        fail_log=$(basename $DIR)-fail-${attempt}.log
        mv $LOG $fail_log

        echo "$DIR FAILED; output in $fail_log"
    fi
done

if [ -z "$dir_passed" ]; then
    exit 1
else
    exit 0
fi
