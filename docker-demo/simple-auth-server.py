#!/usr/bin/env python

import sys

# import functools
# import json
import logging
import pprint

from flask import Flask, Response, jsonify, request

__version__ = '0.0.1'

ValidUsers = {
    'username': 'password'
}

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

logging.info("initializing")

# class LoggingMiddleware(object):
#     def __init__(self, app):
#         self._app = app

#     def __call__(self, environ, resp):
#         errorlog = environ['wsgi.errors']
#         pprint.pprint(('REQUEST', environ), stream=errorlog)

#         def log_response(status, headers, *args):
#             pprint.pprint(('RESPONSE', status, headers), stream=errorlog)
#             return resp(status, headers, *args)

#         return self._app(environ, log_response)

app = Flask(__name__)
# app.wsgi_app = LoggingMiddleware(app.wsgi_app)

@app.before_request
def before():
    logging.debug("=>> %s" % request)

    for header in sorted(request.headers.keys()):
        logging.debug("=>>   %s: %s" % (header, request.headers[header]))

def check_auth(auth):
    if not auth:
        return False

    if auth.username not in ValidUsers:
        return False

    if auth.password != ValidUsers[auth.username]:
        return False

    return True

@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def catch_all(path):
    # Restore the leading '/' to our path.
    path = "/" + path

    resp = Response('You want path: %s' % path)

    if path.startswith("/extauth/qotm/quote"):
        auth = request.authorization

        # We require Basic-Auth for this.
        if not check_auth(auth):
            resp = Response(
                "Authentication is required for %s" % path,
                401,
                { 
                    'WWW-Authenticate': 'Basic realm="Ambassador"',
                    'X-ExtAuth-Required': 'True'
                }
            )
        else:
            resp = Response(
                "Authentication succeeded for %s" % path,
                200,
                {
                    'X-Authenticated-As': auth.username,
                    'X-ExtAuth-Required': 'True'
                }                    
            )
    else:
        resp = Response(
            "Authentication not required for %s" % path,
            200,
            { 'X-ExtAuth-Required': 'False' }
        )

    resp.headers['X-Test'] = 'Should not be seen.'

    logging.debug("=<< %s" % resp)

    for header in sorted(resp.headers.keys()):
        logging.debug("=<<   %s: %s" % (header, resp.headers[header]))

    return resp

app.run(host='127.0.0.1', port=3000, debug=True)
