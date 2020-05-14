#!/bin/bash

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
echo "AUTH: args $@"

pids=()

handle_chld() {
    local tmp=()

    for (( i=0; i<${#pids[@]}; ++i )); do
        split=(${pids[$i]//;/ })    # the space after the trailing / is critical!
        pid=${split[0]}
        name=${split[1]}

        if [ ! -d /proc/$pid ]; then
            wait $pid
            echo "AUTH: $name exited: $?"
            echo "AUTH: shutting down"
            exit 1
        else
            tmp+=(${pids[i]})
        fi
    done

    pids=(${tmp[@]})
}

set -o monitor
trap "handle_chld" CHLD

ROOT=$$

echo "AUTH: starting auth service"
/usr/bin/python3 "$APPDIR/auth.py" "$@" &
pids+=("$!;auth")

echo "AUTH: waiting"
wait

