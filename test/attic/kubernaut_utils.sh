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

if [ -n "$HERE" ]; then
    HERE=$(pwd)
fi

KUBERNAUT="$HERE/kubernaut"

retry () {
    label="$1"
    iteration_function="$2"

    attempts=${3:-3}
    delay=${4:-20}
    succeeded=

    while [ $attempts -gt 0 ]; do
        echo "$attempts: $label"
        attempts=$(( $attempts - 1 ))

        if $iteration_function; then
            succeeded=yes
            break
        fi

        sleep $delay
    done

    if [ -n "$succeeded" ]; then
        return 0
    else
        return 1
    fi
}

_get_kubernaut_iteration () {
    kubernaut_version=$(curl -f -s https://s3.amazonaws.com/datawire-static-files/kubernaut/stable.txt)
    if [ $? -ne 0 ]; then return 1; fi

    curl -f -s -L -o "$KUBERNAUT" https://s3.amazonaws.com/datawire-static-files/kubernaut/$kubernaut_version/kubernaut
    # curl -s -L -o "$KUBERNAUT" https://s3.amazonaws.com/datawire-static-files/kubernaut/0.1.39/kubernaut
    if [ $? -ne 0 ]; then return 1; fi

    chmod +x "$KUBERNAUT"
    return 0
}

get_kubernaut () {
    if [ ! -x "$KUBERNAUT" ]; then
        retry "Fetching kubernaut" _get_kubernaut_iteration 3 5

        if [ ! -x "$KUBERNAUT" ]; then
            return 1
        fi
    fi

    return 0
}

check_kubernaut_token () {
    if [ $("$KUBERNAUT" kubeconfig | grep -c 'Token not found') -gt 0 ]; then
        echo "You need a Kubernaut token. Go to"
        echo ""
        echo "https://kubernaut.io/token"
        echo ""
        echo "to get one, then run"
        echo ""
        echo "sh $ROOT/save-token.sh \"\$token\""
        echo ""
        echo "to save it before trying again."

        return 1
    fi

    return 0
}

_get_kubernaut_cluster_iteration () {
    echo "Dropping old cluster"
    "$KUBERNAUT" discard || return 1

    echo "Claiming new cluster"
    "$KUBERNAUT" claim || return 1

    return 0    
}

get_kubernaut_cluster () {
    get_kubernaut || return 1
    check_kubernaut_token || return 1

    if retry "Aquiring kubernaut cluster" _get_kubernaut_cluster_iteration 10 30; then
        export KUBECONFIG=${HOME}/.kube/kubernaut
        return 0
    else
        echo "Could not acquire kubernaut cluster" >&2
        return 1
    fi
}
