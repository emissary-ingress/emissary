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
from utils import RichStatus, SystemInfo, DelayTrigger

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
CREATE TABLE IF NOT EXISTS services (
    name VARCHAR(64) NOT NULL PRIMARY KEY,
    prefix VARCHAR(2048) NOT NULL,
    port INTEGER NOT NULL
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
        return RichStatus.fromError("no services table in setup: %s" % e)

    return RichStatus.OK()

def getIncomingJSON(req, *needed):
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

def fetch_all_services():
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("SELECT name, prefix FROM services ORDER BY name, prefix")

        services = []

        for name, prefix in cursor:
            services.append({ 'name': name, 'prefix': prefix })

        return RichStatus.OK(services=services, count=len(services))
    except pg8000.Error as e:
        return RichStatus.fromError("services: could not fetch info: %s" % e)

def handle_service_list(req):
    return fetch_all_services()

def handle_service_get(req, name):
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("SELECT prefix FROM services WHERE name = :name", locals())
        [ prefix ] = cursor.fetchone()

        return RichStatus.OK(name=name, prefix=prefix)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not fetch info: %s" % (name, e))

def handle_service_del(req, name):
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("DELETE FROM services WHERE name = :name", locals())
        conn.commit()

        app.reconfigurator.trigger()

        return RichStatus.OK(name=name)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not delete service: %s" % (name, e))

def handle_service_post(req, name):
    try:
        rc = getIncomingJSON(req, 'prefix')

        logging.debug("handle_service_post %s: got args %s" % (name, rc.toDict()))

        if not rc:
            return rc

        prefix = rc.prefix

        logging.debug("handle_service_post %s: prefix %s" % (name, prefix))

        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute('INSERT INTO services VALUES(:name, :prefix, 0)', locals())
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
    rc = fetch_all_services()

    active_service_names = []

    if rc and rc.services:
        active_service_names = [ x['name'] for x in rc.services ]

    app.stats.update(active_service_names)

    return jsonify(app.stats.stats)

def new_config(envoy_base_config=None, envoy_tls_config=None,
               envoy_config_path=None, envoy_restarter_pid=None):
    if not envoy_base_config:
        envoy_base_config = app.envoy_base_config

    if not envoy_tls_config:
        envoy_tls_config = app.envoy_tls_config

    if not envoy_config_path:
        envoy_config_path = app.envoy_config_path

    if not envoy_restarter_pid:
        envoy_restarter_pid = app.envoy_restarter_pid

    config = EnvoyConfig(envoy_base_config, envoy_tls_config)

    rc = fetch_all_services()
    num_services = 0

    if rc and rc.services:
        num_services = len(rc.services)

        for service in rc.services:
            config.add_service(service['name'], service['prefix'])

    config.write_config(envoy_config_path)

    if envoy_restarter_pid > 0:
        os.kill(envoy_restarter_pid, signal.SIGHUP)

    return RichStatus.OK(count=num_services)

@app.route('/ambassador/services', methods=[ 'GET', 'PUT' ])
def root():
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_services: method %s" % request.method)
    
    try:
        rc = setup()

        if rc:
            if request.method == 'PUT':
                app.reconfigurator.trigger()
                rc = RichStatus.OK(msg="reconfigure requested")
            else:
                rc = handle_service_list(request)
    except Exception as e:
        logging.exception(e)
        rc = RichStatus.fromError("handle_services: %s failed: %s" % (request.method, e))

    return jsonify(rc.toDict())

@app.route('/ambassador/service/<name>', methods=[ 'POST', 'GET', 'DELETE' ])
def handle_service(name):
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_service %s: method %s" % (name, request.method))
    
    try:
        rc = setup()

        if rc:
            if request.method == 'POST':
                rc = handle_service_post(request, name)
            elif request.method == 'DELETE':
                rc = handle_service_del(request, name)
            else:
                rc = handle_service_get(request, name)
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

    new_config(envoy_restarter_pid = -1)    # Don't automagically signal here.

    time.sleep(2)

    logging.info("ambassador asking restarter for initial reread")
    os.kill(app.envoy_restarter_pid, signal.SIGHUP)    

    # Set up the trigger for future restarts.
    app.reconfigurator = DelayTrigger(new_config)

    app.run(host='127.0.0.1', port=5000, debug=True)

if __name__ == '__main__':
    setup()
    main()
