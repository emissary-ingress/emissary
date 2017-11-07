#!/usr/bin/env python

import sys

import json
import dpath.util

x = json.load(sys.stdin)

clusters = {}

for route in x.get('routes', []):
    if route['prefix'] == '/demo/':
        for cluster in route['clusters']:
            clusters[cluster['name']] = cluster['weight']

x = [ int(clusters[name]) for name in sorted(clusters.keys()) ]

print("weights: %s" % x)

wanted = list(map(int, sys.argv[1:]))

# print("wanted:  %s" % wanted)

if x != wanted:
    sys.exit(1)
else:
    sys.exit(0)
