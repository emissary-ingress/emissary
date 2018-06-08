#!/bin/bash

# For linify
export MACHINE_READABLE=yes
export SKIP_CHECK_CONTEXT=yes

DIR="$1"
LOG="$(basename $DIR).log"

attempt=0
dir_passed=

while [ $attempt -lt 2 ]; do
    echo
    echo "================================================================"
    echo "${attempt}: $DIR..."

    attempt=$(( $attempt + 1 ))

    if bash $DIR/test.sh 2>&1 | python linify.py $LOG; then
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
