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

# THE DEFAULT BOOT SEQUENCE IS NOW entrypoint.go. HOWEVER, we'll stick
# with entrypoint.sh when the --dev-magic parameter is present. This
# is currently used only for test_scout.py.  This is a BRUTAL HACK.

if [ "$1" != "--dev-magic" ]; then
  echo "Running entrypoint"
  if [ -n "$AMBASSADOR_LOGFILE" ]; then
    exec busyambassador entrypoint "$@" >/tmp/access.log 2>/tmp/entry.log  # See comment above.
  else
    exec busyambassador entrypoint "$@"  # See comment above
  fi
fi

DEVMAGIC=yes

# If we are here, define AMBASSADOR_FAST_RECONFIGURE, to make
# _absolutely certain_ that diagd's localhost checks are in sync with
# what's actually running...
export AMBASSADOR_FAST_RECONFIGURE=false

ENTRYPOINT_DEBUG=

log () {
    local now

    now=$(date +"%Y-%m-%d %H:%M:%S")
    echo "${now} AMBASSADOR INFO ${@}" >&2
}

debug () {
    local now

    if [ -n "$ENTRYPOINT_DEBUG" ]; then
        now=$(date +"%Y-%m-%d %H:%M:%S")
        echo "${now} AMBASSADOR DEBUG ${@}" >&2
    fi
}

wait_for_url () {
    local name url tries_left delay status

    name="$1"
    url="$2"

    tries_left=10
    delay=1

    while (( tries_left > 0 )); do
        debug "pinging $name ($tries_left)..."

        status=$(curl -s -o /dev/null -w "%{http_code}" $url)

        if [ "$status" = "200" ]; then
            break
        fi

        tries_left=$(( tries_left - 1 ))
        sleep $delay
        delay=$(( delay * 2 ))
        if (( delay > 10 )); then delay=5; fi
    done

    if (( tries_left <= 0 )); then
        log "giving up on $name and hoping for the best..."
    else
        log "$name running"
    fi
}

################################################################################
# CONFIG PARSING                                                               #
################################################################################

ambassador_root="/ambassador"

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

# If we have an AGENT_SERVICE, but no AMBASSADOR_ID, force AMBASSADOR_ID
# from the AGENT_SERVICE.

if [ -z "$AMBASSADOR_ID" -a -n "$AGENT_SERVICE" ]; then
    export AMBASSADOR_ID="intercept-${AGENT_SERVICE}"
    log "Intercept: set AMBASSADOR_ID to $AMBASSADOR_ID"
fi

export AMBASSADOR_NAMESPACE="${AMBASSADOR_NAMESPACE:-default}"
export AMBASSADOR_CONFIG_BASE_DIR="${AMBASSADOR_CONFIG_BASE_DIR:-$ambassador_root}"
export ENVOY_DIR="${AMBASSADOR_CONFIG_BASE_DIR}/envoy"
export ENVOY_BOOTSTRAP_FILE="${AMBASSADOR_CONFIG_BASE_DIR}/bootstrap-ads.json"
export ENVOY_BASE_ID="${AMBASSADOR_ENVOY_BASE_ID:-0}"

export APPDIR="${APPDIR:-$ambassador_root}"

# If we don't set PYTHON_EGG_CACHE explicitly, /.cache is set by
# default, which fails when running as a non-privileged user
export PYTHON_EGG_CACHE="${PYTHON_EGG_CACHE:-$AMBASSADOR_CONFIG_BASE_DIR}/.cache"
export PYTHONUNBUFFERED=true

log "running with dev magic"
diagd --dev-magic
