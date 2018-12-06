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

DRAIN_TIME=${AMBASSADOR_DRAIN_TIME:-5}
SHUTDOWN_TIME=${AMBASSADOR_SHUTDOWN_TIME:-10}
AMBASSADOR_ROOT="/ambassador"
CONFIG_DIR="${AMBASSADOR_CONFIG_BASE_DIR:-$AMBASSADOR_ROOT}"

LATEST=$(ls -1v "$CONFIG_DIR"/envoy*.json | tail -1)
exec /usr/local/bin/envoy -c ${LATEST} --restart-epoch $RESTART_EPOCH --drain-time-s "${DRAIN_TIME}" --service-cluster "${AMBASSADOR_ID:-ambassador}-${AMBASSADOR_NAMESPACE}" --parent-shutdown-time-s "${SHUTDOWN_TIME}"
