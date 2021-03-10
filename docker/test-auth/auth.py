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

# import functools
import json
import logging
import os
import pprint

from flask import Flask, Response, jsonify, request

__version__ = '0.0.2'

app = Flask(__name__)

@app.before_request
def before():
    print("---- Incoming Request Headers")
    pprint.pprint(request)

    for header in sorted(request.headers.keys()):
        print("%s: %s" % (header, request.headers[header]))

    print("----")

@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def catch_all(path):
    if not path.startswith('/'):
        path = '/' + path

    if path.startswith('//'):
        path = path[1:]

    status_code = 403
    headers = { 'X-Test': 'Should not be seen.' }

    if path.startswith('/ambassador/'):
        status_code = 200
    elif path.endswith("/good/") or path.endswith("/demo/"):
        status_code = 200
        headers['X-Auth-Route'] = 'Route'
    elif path.endswith("/nohdr/"):
        status_code = 200
        # Don't add the header.

    backend_name = os.environ.get('BACKEND')

    body = 'You want path: %s' % path
    mimetype = 'text/plain'

    if backend_name:
        body_dict = {
            'backend': backend_name,
            'path': path
        }

        body = json.dumps(body_dict)
        mimetype = 'application/json'

        extauth_dict = {
            'request': {
                'url': request.url,
                'path': path,
                'headers': dict(request.headers)
            },
            'resp_headers': headers
        }

        headers['extauth'] = extauth_dict

    return Response(body,
                    mimetype=mimetype,
                    status=status_code,
                    headers=headers)

if __name__ == "__main__":
    ssl_context = None
    conn_type = "HTTP"
    port = 80

    if (len(sys.argv) > 1) and (sys.argv[1] == "--tls"):
        ssl_context = ('authsvc.crt', 'authsvc.key')
        conn_type = "HTTPS"
        port = 443

    if os.environ.get('PORT'):
        port = int(os.environ['PORT'])

    logging.basicConfig(
        # filename=logPath,
        level=logging.DEBUG, # if appDebug else logging.INFO,
        format="%%(asctime)s auth %s %s %%(levelname)s: %%(message)s" % (__version__, conn_type),
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    logging.info("initializing")
    app.run(host='0.0.0.0', port=port, debug=True, ssl_context=ssl_context)
