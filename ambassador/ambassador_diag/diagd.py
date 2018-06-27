#!python

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import sys

import datetime
import functools
import glob
import json
import logging
import multiprocessing
import os
import re
import signal
import time
import uuid

from pkg_resources import Requirement, resource_filename

import clize
from clize import Parameter
from flask import Flask, render_template, send_from_directory, request, jsonify # Response
import gunicorn.app.base
from gunicorn.six import iteritems

from ambassador.config import Config
from ambassador.VERSION import Version
from ambassador.utils import RichStatus, SystemInfo, PeriodicTrigger

from .envoy import EnvoyStats

def number_of_workers():
    return (multiprocessing.cpu_count() * 2) + 1

__version__ = Version

boot_time = datetime.datetime.now()

logging.basicConfig(
    level=logging.INFO,
    format="%%(asctime)s diagd %s [P%%(process)dT%%(threadName)s] %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# Shut up Werkzeug's standard request logs -- they're just too noisy.
logging.getLogger("werkzeug").setLevel(logging.CRITICAL)

# Likewise make requests a bit quieter.
logging.getLogger("urllib3").setLevel(logging.WARNING)
logging.getLogger("requests").setLevel(logging.WARNING)

ambassador_targets = {
    'mapping': 'https://www.getambassador.io/reference/configuration#mappings',
    'module': 'https://www.getambassador.io/reference/configuration#modules',
}

envoy_targets = {
    'route': 'https://envoyproxy.github.io/envoy/configuration/http_conn_man/route_config/route.html',
    'cluster': 'https://envoyproxy.github.io/envoy/configuration/cluster_manager/cluster.html',
}

######## DECORATORS

def standard_handler(f):
    func_name = getattr(f, '__name__', '<anonymous>')

    @functools.wraps(f)
    def wrapper(*args, **kwds):
        reqid = str(uuid.uuid4()).upper()
        prefix = "%s: %s \"%s %s\"" % (reqid, request.remote_addr, request.method, request.path)

        start = datetime.datetime.now()

        app.logger.debug("%s handler %s" % (prefix, func_name))

        result = ("impossible error", 500)
        status_to_log = 500
        result_to_log = "impossible error"
        result_log_level = logging.ERROR

        try:
            result = f(*args, reqid=reqid, **kwds)
            if not isinstance(result, tuple):
                result = (result, 200)

            status_to_log = result[1]

            if (status_to_log // 100) == 2:
                result_log_level = logging.DEBUG
                result_to_log = "success"
            else:
                result_log_level = logging.ERROR
                result_to_log = "failure"
        except Exception as e:
            result_to_log = "server error"
            status_to_log = 500
            result_log_level = logging.ERROR
            result = (result_to_log, status_to_log)

            app.logger.exception(e)

        end = datetime.datetime.now()
        ms = int(((end - start).total_seconds() * 1000) + .5)

        app.logger.log(result_log_level, "%s %dms %d %s" % (prefix, ms, status_to_log, result_to_log))

        return result

    return wrapper

# Get the Flask app defined early.
app = Flask(__name__,
            template_folder=resource_filename(Requirement.parse("ambassador"), "templates"))

# Next, various helpers.
def aconf(app):
    configs = glob.glob("%s-*" % app.config_dir_prefix)

    # # Test crap
    # configs.append("%s-87-envoy.json" % app.config_dir_prefix)

    if configs:
        keyfunc = lambda x: x.split("-")[-1]
        key_match = lambda x: re.match('^\d+$', keyfunc(x))
        key_as_int = lambda x: int(keyfunc(x))

        configs = sorted(filter(key_match, configs), key=key_as_int)

        latest = configs[-1]
    else:
        latest = app.config_dir_prefix

    aconf = Config(latest)

    uptime = datetime.datetime.now() - boot_time
    hr_uptime = td_format(uptime)

    result = Config.scout_report(mode="diagd", runtime=Config.runtime,
                                 uptime=int(uptime.total_seconds()),
                                 hr_uptime=hr_uptime)

    app.logger.info("Scout reports %s" % json.dumps(result))

    return aconf

def td_format(td_object):
    seconds = int(td_object.total_seconds())
    periods = [
        ('year',   60*60*24*365),
        ('month',  60*60*24*30),
        ('day',    60*60*24),
        ('hour',   60*60),
        ('minute', 60),
        ('second', 1)
    ]

    strings=[]
    for period_name,period_seconds in periods:
        if seconds > period_seconds:
            period_value, seconds = divmod(seconds,period_seconds)

            strings.append("%d %s%s" % 
                           (period_value, period_name, "" if (period_value == 1) else "s"))

    formatted = ", ".join(strings)

    if not formatted:
        formatted = "0s"

    return formatted

def interval_format(seconds, normal_format, now_message):
    if seconds >= 1:
        return normal_format % td_format(datetime.timedelta(seconds=seconds))
    else:
        return now_message

def system_info():
    return {
        "version": __version__,
        "hostname": SystemInfo.MyHostName,
        "boot_time": boot_time,
        "hr_uptime": td_format(datetime.datetime.now() - boot_time)
    }

def cluster_stats(clusters):
    cluster_names = [ x['name'] for x in clusters ]
    return { name: app.estats.cluster_stats(name) for name in cluster_names }

def source_key(source):
    return "%s.%d" % (source['filename'], source['index'])

def sorted_sources(sources):
    return sorted(sources, key=source_key)

def route_cluster_info(route, route_clusters, cluster, cluster_info, type_label):
    c_name = cluster['name']

    c_info = cluster_info.get(c_name, None)

    if not c_info:
        c_info = {
            '_service': 'unknown cluster!',
            '_health': 'unknown cluster!',
            '_hmetric': 'unknown',
            '_hcolor': 'orange'
        }

        if route.get('host_redirect', None):
            c_info['_service'] = route['host_redirect']
            c_info['_hcolor'] = 'grey'

    c_service = c_info.get('_service', 'unknown service!')
    c_health = c_info.get('_hmetric', 'unknown')
    c_color = c_info.get('_hcolor', 'orange')
    c_weight = cluster['weight']

    route_clusters[c_name] = {
        'weight': c_weight,
        '_health': c_health,
        '_hcolor': c_color,
        'service': c_service,
    }

    if type_label:
        route_clusters[c_name]['type_label'] = type_label

def route_and_cluster_info(request, overview, clusters, cstats):
    request_host = request.headers.get('Host', '*')
    request_scheme = request.headers.get('X-Forwarded-Proto', 'http').lower()
    tls_active = request_scheme == 'https'

    cluster_info = { cluster['name']: cluster for cluster in clusters }

    for cluster_name, cstat in cstats.items():
        c_info = cluster_info.setdefault(cluster_name, {
            '_service': 'unknown service!',
        })

        c_info['_health'] = cstat['health']
        c_info['_hmetric'] = cstat['hmetric']
        c_info['_hcolor'] = cstat['hcolor']

    route_info = []

    if 'routes' in overview:
        for route in overview['routes']:
            prefix = route['prefix'] if 'prefix' in route else route['regex']
            rewrite = route.get('prefix_rewrite', "/")
            method = '*'
            host = None

            route_clusters = {}

            for cluster in route['clusters']:
                route_cluster_info(route, route_clusters, cluster, cluster_info, None)

            if 'host_redirect' in route:
                    # XXX Stupid hackery here. redirect_cluster should be a real 
                    # Cluster object.
                    redirect_cluster = {
                        'name': route['host_redirect'],
                        'weight': 100
                    }

                    route_cluster_info(route, route_clusters, redirect_cluster, cluster_info, "redirect")
                    app.logger.info("host_redirect route: %s" % route)
                    app.logger.info("host_redirect clusters: %s" % route_clusters)

            if 'shadow' in route:
                shadow_info = route['shadow']
                shadow_name = shadow_info.get('name', None)

                if shadow_name:
                    # XXX Stupid hackery here. shadow_cluster should be a real
                    # Cluster object.
                    shadow_cluster = {
                        'name': shadow_name,
                        'weight': 100
                    }

                    route_cluster_info(route, route_clusters, shadow_cluster, cluster_info, "shadow")

            headers = []

            for header in route.get('headers', []):
                hdr_name = header.get('name', None)
                hdr_value = header.get('value', None)

                if hdr_name == ':authority':
                    host = hdr_value
                elif hdr_name == ':method':
                    method = hdr_value
                else:
                    headers.append(header)

            sep = "" if prefix.startswith("/") else "/"

            route_key = "%s://%s%s%s" % (request_scheme, host if host else request_host, sep, prefix)

            route_info.append({
                '_route': route,
                '_source': route['_source'],
                '_group_id': route['_group_id'],
                'key': route_key,
                'prefix': prefix,
                'rewrite': rewrite,
                'method': method,
                'headers': headers,
                'clusters': route_clusters,
                'host': host if host else '*'
            })

        # app.logger.info("route_info")
        # app.logger.info(json.dumps(route_info, indent=4, sort_keys=True))

        # app.logger.info("cstats")
        # app.logger.info(json.dumps(cstats, indent=4, sort_keys=True))

    return route_info, cluster_info

def envoy_status(estats):
    since_boot = interval_format(estats.time_since_boot(), "%s", "less than a second")

    since_update = "Never updated"

    if estats.time_since_update():
        since_update = interval_format(estats.time_since_update(), "%s ago", "within the last second")

    return {
        "alive": estats.is_alive(),
        "ready": estats.is_ready(),
        "uptime": since_boot,
        "since_update": since_update
    }

def clean_notices(notices):
    cleaned = []

    for notice in notices:
        try:
            if isinstance(notice, str):
                cleaned.append({ "level": "WARNING", "message": notice })
            else:
                lvl = notice['level'].upper()
                msg = notice['message']

                cleaned.append({ "level": lvl, "message": msg })
        except KeyError:
            cleaned.append({ "level": "WARNING", "message": json.dumps(notice) })
        except:
            cleaned.append({ "level": "ERROR", "message": json.dumps(notice) })

    return cleaned

@app.route('/ambassador/v0/favicon.ico', methods=[ 'GET' ])
def favicon():
    template_path = resource_filename(Requirement.parse("ambassador"), "templates")

    return send_from_directory(template_path, "favicon.ico")

@app.route('/ambassador/v0/check_alive', methods=[ 'GET' ])
def check_alive():
    status = envoy_status(app.estats)

    if status['alive']:
        return "ambassador liveness check OK (%s)" % status['uptime'], 200
    else:
        return "ambassador seems to have died (%s)" % status['uptime'], 503

@app.route('/ambassador/v0/check_ready', methods=[ 'GET' ])
def check_ready():
    status = envoy_status(app.estats)

    if status['ready']:
        return "ambassador readiness check OK (%s)" % status['since_update'], 200
    else:
        return "ambassador not ready (%s)" % status['since_update'], 503

@app.route('/ambassador/v0/diag/', methods=[ 'GET' ])
@standard_handler
def show_overview(reqid=None):
    app.logger.debug("OV %s - showing overview" % reqid)

    notices = []
    loglevel = request.args.get('loglevel', None)

    if loglevel:
        app.logger.debug("OV %s -- requesting loglevel %s" % (reqid, loglevel))

        if not app.estats.update_log_levels(time.time(), level=loglevel):
            notices = [ "Could not update log level!" ]
        # else:
        #     return redirect("/ambassador/v0/diag/", code=302)

    ov = aconf(app).diagnostic_overview()
    clusters = ov['clusters']
    cstats = cluster_stats(clusters)

    route_info, cluster_info = route_and_cluster_info(request, ov, clusters, cstats)

    notices.extend(clean_notices(Config.scout_notices))

    errors = []

    for source in ov['sources']:
        for obj in source['objects'].values():
            obj['target'] = ambassador_targets.get(obj['kind'].lower(), None)

            if obj['errors']:
                errors.extend([ (obj['key'], error['summary'])
                                 for error in obj['errors'] ])

    tvars = dict(system=system_info(), 
                 envoy_status=envoy_status(app.estats), 
                 loginfo=app.estats.loginfo,
                 cluster_stats=cstats,
                 notices=notices,
                 errors=errors,
                 route_info=route_info,
                 **ov)

    if request.args.get('json', None):
        result = jsonify(tvars)
    else:
        return render_template("overview.html", **tvars)

    # app.logger.debug("OV %s from %s --- rendering complete" % (reqid, request.remote_addr))

    return result

@app.route('/ambassador/v0/diag/<path:source>', methods=[ 'GET' ])
@standard_handler
def show_intermediate(source=None, reqid=None):
    app.logger.debug("SRC %s - getting intermediate for '%s'" % (reqid, source))

    result = aconf(app).get_intermediate_for(source)

    # app.logger.debug("result\n%s" % json.dumps(result, indent=4, sort_keys=True))

    method = request.args.get('method', None)
    resource = request.args.get('resource', None)
    route_info = None
    errors = []

    if "error" not in result:
        clusters = result['clusters']
        cstats = cluster_stats(clusters)

        route_info, cluster_info = route_and_cluster_info(request, result, clusters, cstats)

        result['cluster_stats'] = cstats
        result['sources'] = sorted_sources(result['sources'])
        result['source_dict'] = { source_key(source): source 
                                  for source in result['sources']}

        for source in result['sources']:
            source['target'] = ambassador_targets.get(source['kind'].lower(), None)

            if source['errors']:
                errors.extend([ (source['filename'], error['summary'])
                                 for error in source['errors'] ])

    tvars = dict(system=system_info(),
                 envoy_status=envoy_status(app.estats),
                 loginfo=app.estats.loginfo,
                 method=method, resource=resource,
                 route_info=route_info,
                 errors=errors,
                 notices=clean_notices(Config.scout_notices),
                 **result)

    if request.args.get('json', None):
        return jsonify(tvars)
    else:
        return render_template("diag.html", **tvars)

@app.template_filter('pretty_json')
def pretty_json(obj):
    if isinstance(obj, dict):
        obj = dict(**obj)

        keys_to_drop = [ key for key in obj.keys() if key.startswith('_') ]

        for key in keys_to_drop:
            del(obj[key])

        # if '_source' in obj:
        #     del(obj['_source'])

        # if '_referenced_by' in obj:
        #     del(obj['_referenced_by'])

    return json.dumps(obj, indent=4, sort_keys=True)

@app.template_filter('sort_clusters_by_service')
def sort_clusters_by_service(clusters):
    return sorted([ c for c in clusters.values() ], key=lambda x: x['service'])

@app.template_filter('source_lookup')
def source_lookup(name, sources):
    app.logger.info("%s => sources %s" % (name, sources))

    source = sources.get(name, {})

    app.logger.info("%s => source %s" % (name, source))

    return source.get('_source', name)

def create_diag_app(config_dir_path, do_checks=False, debug=False, verbose=False):
    app.estats = EnvoyStats()
    app.health_checks = False
    app.debugging = debug

    # This feels like overkill.
    app._logger = logging.getLogger(app.logger_name)
    app.logger.setLevel(logging.INFO)

    if app.debugging or verbose:
        app.logger.setLevel(logging.DEBUG)
        logging.getLogger().setLevel(logging.DEBUG)
    else:
        logging.getLogger("ambassador.config").setLevel(logging.INFO)

    if do_checks:
        app.health_checks = True

    app.config_dir_prefix = config_dir_path

    return app

class StandaloneApplication(gunicorn.app.base.BaseApplication):
    def __init__(self, app, options=None):
        self.options = options or {}
        self.application = app
        super(StandaloneApplication, self).__init__()

    def load_config(self):
        config = dict([(key, value) for key, value in iteritems(self.options)
                       if key in self.cfg.settings and value is not None])
        for key, value in iteritems(config):
            self.cfg.set(key.lower(), value)

    def load(self):
        if self.application.health_checks:
            self.application.logger.info("Starting periodic updates")
            self.application.stats_updater = PeriodicTrigger(self.application.estats.update, period=5)

        return self.application


def _main(config_dir_path:Parameter.REQUIRED, *, no_checks=False, no_debugging=False, verbose=False,
          workers=None, port=8877, host='0.0.0.0'):
    """
    Run the diagnostic daemon.

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param no_checks: If True, don't do Envoy-cluster health checking
    :param no_debugging: If True, don't run Flask in debug mode
    :param verbose: If True, be more verbose
    :param workers: Number of workers; default is based on the number of CPUs present
    :param host: Interface on which to listen (default 0.0.0.0)
    :param port: Port on which to listen (default 8877)
    """
    
    # Create the application itself.
    flask_app = create_diag_app(config_dir_path, not no_checks, not no_debugging, verbose)

    if workers == None:
        workers = number_of_workers()

    gunicorn_config = {
        'bind': '%s:%s' % (host, port),
        # 'workers': 1,
        'threads': workers,
    }

    app.logger.info("thread count %d, listening on %s" % (gunicorn_config['threads'], gunicorn_config['bind']))

    StandaloneApplication(flask_app, gunicorn_config).run()

def main():
    clize.run(_main)

if __name__ == "__main__":
    main()
