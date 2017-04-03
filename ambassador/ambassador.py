#!/usr/bin/env python

import sys

import copy
import json
import logging
import os
import signal
import socket
import time
import uuid

import dpath
import pg8000
from flask import Flask, jsonify, request

pg8000.paramstyle = 'named'

logPath = "/tmp/flasklog"

MyHostName = socket.gethostname()
MyResolvedName = socket.gethostbyname(socket.gethostname())

# Don't change this line without also changing .bumpversion.cfg
__version__ = "0.1.7"

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%(asctime)s ambassador 0.1.7 %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

logging.info("ambassador initializing on %s (resolved %s)" % (MyHostName, MyResolvedName))

app = Flask(__name__)

AMBASSADOR_TABLE_SQL = '''
CREATE TABLE IF NOT EXISTS services (
    name VARCHAR(64) NOT NULL PRIMARY KEY,
    prefix VARCHAR(2048) NOT NULL,
    port INTEGER NOT NULL
)
'''

class RichStatus (object):
    def __init__(self, ok, **kwargs):
        self.ok = ok
        self.info = kwargs
        self.info['hostname'] = MyHostName
        self.info['resolvedname'] = MyResolvedName
        self.info['version'] = __version__

    # Remember that __getattr__ is called only as a last resort if the key
    # isn't a normal attr.
    def __getattr__(self, key):
        return self.info.get(key)

    def __nonzero__(self):
        return self.ok

    def __str__(self):
        attrs = ["%=%s" % (key, self.info[key]) for key in sorted(self.info.keys())]
        astr = " ".join(attrs)

        if astr:
            astr = " " + astr

        return "<RichStatus %s%s>" % ("OK" if self else "BAD", astr)

    def toDict(self):
        d = { 'ok': self.ok }

        for key in self.info.keys():
            d[key] = self.info[key]

        return d

    @classmethod
    def fromError(self, error, **kwargs):
        kwargs['error'] = error
        return RichStatus(False, **kwargs)

    @classmethod
    def OK(self, **kwargs):
        return RichStatus(True, **kwargs)


class EnvoyConfig (object):
    route_template = '''
    {{
        "timeout_ms": 0,
        "prefix": "{url_prefix}",
        "cluster": "{cluster_name}"
    }}
    '''

    # We may append the 'features' element to the cluster definition, too.
    cluster_template = '''
    {{
        "name": "{cluster_name}",
        "connect_timeout_ms": 250,
        "type": "sds",
        "service_name": "{service_name}",
        "lb_type": "round_robin"
    }}
    '''

    self_routes = [
        {
            "timeout_ms": 0,
            "prefix": "/ambassador/",
            "cluster": "ambassador_cluster"
        },
        {
            "timeout_ms": 0,
            "prefix": "/ambassador-config/",
            "prefix_rewrite": "/",
            "cluster": "ambassador_config_cluster"
        }
    ]

    self_clusters = [
        {
            "name": "ambassador_cluster",
            "connect_timeout_ms": 250,
            "type": "static",
            "lb_type": "round_robin",
            "hosts": [
                {
                    "url": "tcp://127.0.0.1:5000"
                }
            ]
        },
        {
            "name": "ambassador_config_cluster",
            "connect_timeout_ms": 250,
            "type": "static",
            "lb_type": "round_robin",
            "hosts": [
                {
                    "url": "tcp://127.0.0.1:8001"
                }
            ]
        }
    ]

    def __init__(self, base_config):
        self.services = {}
        self.base_config = base_config

    def add_service(self, name, prefix, port):
        self.services[name] = {
            'prefix': prefix,
            'port': port
        }

    def write_config(self, path):
        # Generate routes and clusters.
        routes = copy.deepcopy(EnvoyConfig.self_routes)
        clusters = copy.deepcopy(EnvoyConfig.self_clusters)

        logging.info("writing Envoy config to %s" % path)
        logging.info("initial routes: %s" % routes)
        logging.info("initial clusters: %s" % clusters)

        for service_name in self.services.keys():
            service = self.services[service_name]

            service_def = {
                'service_name': service_name,
                'url_prefix': service['prefix'],
                'cluster_name': '%s_cluster' % service_name
            }

            route_json = EnvoyConfig.route_template.format(**service_def)
            route = json.loads(route_json)
            logging.info("add route %s" % route)
            routes.append(route)

            cluster_json = EnvoyConfig.cluster_template.format(**service_def)
            cluster = json.loads(cluster_json)
            logging.info("add cluster %s" % cluster)
            clusters.append(cluster)

        config = copy.deepcopy(self.base_config)

        logging.info("final routes: %s" % routes)
        logging.info("final clusters: %s" % clusters)

        dpath.util.set(
            config,
            "/listeners/0/filters/0/config/route_config/virtual_hosts/0/routes", 
            routes
        )

        dpath.util.set(
            config,
            "/cluster_manager/clusters",
            clusters
        )

        output_file = open(path, "w")

        json.dump(config, output_file, 
                  indent=4, separators=(',',':'), sort_keys=True)
        output_file.write('\n')
        output_file.close()

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

        cursor.execute("SELECT name, prefix, port FROM services ORDER BY name, prefix")

        services = []

        for name, prefix, port in cursor:
            services.append({ 'name': name, 'prefix': prefix, 'port': port })

        return RichStatus.OK(services=services, count=len(services))
    except pg8000.Error as e:
        return RichStatus.fromError("services: could not fetch info: %s" % e)

def handle_service_list(req):
    return fetch_all_services()

def handle_service_get(req, name):
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("SELECT prefix, port FROM services WHERE name = :name", locals())
        [ prefix, port ] = cursor.fetchone()

        return RichStatus.OK(name=name, prefix=prefix, port=port)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not fetch info: %s" % (name, e))

def handle_service_del(req, name):
    try:
        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute("DELETE FROM services WHERE name = :name", locals())
        conn.commit()

        return RichStatus.OK(name=name)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not delete service: %s" % (name, e))

def handle_service_post(req, name):
    try:
        rc = getIncomingJSON(req, 'prefix', 'port')

        logging.debug("handle_service_post %s: got args %s" % (name, rc.toDict()))

        if not rc:
            return rc

        prefix = rc.prefix
        port = int(rc.port)

        logging.debug("handle_service_post %s: prefix %s port %d" % (name, prefix, port))

        conn = get_db("ambassador")
        cursor = conn.cursor()

        cursor.execute('INSERT INTO services VALUES(:name, :prefix, :port)', locals())
        conn.commit()

        return RichStatus.OK(name=name)
    except pg8000.Error as e:
        return RichStatus.fromError("%s: could not save info: %s" % (name, e))

@app.route('/ambassador/health', methods=[ 'GET' ])
def health():
    rc = RichStatus.OK(msg="ambassador health check OK")

    return jsonify(rc.toDict())

def new_config(envoy_base_config, envoy_config_path, envoy_restarter_pid):
    config = EnvoyConfig(envoy_base_config)

    rc = fetch_all_services()

    if rc and rc.services:
        for service in rc.services:
            config.add_service(service['name'], service['prefix'], service['port'])

    config.write_config(envoy_config_path)

    if envoy_restarter_pid > 0:
        os.kill(envoy_restarter_pid, signal.SIGHUP)

    return RichStatus.OK(count=len(rc.services))

@app.route('/ambassador/services', methods=[ 'GET', 'PUT' ])
def root():
    rc = RichStatus.fromError("impossible error")
    logging.debug("handle_services: method %s" % request.method)
    
    try:
        rc = setup()

        if rc:
            if request.method == 'PUT':
                rc = new_config(
                    app.envoy_base_config,      # base config we read earlier
                    app.envoy_config_path,      # where to write full config
                    app.envoy_restarter_pid     # PID to signal for reload
                )
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

    new_config(
        app.envoy_base_config,      # base config we read earlier
        app.envoy_config_path,      # where to write full config
        -1                          # don't signal automagically here
    )

    time.sleep(2)

    logging.info("ambassador asking restarter for initial reread")
    os.kill(app.envoy_restarter_pid, signal.SIGHUP)    

    app.run(host='127.0.0.1', port=5000, debug=True)

if __name__ == '__main__':
    setup()
    main()
