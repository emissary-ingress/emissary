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

import logging
import os

from flask import Flask, Response, jsonify, request

__version__ = "?.?.?"

app = Flask(__name__)

@app.before_request
def before():
    logging.debug("=>> %s" % request)

    for header in sorted(request.headers.keys()):
        logging.debug("=>>   %s: %s" % (header, request.headers[header]))

@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def demo(path):
    # Return the version both in the body and in the X-Demo-Version
    # header (returning as a header allows us to use this service as
    # an extauth filter that allows everything, but still identifies
    # itself). We also use the X-QOTM-Session header, since our QOTM
    # service knows how to pass that all the way to the client.

    if not path.startswith('/'):
        path = '/' + path

    logging.debug("desired path: %s" % path)

    resp = Response("VERSION %s" % __version__)
    resp.headers['X-Demo-Version'] = __version__
    resp.headers['X-QOTM-Session'] = __version__
    return resp

if __name__ == "__main__":
    __version__ = sys.argv[1]

    ssl_context = None
    conn_type = "HTTP"

    if (len(sys.argv) > 2) and (sys.argv[2] == "--tls"):
        ssl_context = ('demosvc.crt', 'demosvc.key')
        conn_type = "HTTPS"

    logging.basicConfig(
        # filename=logPath,
        level=logging.DEBUG, # if appDebug else logging.INFO,
        format="%%(asctime)s demo %s %s %%(levelname)s: %%(message)s" % (__version__, conn_type),
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    logging.info("initializing")
    app.run(host='0.0.0.0', port=3000, debug=True, ssl_context=ssl_context)
