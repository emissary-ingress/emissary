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

counts = {}

@app.before_request
def before():
    logging.debug("=>> %s" % request)

    for header in sorted(request.headers.keys()):
        logging.debug("=>>   %s: %s" % (header, request.headers[header]))

@app.route('/clear/')
def clear():
    global counts
    counts = {}

    resp = Response("CLEARED")

    return resp

@app.route('/mark/<count>')
def mark(count):
    c = counts.setdefault(count, 0)
    counts[count] = c + 1

    resp = Response("COUNT %d" % counts[count])
    resp.headers['X-Shadowed'] = True

    return resp

@app.route('/check/')
def check():
    return jsonify(counts)

if __name__ == "__main__":
    port = 3000
    ssl_context = None
    conn_type = "HTTP"

    if len(sys.argv) > 1:
        __version__ = sys.argv[1]

    if (len(sys.argv) > 2) and (sys.argv[2] == "--tls"):
        ssl_context = ('shadowsvc.crt', 'shadowsvc.key')
        conn_type = "HTTPS"

    logging.basicConfig(
        level=logging.DEBUG,
        format="%%(asctime)s shadow %s %s %%(levelname)s: %%(message)s" % (__version__, conn_type),
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    logging.info("initializing using %s on port %d" % (conn_type, port))
    app.run(host='0.0.0.0', port=port, debug=True, ssl_context=ssl_context)
