#!/bin/sh

./teleproxy -dns 10.0.0.1 -mode intercept > /tmp/teleproxy.log 2>&1 &
BACKEND=tzone ./kat-server > /tmp/backend.log 2>&1 &
