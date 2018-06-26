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

set -e

mkdir_if_needed () {
    d="$1"

    if [ ! -d "$d" ]; then mkdir "$d"; fi
}

HERE=$(cd $(dirname $0); pwd)

cd "${HERE}/../templates"

ODIR="../docs/yaml"

mkdir_if_needed "$ODIR"

for tdir in *; do
    mkdir_if_needed "$ODIR/$tdir"

    for tfile in "$tdir"/*; do
        echo "---- $tfile"
        python "$HERE/template.py" < $tfile > "$ODIR/$tfile"
    done
done

# ADIR="$ODIR/ambassador"
# echo "---- synth ambassador.yaml"
# cat "$ADIR"/ambassador-{empty-certs,http,proxy}.yaml > "$ADIR/ambassador.yaml"
