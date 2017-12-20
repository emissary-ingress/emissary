#!/bin/sh

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
