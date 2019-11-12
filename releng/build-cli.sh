#!/usr/bin/env bash

# Copyright 2019 Datawire. All rights reserved.
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

binary_version="$(make version)"
release_version="$(./builder/builder.sh release-version)"

case "$1" in
    version)
        echo "$release_version"
        ;;
    build)
        if [ -n "$(builder/builder.sh builder)" ]; then
            # Copy binary from container
            docker cp "$(builder/builder.sh builder):/buildroot/bin/edgectl" ~/bin/edgectl
        else
            go build -trimpath -ldflags "-X main.Version=$binary_version" -o ~/bin/edgectl ./cmd/edgectl
        fi
        ;;
    push)
        # Push this OS/arch binary
        aws s3api put-object \
            --bucket datawire-static-files \
            --key "edgectl/$release_version/$(go env GOOS)/$(go env GOARCH)/edgectl" \
            --body ~/bin/edgectl
        ;;
    tag)
        # Update latest.txt
        pushtmp=$(mktemp -d)
        echo "$release_version" > "${pushtmp}/latest.txt"
        aws s3api put-object \
            --bucket datawire-static-files \
            --key edgectl/latest.txt \
            --body "${pushtmp}/latest.txt"
        rm -rf "$pushtmp"
        ;;
    promote)
        # Replace stable.txt with the contents of latest.txt
        pushtmp=$(mktemp -d)
        curl -s -o "${pushtmp}/stable.txt" https://s3.amazonaws.com/datawire-static-files/edgectl/latest.txt
        aws s3api put-object \
            --bucket datawire-static-files \
            --key edgectl/stable.txt \
            --body "${pushtmp}/stable.txt"
        rm -rf "$pushtmp"
        ;;
    *)
        echo "usage: $0 {version|build|push|tag|promote}"
        exit 1
        ;;
esac
