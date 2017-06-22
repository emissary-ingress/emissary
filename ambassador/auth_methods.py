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

        # logging.info("auth returns %s" % rc)

        return rc

    return wrapper

@add_headers
def BasicAuth(store, auth_mapping, auth_headers, req_headers):
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

    # Grab the auth_module from storage. There are enough ways this can fail
    # that we need to manage exceptions here too.
    auth_module = None

    try:
        rc = store.fetch_consumer(username=username)
    except Exception as e:
        rc = RichStatus.fromError("storage exception %s" % (username, e))

    if not rc:
        logging.error("consumer %s: fetch failed: %s" % (username, str(rc)))
    else:
        auth_module = rc.modules.get('authentication', {})

        if not auth_module:
            logging.error("consumer %s: no authentication module" % username)

    # OK, auth_module is valid here, meaning that either it's really the thing
    # from storage and is not None, or it's None meaning that storage failed.
    # From this point forward, we're just going to give a canned
    # "Unauthorized" if anything goes wrong, because we're into the realm of 
    # places where anything more is a security issue. 

    rc = RichStatus.fromError("Unauthorized")

    # To go further we need the auth type, so let's figure that out.
    auth_type = None

    if auth_module:
        try:
            auth_type = auth_module.get('type', None)
        except Exception as e:
            logging.exception(e)
            logging.error("consumer %s: couldn't fetch type from auth config: %s" % (username, e))

    # From this point forward, auth_type is valid (but it might be None,
    # meaning that the admin who set up the consumer got it wrong).

    if auth_type == "basic":
        # If auth_type is correctly set, we know that auth_module is a dict,
        # so we don't need the try/except to look for the password.
        auth_password = auth_module.get('password', None)

        if password == auth_password:
            # Finally, a good result.
            rc = RichStatus.OK(msg="Auth good")

    return rc
