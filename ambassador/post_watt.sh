#!/bin/sh

ROOT=$(cd $(dirname $0); pwd)

AMBASSADOR_EVENT_URL=http://localhost:8877/_internal/v0/watt python3 $ROOT/post_update.py "$@"
