#!/usr/bin/env python

import sys

import functools
import json
import logging
import os
import signal
import time
import uuid

import dpath

from flask import Flask, Response, jsonify, request

import VERSION

from storage_postgres import AmbassadorStore
from envoy import EnvoyStats, EnvoyConfig, TLSConfig
from utils import RichStatus, SystemInfo, PeriodicTrigger

from auth_methods import BasicAuth

__version__ = VERSION.Version

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

logging.info("initializing on %s (resolved %s)" %
             (SystemInfo.MyHostName, SystemInfo.MyResolvedName))

app = Flask(__name__)

def getIncomingJSON(req, *needed):
    """
    Pull incoming arguments from JSON into a RichStatus. 'needed' specifies
    keys that are mandatory, but _all keys are converted_, so for any optional
    things, just check if the key is present in the returned RichStatus.

    If any 'needed' keys are missing, returns a False RichStatus including error
    text. If all 'needed' keys are present, returns a True RichStatus with elements
    copied from the input.
    """

    try:
        incoming = req.get_json()
    except Exception as e:
        return RichStatus.fromError("invalid JSON: %s" % e)

    logging.debug("getIncomingJSON: %s" % incoming)

    if not incoming:
        incoming = {}

    missing = []

    for key in needed:
        if key not in incoming:
            missing.append(key)

    if missing:
        return RichStatus.fromError("Required fields missing: %s" % " ".join(missing))
    else:
        return RichStatus.OK(**incoming)

######## MODULE UTILITIES

def handle_module_list(req):
    return app.storage.fetch_all_modules()

def handle_module_get(req, module_name):
    return app.storage.fetch_module(module_name)

def handle_module_del(req, module_name):
    rc = app.storage.delete_module(module_name)

    if rc:
        app.reconfigurator.trigger()

    return rc

def handle_module_store(req, module_name):
    module_data = req.json

    logging.debug("handle_mapping_store_module: got args %s" % module_data)

    rc = app.storage.store_module(module_name, module_data)

    if rc:
        app.reconfigurator.trigger()

    return rc

######## MAPPING UTILITIES

def handle_mapping_list(req):
    return app.storage.fetch_all_mappings()

def handle_mapping_get(req, name):
    return app.storage.fetch_mapping(name)

def handle_mapping_del(req, name):
    rc = app.storage.delete_mapping(name)

    if rc:
        app.reconfigurator.trigger()

    return rc

def handle_mapping_store(req, name):
    rc = getIncomingJSON(req, 'prefix', 'service')

    logging.debug("handle_mapping_store %s: got args %s" % (name, rc.toDict()))

    if not rc:
        return rc

    prefix = rc.prefix
    service = rc.service
    rewrite = rc.rewrite if ('rewrite' in rc) else '/'
    modules = rc.modules if ('modules' in rc) else {}

    logging.debug("handle_mapping_store %s: pfx %s => svc %s (rewrite %s, modules %s)" %
                  (name, prefix, service, rewrite, modules))

    rc = app.storage.store_mapping(name, prefix, service, rewrite, modules)

    if rc:
        app.reconfigurator.trigger()

    return rc

def handle_mapping_get_module(req, mapping_name, module_name):
    return app.storage.fetch_mapping_module(mapping_name, module_name)

def handle_mapping_delete_module(req, mapping_name, module_name):
    return app.storage.delete_mapping_module(mapping_name, module_name)

def handle_mapping_store_module(req, mapping_name, module_name):
    module_data = req.json

    logging.debug("handle_mapping_store_module: got args %s" % module_data)

    return app.storage.store_mapping_module(mapping_name, module_name, module_data)

######## PRINCIPAL UTILITIES

def handle_principal_list(req):
    return app.storage.fetch_all_principals()

def handle_principal_get(req, name):
    return app.storage.fetch_principal(name)

def handle_principal_del(req, name):
    rc = app.storage.delete_principal(name)

    if rc:
        app.reconfigurator.trigger()

    return rc

def handle_principal_post(req, name):
    rc = getIncomingJSON(req, 'fingerprint')

    logging.debug("handle_principal_post %s: got args %s" % (name, rc.toDict()))

    if not rc:
        return rc

    fingerprint = rc.fingerprint

    logging.debug("handle_principal_post %s: fingerprint %s" % (name, fingerprint))

    rc = app.storage.store_principal(name, fingerprint)

    if rc:
        app.reconfigurator.trigger()

    return rc

######## CONSUMER UTILITIES

def handle_consumer_list(req):
    return app.storage.fetch_all_consumers()

def handle_consumer_get(req, consumer_id):
    return app.storage.fetch_consumer(consumer_id)

def handle_consumer_del(req, consumer_id):
    return app.storage.delete_consumer(consumer_id)

def handle_consumer_store(req, consumer_id):
    rc = getIncomingJSON(req, 'username', 'fullname')

    logging.debug("handle_consumer_store: got args %s" % rc.toDict())

    if not rc:
        return rc

    username = rc.username
    fullname = rc.fullname
    shortname = rc.shortname if ('shortname' in rc) else fullname
    modules = rc.modules if ('modules' in rc) else {}

    logging.debug(
        "handle_consumer_post %s: username '%s', fullname '%s', shortname '%s', modules %d" %
        (consumer_id, username, fullname, shortname, len(modules.keys()))
    )

    return app.storage.store_consumer(consumer_id, username, fullname, shortname, modules)

def handle_consumer_get_module(req, consumer_id, module_name):
    return app.storage.fetch_consumer_module(consumer_id, module_name)

def handle_consumer_delete_module(req, consumer_id, module_name):
    return app.storage.delete_consumer_module(consumer_id, module_name)

def handle_consumer_store_module(req, consumer_id, module_name):
    module_data = req.json

    logging.debug("handle_consumer_store_module: got args %s" % module_data)

    return app.storage.store_consumer_module(consumer_id, module_name, module_data)

######## CONFIG UTILITIES

def new_config(envoy_base_config=None, envoy_tls_config=None, envoy_config_path=None, envoy_restarter_pid=None):
    # logging.debug("new_config entry...")

    rc = app.storage.fetch_all_modules()

    current_modules = rc.modules if rc else {}
    
    rc = app.storage.fetch_all_mappings()

    if not rc:
        # Failed to fetch mappings from DB.
        logging.debug("new_config could not fetch mappings: %s" % rc)
        return rc

    # Suppose the fetch "succeeded" but we got no mappings?
    if 'mappings' not in rc:
        # This shouldn't happen. Clear out app.current_mappings so that
        # we have a better chance of recovering when we get data, but don't
        # actually change our Envoy config.

        app.current_mappings = None

        logging.debug("new_config got no mappings at all? %s" % str(rc))
        return RichStatus.fromError("no mappings found at all, original rc %s" % str(rc))

    num_mappings = 0
    current_mappings = rc.mappings

    try:
        num_mappings = len(current_mappings)
    except:
        # Huh. This can really only happen if something isn't quite initialized
        # in the database yet.
        logging.debug("new_config got %s for mappings? assuming empty" % type(current_mappings))
        current_mappings = []

    # print("mappings in new_config: %s" % ", ".join([ "%s - %s" % (m['name'], m['prefix']) for m in current_mappings ]))

    if (current_modules != app.current_modules) or (current_mappings != app.current_mappings):
        logging.debug("new_config found changes (modules %d, mappings %d)" % 
                      (len(current_modules.keys()), num_mappings))

        # Mappings have changed. Really perform work for a new config.
        if not envoy_base_config:
            envoy_base_config = app.envoy_base_config

        if not envoy_tls_config:
            envoy_tls_config = app.envoy_tls_config

        if not envoy_config_path:
            envoy_config_path = app.envoy_config_path

        if not envoy_restarter_pid:
            envoy_restarter_pid = app.envoy_restarter_pid

        config = EnvoyConfig(envoy_base_config, envoy_tls_config, current_modules)

        for mapping in current_mappings:
            config.add_mapping(mapping['name'], mapping['prefix'],
                               mapping['service'], mapping['rewrite'])

        try:
            config.write_config(envoy_config_path)

            if envoy_restarter_pid > 0:
                os.kill(envoy_restarter_pid, signal.SIGHUP)

            app.current_modules = current_modules
            app.current_mappings = current_mappings
        except Exception as e:
            logging.exception(e)
            logging.error("new_config couldn't write config")
    else:
        # logging.debug("new_config found NO changes (count %d)" % num_mappings)
        pass

    return RichStatus.OK(count=num_mappings)

######## DECORATORS

def standard_handler(f):
    func_name = getattr(f, '__name__', '<anonymous>')

    @functools.wraps(f)
    def wrapper(*args, **kwds):
        rc = RichStatus.fromError("impossible error")
        logging.debug("%s: method %s" % (func_name, request.method))

        try:
            rc = f(*args, **kwds)
        except Exception as e:
            logging.exception(e)
            rc = RichStatus.fromError("%s: %s failed: %s" % (func_name, request.method, e))

        return jsonify(rc.toDict())

    return wrapper

######## FLASK ROUTES

@app.route('/ambassador/health', methods=[ 'GET' ])
@standard_handler
def health():
    return RichStatus.OK(msg="ambassador health check OK")

@app.route('/ambassador/stats', methods=[ 'GET' ])
@standard_handler
def ambassador_stats():
    rc = handle_mapping_list(request)

    active_mapping_names = []

    if rc and rc.mappings:
        active_mapping_names = [ x['name'] for x in rc.mappings ]

    app.stats.update(active_mapping_names)

    return RichStatus.OK(stats=app.stats.stats)

@app.route('/ambassador/module', methods=[ 'GET' ])
@standard_handler
def handle_modules():
    return handle_module_list(request)

@app.route('/ambassador/module/<module_name>', methods=[ 'POST', 'PUT', 'GET', 'DELETE' ])
@standard_handler
def handle_module(module_name):
    if request.method == 'PUT':
        return handle_module_store(request, module_name)
    elif request.method == 'DELETE':
        return handle_module_del(request, module_name)
    else:
        return handle_module_get(request, module_name)

@app.route('/ambassador/mapping', methods=[ 'GET' ])
@standard_handler
def handle_mappings():
    return handle_mapping_list(request)

# Backward compatability. No one should be using this.
@app.route('/ambassador/mappings', methods=[ 'GET', 'PUT' ])
@standard_handler
def handle_mappings_deprecated():
    if request.method == 'PUT':
        app.reconfigurator.trigger()
        return RichStatus.OK(msg="DEPRECATED: was reconfigure requested, now basically no-op")
    else:
        rc = handle_mapping_list(request)

        # XXX Hackery!
        rc.info['deprecated'] = 'use /ambassador/mapping (singular) instead'
        return rc

@app.route('/ambassador/mapping/<name>', methods=[ 'POST', 'PUT', 'GET', 'DELETE' ])
@standard_handler
def handle_mapping(name):
    if request.method == 'PUT':
        return handle_mapping_store(request, name)
    elif request.method == 'POST':
        rc = handle_mapping_store(request, name)

        # XXX Hackery!
        rc.info['deprecated'] = 'use PUT instead'

        return rc
    elif request.method == 'DELETE':
        return handle_mapping_del(request, name)
    else:
        return handle_mapping_get(request, name)

@app.route('/ambassador/mapping/<mapping_id>/module/<module_name>', methods=[ 'GET', 'PUT', 'DELETE' ])
@standard_handler
def handle_mapping_module(mapping_id, module_name):
    if request.method == 'PUT':
        return handle_mapping_store_module(request, mapping_id, module_name)
    elif request.method == 'DELETE':
        return handle_mapping_delete_module(request, mapping_id, module_name)
    else:
        return handle_mapping_get_module(request, mapping_id, module_name)

@app.route('/ambassador/principals', methods=[ 'GET', 'PUT' ])
@standard_handler
def handle_principals():
    if request.method == 'PUT':
        app.reconfigurator.trigger()
        return RichStatus.OK(msg="reconfigure requested")
    else:
        return handle_principal_list(request)

@app.route('/v1/certs/list/approved', methods=[ 'GET', 'PUT' ])
@standard_handler
def handle_approved():
    rc = handle_principal_list(request)

    if rc:
        principals = [ { "fingerprint_sha256": x['fingerprint'] } for x in rc.principals ]

        rc = RichStatus.OK(certificates=principals)

    return rc

@app.route('/ambassador/principal/<name>', methods=[ 'POST', 'GET', 'DELETE' ])
@standard_handler
def handle_principal(name):
    if request.method == 'POST':
        return handle_principal_post(request, name)
    elif request.method == 'DELETE':
        return handle_principal_del(request, name)
    else:
        return handle_principal_get(request, name)

@app.route('/ambassador/consumer', methods=[ 'GET', 'POST' ])
@standard_handler
def handle_consumers():
    if request.method == 'POST':
        consumer_id = uuid.uuid4().hex.upper();

        return handle_consumer_store(request, consumer_id)
    else:
        return handle_consumer_list(request)

@app.route('/ambassador/consumer/<consumer_id>', methods=[ 'PUT', 'GET', 'DELETE' ])
@standard_handler
def handle_consumer(consumer_id):
    if request.method == 'PUT':
        return handle_consumer_store(request, consumer_id)
    elif request.method == 'DELETE':
        return handle_consumer_del(request, consumer_id)
    else:
        return handle_consumer_get(request, consumer_id)

@app.route('/ambassador/consumer/<consumer_id>/module/<module_name>', methods=[ 'GET', 'PUT', 'DELETE' ])
@standard_handler
def handle_consumer_module(consumer_id, module_name):
    if request.method == 'PUT':
        return handle_consumer_store_module(request, consumer_id, module_name)
    elif request.method == 'DELETE':
        return handle_consumer_delete_module(request, consumer_id, module_name)
    else:
        return handle_consumer_get_module(request, consumer_id, module_name)

@app.route('/ambassador/auth', methods=[ 'POST' ])
def handle_extauth():
    auth_headers = request.json
    req_headers = request.headers

    # logging.info("======== auth:")
    # logging.info("==== REQ")

    # for header in sorted(req_headers.keys()):
    #     logging.info("%s: %s" % (header, req_headers[header]))

    # logging.info("==== AUTH")

    # for header in sorted(auth_headers.keys()):
    #     logging.info("%s: %s" % (header, auth_headers[header]))

    path = auth_headers.get(':path', None)

    auth_mapping = None

    if path and len(path):
        # logging.info("==== MAPPINGS")

        for mapping in app.current_mappings:
            match = path.startswith(mapping['prefix'])

            # logging.debug("checking %s: %s -- %smatch" %
            #                (mapping['name'], mapping['prefix'],
            #                ("" if match else "no ")))

            if match:
                auth_mapping = mapping
                break

    if not auth_mapping:
        return Response('Auth not required for path %s' % path, 200)

    auth_modules = auth_mapping.get('modules', {})

    if not 'authentication' in auth_modules:
        return Response('Auth not required for path %s' % path, 200)

    auth_config = auth_modules['authentication']

    rc = RichStatus.fromError('authentication failed')

    try:
        auth_type = auth_config.get('type', None)

        if auth_type == 'basic':
            rc = BasicAuth(app.storage, auth_mapping, auth_headers, req_headers)
        elif auth_type:
            logging.error('%s: authentication type %s is not supported' % 
                          (auth_mapping['prefix'], auth_type))
        else:
            logging.error('%s: no authentication type given?' % auth_mapping['prefix'])
    except Exception as e:
        logging.exception(e)
        logging.error("%s: couldn't fetch authentication type at all??" % auth_mapping['prefix'])

    if not rc:
        logging.info("auth failed: %s" % rc.error)
        return Response(rc.error, 401, rc.headers if 'headers' in rc else None)
    else:
        logging.info("auth OK: %s" % rc.msg)
        return Response(rc.msg, 200)

def main():
    # Set up storage.
    app.storage = AmbassadorStore()

    # Set up config templates and restarter.
    app.envoy_template_path = sys.argv[1]
    app.envoy_config_path = sys.argv[2]
    app.envoy_restarter_pid_path = sys.argv[3]
    app.envoy_restarter_pid = None

    # Load the base config.
    app.envoy_base_config = json.load(open(app.envoy_template_path, "r"))

    # Set up the TLS config stuff.
    app.envoy_tls_config = TLSConfig(
        "AMBASSADOR_CHAIN_PATH", "/etc/certs/fullchain.pem",
        "AMBASSADOR_PRIVKEY_PATH", "/etc/certs/privkey.pem",
        "AMBASSADOR_CACERT_PATH", "/etc/cacert/fullchain.pem"
    )

    # Set up stats.
    app.stats = EnvoyStats()

    # Learn the PID of the restarter.
    while app.envoy_restarter_pid is None:
        try:
            pid_file = open(app.envoy_restarter_pid_path, "r")

            app.envoy_restarter_pid = int(pid_file.read().strip())
        except FileNotFoundError:
            logging.info("ambassador found no restarter PID")
            time.sleep(1)
        except IOError:
            logging.info("ambassador found unreadable restarter PID")
            time.sleep(1)
        except ValueError:
            logging.info("ambassador found invalid restarter PID")
            time.sleep(1)

    logging.info("ambassador found restarter PID %d" % app.envoy_restarter_pid)

    app.current_modules = None
    app.current_mappings = None
    new_config(envoy_restarter_pid = -1)    # Don't automagically signal here.

    time.sleep(2)

    logging.info("ambassador asking restarter for initial reread")
    os.kill(app.envoy_restarter_pid, signal.SIGHUP)    

    # Set up the trigger for future restarts.
    app.reconfigurator = PeriodicTrigger(new_config)

    app.run(host='127.0.0.1', port=5000, debug=True)

if __name__ == '__main__':
    main()
