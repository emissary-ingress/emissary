import sys

import base64
import functools
import logging

import VERSION

from utils import RichStatus

def add_headers(f):
    func_name = getattr(f, '__name__', '<anonymous>')

    @functools.wraps(f)
    def wrapper(*args, **kwds):
        rc = f(*args, **kwds)

        # Hackery...
        rc.info['headers'] = {
            'Auth-Service': 'Ambassador BasicAuth %s' % VERSION.Version,
        }

        if not rc: 
            rc.headers['WWW-Authenticate'] = 'Basic realm="Login Required"'

        logging.info("auth returns %s" % rc)

        return rc

    return wrapper

@add_headers
def BasicAuth(store, auth_mapping, auth_headers, req_headers):
    if 'BasicAuth' not in auth_mapping['modules']:
        return RichStatus.OK(msg="Auth not required for mapping %s" % auth_mapping['name'])

    ah = auth_headers.get('authorization', None)

    if not ah:
        return RichStatus.fromError("No authorization provided") 

    # logging.info("auth %s" % auth_headers['authorization'])

    if not ah.startswith('Basic '):
        return RichStatus.fromError("Authorization is not basic auth")

    ah = ah[len('Basic '):]

    auth_value = None

    try:
        auth_value = base64.b64decode(ah).decode('utf-8')
    except Exception as e:
        return RichStatus.fromError("Error decoding authorization: %s" % e)

    username = None
    password = None

    try:
        (username, password) = auth_value.split(':', 1)
    except Exception:
        return RichStatus.fromError("Error decoding authorization: need both username and password")

    rc = RichStatus.fromError("impossible error")

    try:
        rc = store.fetch_consumer(username=username)
    except Exception as e:
        rc = RichStatus.fromError("storage failure: %s" % e)

    if not rc:
        return RichStatus.fromError("Could not fetch '%s': %s" % (username, e))

    auth_module = rc.modules.get('BasicAuth', None)

    if not auth_module:
        return RichStatus.fromError("Basic Auth not allowed for %s" % username)

    if password != auth_module.get('password', None):
        return RichStatus.fromError("Unauthorized")

    return RichStatus.OK(msg="Auth good")
