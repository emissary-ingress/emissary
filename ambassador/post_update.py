import sys

import os
import urllib

import requests

base_url = os.environ.get('AMBASSADOR_EVENT_URL', 'http://localhost:8877/_internal/v0/update')

if len(sys.argv) < 2:
    sys.stderr.write("Usage: %s update-url\n" % os.path.basename(sys.argv[0]))
    sys.exit(1)

r = requests.post(base_url, params={ 'url': sys.argv[1] })

if r.status_code != 200:
    sys.stderr.write("update to %s failed:\nstatus %d: %s" % (r.url, r.status_code, r.text))
    sys.exit(1)
else:
    sys.exit(0)
