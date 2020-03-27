#!/usr/bin/env python

import sys

import logging
import uuid

from flask import Flask, Response, jsonify, request

__version__ = '0.0.1'

AdminUsers = {
    'admin': 'admin',
    '<<realm>>': 'Ambassador Diagnostics'
}

DemoUsers = {
    'username': 'password',
    '<<realm>>': 'Ambassador'
}

app = Flask(__name__)

@app.before_request
def before():
    logging.debug("=>> %s" % request)

    for header in sorted(request.headers.keys()):
        logging.debug("=>>   %s: %s" % (header, request.headers[header]))

def check_auth(auth, validusers):
    if not auth:
        return False

    if auth.username not in validusers:
        return False

    if auth.password != validusers[auth.username]:
        return False

    return True

@app.route('/', defaults={'path': ''}, methods=['GET', 'PUT', 'POST', 'DELETE'])
@app.route('/<path:path>', methods=['GET', 'PUT', 'POST', 'DELETE'])
def catch_all(path):
    # Restore the leading '/' to our path.
    path = "/" + path

    resp = Response('You want path: %s' % path)

    validusers = None
    generate_session = False


    if not path.startswith("/auth/v0/"):
        logging.info("direct access attempted to %s" % path)
        resp = Response(
            "Direct access not supported",
            400,
            {
                'X-ExtAuth-Required': 'True',
            }
        )

    # Drop the auth prefix, but leave a leading /.
    path = path[len("/auth/v0"):]

    while path.startswith("//"):
        path = path[1:]

    logging.debug("Requested path is %s" % path)

    if path.startswith("/ambassador/"):
        # Require auth as an admin.
        validusers = AdminUsers
    elif path.startswith("/qotm/quote"):
        # Require auth as a user, and generate a session.
        validusers = DemoUsers
        generate_session = True

    if validusers:
        auth = request.authorization

        # We require Basic-Auth for this.
        if not check_auth(auth, validusers):
            resp = Response(
                "Authentication is required for %s" % path,
                401,
                {
                    'WWW-Authenticate': 'Basic realm="%s"' % validusers['<<realm>>'],
                    'X-ExtAuth-Required': 'True'
                }
            )
        elif generate_session:
            session = request.headers.get('x-qotm-session', None)

            if not session:
                session = str(uuid.uuid4()).upper()
                logging.debug("Generated new QOTM session ID %s" % session)

            resp = Response(
                "Authentication succeeded for %s" % path,
                200,
                {
                    'X-Authenticated-As': auth.username,
                    'X-ExtAuth-Required': 'True',
                    'X-QoTM-Session': session
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

if __name__ == "__main__":
    logging.basicConfig(
        # filename=logPath,
        level=logging.DEBUG, # if appDebug else logging.INFO,
        format="%%(asctime)s demo-auth %s %%(levelname)s: %%(message)s" % __version__,
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    logging.info("initializing")
    app.run(host='0.0.0.0', port=5050, debug=True)
