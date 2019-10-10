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

printf "== Begin: travis-cleanup.sh ==\n"

#eval $(make export-vars)

echo "GIT_BRANCH ${GIT_BRANCH}"
echo "CI_DEBUG_KAT_BRANCH ${CI_DEBUG_KAT_BRANCH}"

teardown=yes

if [ \( -n "$CI_DEBUG_KAT_BRANCH" \) ]; then
	if [ "$GIT_BRANCH" = "$CI_DEBUG_KAT_BRANCH" ]; then
		echo "Leaving Kat cluster intact for debugging:"
		echo "===="
		kubectl cluster-info
        echo "==== DEV_KUBECONFIG ===="
	    gzip -9 < ${DEV_KUBECONFIG} | base64
		
		teardown=
	else
		echo "Not running on debug branch ${CI_DEBUG_KAT_BRANCH}"
	fi
else
	echo "No debug branch is set"
fi

if [ -n "$teardown" ]; then
    kubernaut claims delete ${CLAIM_NAME}
fi





