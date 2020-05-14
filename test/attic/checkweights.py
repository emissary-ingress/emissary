#!/usr/bin/env python

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

import sys

import json
import requests

# Yes, it's a terrible idea to use skip cert verification for TLS.
# We really don't care for this test though.
import urllib3
urllib3.disable_warnings()

import dpath.util

headers = []
wanted = []

base_url = sys.argv[1]

i = 2
while i < len(sys.argv):
    arg = sys.argv[i]

    if '=' in arg:
        header, value = arg.split('=')
        headers.append((header, value))
    else:
        wanted.append(int(arg))

    i += 1

headers.sort()

# print("Headers: %s" % headers)
# print("Wanted:  %s" % wanted)

r = requests.get("%s/ambassador/v0/diag/?json=true" % base_url, verify=False)

if r.status_code != 200:
    print("couldn't load diagnostics: %d" % r.status_code)
    sys.exit(1)

x = r.json()

clusters = {}

for route in x.get('routes', []):
    if route['prefix'] == '/demo/':
        route_headers = [ (hdr['name'], hdr['value']) 
                          for hdr in route.get('headers', []) ]
        route_headers.sort()

        if headers != route_headers:
            # print("missed route_headers %s" % route_headers)
            continue

        # print("hit route_headers %s" % route_headers)

        for cluster in route['clusters']:
            clusters[cluster['name']] = cluster['weight']

x = [ int(clusters[name]) for name in sorted(clusters.keys()) ]

# print("weights: %s" % x)
# print("wanted:  %s" % wanted)

if x != wanted:
    sys.exit(1)
else:
    sys.exit(0)
