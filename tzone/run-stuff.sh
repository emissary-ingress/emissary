#!/bin/sh

./teleproxy -mode intercept > /tmp/teleproxy.log 2>&1 &
BACKEND=tzone ./kat-server > /tmp/backend.log 2>&1 &
