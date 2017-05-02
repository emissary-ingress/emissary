#!/usr/bin/env python

import sys

import json
import logging
import os
import signal
import time

import dpath
import pg8000

from flask import Flask, jsonify, request

import VERSION

from envoy import EnvoyStats, EnvoyConfig, TLSConfig
from utils import RichStatus, SystemInfo, PeriodicTrigger

__version__ = VERSION.Version

pg8000.paramstyle = 'named'

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

logging.info("initializing on %s (resolved %s)" %
             (SystemInfo.MyHostName, SystemInfo.MyResolvedName))

app = Flask(__name__)

AMBASSADOR_TABLE_SQL = '''
CREATE TABLE IF NOT EXISTS mappings (
    name VARCHAR(64) NOT NULL PRIMARY KEY,
    prefix VARCHAR(2048) NOT NULL,
    service VARCHAR(2048) NOT NULL,
    rewrite VARCHAR(2048) NOT NULL
)
'''


def get_db(database):
    db_host = "ambassador-store"
    db_port = 5432

    if "AMBASSADOR_DB_HOST" in os.environ:
        db_host = os.environ["AMBASSADOR_DB_HOST"]

    if "AMBASSADOR_DB_PORT" in os.environ:
        db_port = int(os.environ["AMBASSADOR_DB_PORT"])

    return pg8000.connect(user="postgres", password="postgres",
                          database=database, host=db_host, port=db_port)

def setup():
    try:
        conn = get_db("postgres")
        conn.autocommit = True

        cursor = conn.cursor()
        cursor.execute("SELECT 1 FROM pg_database WHERE datname = 'ambassador'")
        results = cursor.fetchall()

        if not results:
            cursor.execute("CREATE DATABASE ambassador")

        conn.close()
    except pg8000.Error as e:
        return RichStatus.fromError("no ambassador database in setup: %s" % e)

    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()
        cursor.execute(AMBASSADOR_TABLE_SQL)
        conn.commit()
        conn.close()
    except pg8000.Error as e:
        return RichStatus.fromError("no mappings table in setup: %s" % e)

    return RichStatus.OK()

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

########

def fetch_all_mappings():
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("SELECT name, prefix, service, rewrite FROM mappings ORDER BY name, prefix")

        mappings = []

        for name, prefix, service, rewrite in cursor:
            mappings.append({ 'name': name, 'prefix': prefix, 
                              'service': service, 'rewrite': rewrite })

        return RichStatus.OK(mappings=mappings, count=len(mappings))
    except pg8000.Error as e:
        return RichStatus.fromError("mappings: could not fetch info: %s" % e)

def handle_mapping_list(req):
    return fetch_all_mappings()

def handle_mapping_get(req, name):
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("SELECT prefix, service, rewrite FROM mappings WHERE name = :name", locals())
        [ prefix, service, rewrite ] = cursor.fetchone()

        return RichStatus.OK(name=name, prefix=prefix, service=service, rewrite=rewrite)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not fetch info: %s" % (name, e))

def handle_mapping_del(req, name):
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("DELETE FROM mappings WHERE name = :name", locals())
        conn.commit()

        app.reconfigurator.trigger()

        return RichStatus.OK(name=name)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not delete mapping: %s" % (name, e))

def handle_mapping_post(req, name):
    try:
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

        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute('INSERT INTO mappings VALUES(:name, :prefix, :service, :rewrite)',
                       locals())
        conn.commit()

        app.reconfigurator.trigger()

        return RichStatus.OK(name=name)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not save info: %s" % (name, e))

@app.route('/ambassador/health', methods=[ 'GET' ])
def health():
    rc = RichStatus.OK(msg="ambassador health check OK")

    return jsonify(rc.toDict())

@app.route('/ambassador/stats', methods=[ 'GET' ])
def ambassador_stats():
    rc = fetch_all_mappings()

    active_mapping_names = []

    if rc and rc.mappings:
        active_mapping_names = [ x['name'] for x in rc.mappings ]

    app.stats.update(active_mapping_names)

    return jsonify(app.stats.stats)

def new_config(envoy_base_config=None, envoy_tls_config=None,
               envoy_config_path=None, envoy_restarter_pid=None):
    rc = fetch_all_mappings()
    if not rc:
        # Failed to fetch mappings from DB.
        return rc

    num_mappings = len(rc.mappings)
    if rc.mappings != app.current_mappings:
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

        config.write_config(envoy_config_path)

        if envoy_restarter_pid > 0:
            os.kill(envoy_restarter_pid, signal.SIGHUP)

        app.current_mappings = rc.mappings

    return RichStatus.OK(count=num_mappings)

@app.route('/ambassador/mappings', methods=[ 'GET', 'PUT' ])
def handle_mappings():
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_mappings: method %s" % request.method)
    
    try:
        rc = setup()

        if rc:
            if request.method == 'PUT':
                app.reconfigurator.trigger()
                rc = RichStatus.OK(msg="reconfigure requested")
            else:
                rc = handle_mapping_list(request)
    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("handle_mappings: %s failed: %s" % (request.method, e))

    return jsonify(rc.toDict())

@app.route('/ambassador/mapping/<name>', methods=[ 'POST', 'GET', 'DELETE' ])
def handle_mapping(name):
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_mapping %s: method %s" % (name, request.method))
    
    try:
        rc = setup()

        if rc:
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

def main():
    app.envoy_template_path = sys.argv[1]
    app.envoy_config_path = sys.argv[2]
    app.envoy_restarter_pid_path = sys.argv[3]
    app.envoy_restarter_pid = None

    # Load the base config.
    app.envoy_base_config = json.load(open(app.envoy_template_path, "r"))

    # Set up the TLS config stuff.
    app.envoy_tls_config = TLSConfig(
        "AMBASSADOR_CHAIN_PATH", "/etc/certs/fullchain.pem",
        "AMBASSADOR_PRIVKEY_PATH", "/etc/certs/privkey.pem"
    )

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
    setup()
    main()
