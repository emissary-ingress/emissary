#!python

import sys

import datetime
import functools
import glob
import json
import logging
import os
import signal
import time
import uuid

from pkg_resources import Requirement, resource_filename

import clize
from clize import Parameter
from flask import Flask, render_template, request, jsonify # Response

from ambassador.config import Config
from ambassador.VERSION import Version
from ambassador.utils import RichStatus, SystemInfo, PeriodicTrigger

from .envoy import EnvoyStats

__version__ = Version

boot_time = datetime.datetime.now()
last_scout_update = datetime.datetime.now() - datetime.timedelta(hours=24)

logging.basicConfig(
    level=logging.INFO,
    format="%%(asctime)s diagd %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# Shut up Werkzeug's standard request logs -- they're just too noisy.
logging.getLogger("werkzeug").setLevel(logging.CRITICAL)

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

    if configs:
        configs.sort(key=lambda x: int(x.split("-")[-1]))
        latest = configs[-1]
    else:
        latest = app.config_dir_prefix

    aconf = Config(latest)

    # How long since the last Scout update? If it's been more than an hour, 
    # check Scout again.

    now = datetime.datetime.now()

    if (now - last_scout_update) > datetime.timedelta(hours=1):
        uptime = now - boot_time
        hr_uptime = td_format(uptime)

        result = Config.scout_report(mode="diagd", runtime=Config.runtime,
                                     uptime=int(uptime.total_seconds()),
                                     hr_uptime=hr_uptime)

        app.logger.debug("Scout reports %s" % json.dumps(result))

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

def sorted_sources(sources):
    return sorted(sources, key=lambda x: "%s.%d" % (x['filename'], x['index']))

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

    ov = aconf(app).diagnostic_overview()
    cstats = cluster_stats(ov['clusters'])
    del(ov['clusters'])

    for source in ov['sources']:
        for obj in source['objects'].values():
            obj['target'] = ambassador_targets.get(obj['kind'].lower(), None)

    tvars = dict(system=system_info(), 
                 envoy_status=envoy_status(app.estats), 
                 cluster_stats=cstats,
                 notices=clean_notices(Config.scout_notices),
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

    # app.logger.debug(json.dumps(result, indent=4))

    method = request.args.get('method', None)
    resource = request.args.get('resource', None)

    if "error" not in result:
        result['cluster_stats'] = cluster_stats(result['clusters'])
        result['sources'] = sorted_sources(result['sources'])

        for source in result['sources']:
            source['target'] = ambassador_targets.get(source['kind'].lower(), None)

    tvars = dict(system=system_info(),
                 envoy_status=envoy_status(app.estats),
                 method=method, resource=resource,
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

        if '_source' in obj:
            del(obj['_source'])

        if '_referenced_by' in obj:
            del(obj['_referenced_by'])

    return json.dumps(obj, indent=4, sort_keys=True)

def _main(config_dir_path:Parameter.REQUIRED, *, no_checks=False, no_debugging=False, verbose=False):
    """
    Run the diagnostic daemon.

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param no_checks: If True, don't do Envoy-cluster health checking
    :param no_debugging: If True, don't run Flask in debug mode
    :param verbose: If True, be more verbose
    """

    app.estats = EnvoyStats()
    app.health_checks = False
    app.debugging = not no_debugging

    # This feels like overkill.
    app._logger = logging.getLogger(app.logger_name)
    app.logger.setLevel(logging.INFO)

    if app.debugging or verbose:
        app.logger.setLevel(logging.DEBUG)
        logging.getLogger().setLevel(logging.DEBUG)

    if not no_checks:
        app.health_checks = True
        app.logger.debug("Starting periodic updates")
        app.stats_updater = PeriodicTrigger(app.estats.update, period=5)

    app.config_dir_prefix = config_dir_path

    app.run(host='0.0.0.0', port=aconf(app).diag_port(), debug=app.debugging)

def main():
    clize.run(_main)

if __name__ == "__main__":
    main()
