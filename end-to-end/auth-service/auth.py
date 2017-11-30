#!/usr/bin/env python

import sys

# import functools
# import json
import logging
import pprint

from flask import Flask, Response, jsonify, request

__version__ = '0.0.1'

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

    resp = Response('You want path: %s' % path)

    if path.startswith('/ambassador/'):
        resp.status_code = 200     
    elif path.endswith("/good/") or path.endswith("/demo/"):
        resp.status_code = 200
        resp.headers['X-Hurkle'] = 'Oh baby, oh baby.'
    elif path.endswith("/nohdr/"):
        resp.status_code = 200
        # Don't add the header.
    else:
        resp.status_code = 403

    resp.headers['X-Test'] = 'Should not be seen.'
    return resp

if __name__ == "__main__":
    ssl_context = None
    conn_type = "HTTP"
    port = 80

    if (len(sys.argv) > 1) and (sys.argv[1] == "--tls"):
        ssl_context = ('authsvc.crt', 'authsvc.key')
        conn_type = "HTTPS"
        port = 443

    logging.basicConfig(
        # filename=logPath,
        level=logging.DEBUG, # if appDebug else logging.INFO,
        format="%%(asctime)s auth %s %s %%(levelname)s: %%(message)s" % (__version__, conn_type),
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    logging.info("initializing")
    app.run(host='0.0.0.0', port=port, debug=True, ssl_context=ssl_context)
