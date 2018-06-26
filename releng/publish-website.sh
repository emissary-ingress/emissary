#!/usr/bin/env bash

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

set -o errexit
set -o nounset

RELEASE_TYPE=${RELEASE_TYPE:?RELEASE_TYPE not set or empty}

NETLIFY_TOKEN=${NETLIFY_TOKEN:?NETLIFY_TOKEN not set or empty}
NETLIFY_SITE=${NETLIFY_SITE:?NETLIFY_SITE not set or empty}
NETLIFY_OPTS=${NETLIFY_OPTS:-"--draft"}
if [[ "$RELEASE_TYPE" == "stable" ]]; then
    NETLIFY_OPTS=
fi

docs/node_modules/.bin/netlify \
	--access-token ${NETLIFY_TOKEN} \
	deploy \
	${NETLIFY_OPTS} \
	--path docs/_book \
	--site-id ${NETLIFY_SITE}
