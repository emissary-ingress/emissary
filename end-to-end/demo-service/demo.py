#!/usr/bin/env python

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

@app.route('/', methods=[ 'GET' ])
def demo():
    return "VERSION %s" % __version__

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
