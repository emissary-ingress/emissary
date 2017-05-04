#!/bin/sh

set -ex

env | sort 

TYPE=$(python scripts/bumptype.py)

echo "would make new-$TYPE"

exit 1
