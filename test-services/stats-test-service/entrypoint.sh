#!/bin/sh

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

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

APPDIR=${APPDIR:-/application}

env | grep V
echo "STATS-TEST: args $@"

pids=""

diediedie() {
    NAME=$1
    STATUS=$2

    if [ $STATUS -eq 0 ]; then
        echo "STATS-TEST: $NAME claimed success, but exited \?\?\?\?"
    else
        echo "STATS-TEST: $NAME exited with status $STATUS"
    fi

    ambassador_exit 1
}

handle_chld() {
    trap - CHLD
    local tmp
    for entry in $pids; do
        local pid="${entry%:*}"
        local name="${entry#*:}"
        if [ ! -d "/proc/${pid}" ]; then
            wait "${pid}"
            STATUS=$?
            diediedie "${name}" "$STATUS"
        else
            tmp="${tmp:+${tmp} }${entry}"
        fi
    done

    pids="$tmp"
    trap "handle_chld" CHLD
}

handle_int() {
    echo "Exiting due to Control-C"
}

# set -o monitor
trap "handle_chld" CHLD
trap "handle_int" INT

ROOT=$$

echo "STATS-TEST: starting stats-test service"
/usr/bin/python3 "$APPDIR/stats-test.py" "$@" &
TEST_PID=$!
pids="${pids:+${pids} }${TEST_PID}:stats-test"

echo "STATS-TEST: starting stats-web service"
/usr/bin/python3 "$APPDIR/stats-web.py" &
WEB_PID=$!
pids="${pids:+${pids} }${WEB_PID}:stats-web"

echo "STATS-TEST: waiting"
wait

