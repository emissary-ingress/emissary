#!/bin/sh

set -ex

env | sort 

python --version
python3 --version

TYPE=$(python3 scripts/bumptype.py)

echo "would make new-$TYPE"

exit 1
