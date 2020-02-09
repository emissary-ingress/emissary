#!/usr/bin/env python3

import sys

import os
import urllib

import requests
from ambassador.utils import parse_bool

def usage(program):
    sys.stderr.write(f'Usage: {program} [--watt|--k8s|--fs] UPDATE_URL\n')
    sys.stderr.write('Notify `diagd` (and `amb-sidecar`, if AES) that a new WATT snapshot is available at UPDATE_URL.\n')
    sys.exit(1)


base_host = os.environ.get('DEV_AMBASSADOR_EVENT_HOST', 'http://localhost:8877')
base_path = os.environ.get('DEV_AMBASSADOR_EVENT_PATH', '_internal/v0')

sidecar_host = os.environ.get('DEV_AMBASSADOR_SIDECAR_HOST', 'http://localhost:8500')
sidecar_path = os.environ.get('DEV_AMBASSADOR_SIDECAR_PATH', '_internal/v0')

url_type = 'update'
arg_key = 'url'

program = os.path.basename(sys.argv[0])
args = sys.argv[1:]

while args and args[0].startswith('--'):
    arg = args.pop(0)

    if arg == '--k8s':
        # Already set up.
        pass
    elif arg == '--watt':
        url_type = 'watt'
    elif arg == '--fs':
        url_type = 'fs'
        arg_key = 'path'
    else:
        usage(program)

if len(args) != 1:
    usage(program)

urls = [ f'{base_host}/{base_path}/{url_type}' ]

if parse_bool(os.environ.get('EDGE_STACK', 'false')) or os.path.exists('/ambassador/.edge_stack'):
    urls.append(f'{sidecar_host}/{sidecar_path}/{url_type}')

exitcode = 0

for url in urls:
    r = requests.post(url, params={ arg_key: args[0] })

    if r.status_code != 200:
        sys.stderr.write("failed to update %s: %d: %s" % (r.url, r.status_code, r.text))
        exitcode = 1

sys.exit(exitcode)
