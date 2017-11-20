#!/usr/bin/env python

import sys

import json
import dpath.util

x = json.load(sys.stdin)
y = None

try:
    y = dpath.util.get(x, sys.argv[1])
    print(json.dumps(y, sort_keys=True, indent=4))
    sys.exit(0)
except KeyError:
    sys.exit(1)

