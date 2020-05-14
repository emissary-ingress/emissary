#!/bin/bash

# This script is functionally equivalent to invoking `docker build`
# directly, however it provides an enhanced UX.
#
# Building any given dockerfile can range from really fast (<1s if it
# is cached) to really really slow (e.g. 15 minutes) if entirely
# uncached. When the build is really fast, it can be tempting to use
# the -q (quiet) option for docker build in order to reduce noise. The
# trouble is, if you change something early in the dockerfile or you
# run the build on a new machine, that -q option suddently appears as
# the build hanging entirely since it will happily run for the full
# uncached build time (e.g. 15 minutes) generating no output. This
# script fixes this problem by behaving like docker build -q for any
# build that is faster than 3 seconds, but behaving like a normal
# docker build for any build that takes longer than one second.

# make sure we clean up after ourselves when we exit, this means
# killing the three subprocess we started and removing the two tmp
# files
cleanup() {
    kill $build_pid $sleep_pid $tail_pid 2> /dev/null
    wait $build_pid $sleep_pid $tail_pid 2> /dev/null
    rm -f $outfile $tmpiidfile
}
trap cleanup EXIT

# look to see if they passed us an --iidfile arg
for var in "$@"
do
    if [ -n "$found" ] && [ -z "$iidfile" ]; then
        iidfile="$var"
    fi
    case "$var" in
        "--iidfile")
            found=1
            ;;
        "--help")
            exec docker build "$@"
            ;;
    esac
done

outfile=$(mktemp /tmp/docker-build.XXXXXX)
if [ -z "$iidfile" ]; then
    tmpiidfile=$(mktemp /tmp/docker-build-iid.XXXXXX)
    extra_args="--iidfile $tmpiidfile"
    iidfile=$tmpiidfile
fi

# start docker and sleep in a race
docker build ${DBUILD_ARGS} $extra_args "$@" > $outfile 2>&1 &
build_pid=$!
sleep 3 &
sleep_pid=$!

# wait till one of them wins
if ((BASH_VERSINFO[0] < 4)); then
    until ! kill -s 0 $build_pid > /dev/null 2>&1 || ! kill -s 0 $sleep_pid > /dev/null 2>&1 ; do
        sleep 0.1
    done
else
    wait -n
fi

# if the build is still running lets tail the output
if kill -s 0 $build_pid > /dev/null 2>&1; then
    tail -f -n +0 $outfile &
    tail_pid=$!
fi

# wait for the build to finish
wait $build_pid
RESULT=$?

# if didn't tail the output we need to provide some output
if [ -z "$tailpid" ]; then

    # if the build was good, lets just display the image id, otherwise
    # if the build was bad, we need the full output
    if [ $RESULT == 0 ]; then
        echo $(cat $iidfile)
    else
        cat $outfile
    fi
fi

exit $RESULT
