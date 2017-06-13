#!/usr/bin/env python

import sys

import json
import logging
import os
import signal
import time

import dpath

from flask import Flask, jsonify, request

import VERSION

from storage_postgres import AmbassadorStore
from envoy import EnvoyStats, EnvoyConfig, TLSConfig
from utils import RichStatus, SystemInfo, PeriodicTrigger

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

######## SERVICE UTILITIES

def handle_mapping_list(req):
    return app.storage.fetch_all_mappings()

def handle_mapping_get(req, name):
    return app.storage.fetch_mapping(name)

def handle_mapping_del(req, name):
    rc = app.storage.delete_mapping(name)

    if rc:
        app.reconfigurator.trigger()

    return rc

def handle_mapping_post(req, name):
    rc = getIncomingJSON(req, 'prefix', 'service')

    logging.debug("handle_mapping_post %s: got args %s" % (name, rc.toDict()))

    if not rc:
        return rc

    prefix = rc.prefix
    service = rc.service
    rewrite = '/'

    if 'rewrite' in rc:
        rewrite = rc.rewrite

    logging.debug("handle_mapping_post %s: pfx %s => svc %s (rewrite %s)" %
                  (name, prefix, service, rewrite))

    rc = app.storage.store_mapping(name, prefix, service, rewrite)

    if rc:
        app.reconfigurator.trigger()

    return rc

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

######## CONFIG UTILITIES

def new_config(envoy_base_config=None, envoy_tls_config=None, envoy_config_path=None, envoy_restarter_pid=None):
    # logging.debug("new_config entry...")
    
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

    try:
        num_mappings = len(rc.mappings)
    except:
        # Huh. This can really only happen if something isn't quite initialized
        # in the database yet.
        logging.debug("new_config got %s for mappings? assuming empty" % type(rc.mappings))
        rc.mappings = []

    if rc.mappings != app.current_mappings:
        logging.debug("new_config found changes (count %d)" % num_mappings)

        # Mappings have changed. Really perform work for a new config.
        if not envoy_base_config:
            envoy_base_config = app.envoy_base_config

        if not envoy_tls_config:
            envoy_tls_config = app.envoy_tls_config

        if not envoy_config_path:
            envoy_config_path = app.envoy_config_path

        if not envoy_restarter_pid:
            envoy_restarter_pid = app.envoy_restarter_pid

        config = EnvoyConfig(envoy_base_config, envoy_tls_config)

        for mapping in rc.mappings:
            config.add_mapping(mapping['name'], mapping['prefix'],
                               mapping['service'], mapping['rewrite'])

        try:
            config.write_config(envoy_config_path)

            if envoy_restarter_pid > 0:
                os.kill(envoy_restarter_pid, signal.SIGHUP)

            app.current_mappings = rc.mappings
        except Exception as e:
            logging.exception(e)
            logging.error("new_config couldn't write config")
    else:
        # logging.debug("new_config found NO changes (count %d)" % num_mappings)
        pass

    return RichStatus.OK(count=num_mappings)

######## FLASK ROUTES

@app.route('/ambassador/health', methods=[ 'GET' ])
def health():
    rc = RichStatus.OK(msg="ambassador health check OK")

    return jsonify(rc.toDict())

@app.route('/ambassador/stats', methods=[ 'GET' ])
def ambassador_stats():
    rc = handle_mapping_list(request)

    active_mapping_names = []

    if rc and rc.mappings:
        active_mapping_names = [ x['name'] for x in rc.mappings ]

    app.stats.update(active_mapping_names)

    return jsonify(app.stats.stats)

@app.route('/ambassador/mappings', methods=[ 'GET', 'PUT' ])
def handle_mappings():
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_mappings: method %s" % request.method)

    try:
        if request.method == 'PUT':
            app.reconfigurator.trigger()
            rc = RichStatus.OK(msg="reconfigure requested")
        else:
            rc = handle_mapping_list(request)
    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("%s: %s failed: %s" % (name, request.method, e))

    return jsonify(rc.toDict())

@app.route('/ambassador/mapping/<name>', methods=[ 'POST', 'GET', 'DELETE' ])
def handle_mapping(name):
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_mapping %s: method %s" % (name, request.method))
    
    try:
        if request.method == 'POST':
            rc = handle_mapping_post(request, name)
        elif request.method == 'DELETE':
            rc = handle_mapping_del(request, name)
        else:
            rc = handle_mapping_get(request, name)
    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("%s: %s failed: %s" % (name, request.method, e))

    return jsonify(rc.toDict())

@app.route('/ambassador/principals', methods=[ 'GET', 'PUT' ])
def handle_principals():
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_principals: method %s" % request.method)
    
    try:
        if request.method == 'PUT':
            app.reconfigurator.trigger()
            rc = RichStatus.OK(msg="reconfigure requested")
        else:
            rc = handle_principal_list(request)
    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("handle_principals: %s failed: %s" % (request.method, e))

    return jsonify(rc.toDict())

@app.route('/v1/certs/list/approved', methods=[ 'GET', 'PUT' ])
def handle_approved():
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_principals: method %s" % request.method)
    
    try:
        rc = handle_principal_list(request)

        if rc:
            principals = [ { "fingerprint_sha256": x['fingerprint'] } for x in rc.principals ]

            rc = RichStatus.OK(certificates=principals)

    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("handle_principals: %s failed: %s" % (request.method, e))

    return jsonify(rc.toDict())

@app.route('/ambassador/principal/<name>', methods=[ 'POST', 'GET', 'DELETE' ])
def handle_principal(name):
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_principal %s: method %s" % (name, request.method))
    
    try:
        if request.method == 'POST':
            rc = handle_principal_post(request, name)
        elif request.method == 'DELETE':
            rc = handle_principal_del(request, name)
        else:
            rc = handle_principal_get(request, name)
    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("%s: %s failed: %s" % (name, request.method, e))

    return jsonify(rc.toDict())

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
