#!/usr/bin/env python3

# Copyright 2019 Datawire. All rights reserved.
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
import select
import socket

__version__ = '0.1.0'

logging.basicConfig(
    level=logging.DEBUG,
    format="%%(asctime)s stats-web %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

from flask import Flask, Response, jsonify, request

__version__ = "?.?.?"

app = Flask(__name__)

@app.before_request
def before():
    logging.debug("=>> %s" % request)

    for header in sorted(request.headers.keys()):
        logging.debug("=>>   %s: %s" % (header, request.headers[header]))

@app.route('/', defaults={'cmd': ''})
@app.route('/<path:cmd>')
def hrm(cmd):
    while cmd.startswith('/'):
        cmd = cmd[1:]

    while cmd.endswith('/'):
        cmd = cmd[:-1]

    logging.debug(f"forwarding command {cmd}")

    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)  # UDP
    sock.setblocking(0)

    sock.sendto(bytes(cmd, "utf-8"), ('127.0.0.1', 8125))

    status = 500
    text = f'{cmd} timed out'

    ready = select.select([ sock ], [], [], 2)

    if ready[0]:
        text, server_address = sock.recvfrom(8192)
        status = 200

    resp = Response(text, status=status)
    resp.headers['X-StatsWeb-Version'] = __version__

    return resp

if __name__ == "__main__":
    logging.info("initializing")
    app.run(host='0.0.0.0', port=3000, debug=True)
