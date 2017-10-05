#!python

import sys

import datetime
import json
import logging

import VERSION

from flask import Flask, render_template, request # Response, jsonify

from AmbassadorConfig import AmbassadorConfig
from envoy import EnvoyStats
from utils import RichStatus, SystemInfo, PeriodicTrigger

__version__ = VERSION.Version
boot_time = datetime.datetime.now()

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s diagd %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

estats = EnvoyStats()

dir_index = 1
health_checks = True
errors = 0

while sys.argv[dir_index].startswith('-'):
    arg = sys.argv[dir_index]
    dir_index += 1

    if arg == '--no-health':
        health_checks = False
    else:
        logging.error("unknown argument %s" % arg)
        errors += 1

if errors:
    sys.exit(errors)

if health_checks:
    logging.debug("Starting periodic updates")
    stats_updater = PeriodicTrigger(estats.update, period=5)

aconf = AmbassadorConfig(sys.argv[dir_index])

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

    return ", ".join(strings)

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
    return { name: estats.cluster_stats(name) for name in cluster_names }

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

ambassador_targets = {
    'mapping': 'https://www.getambassador.io/reference/configuration#mappings',
    'module': 'https://www.getambassador.io/reference/configuration#modules',
}

envoy_targets = {
    'route': 'https://envoyproxy.github.io/envoy/configuration/http_conn_man/route_config/route.html',
    'cluster': 'https://envoyproxy.github.io/envoy/configuration/cluster_manager/cluster.html',
}

app = Flask(__name__)

@app.route('/ambassador/v0/check_alive', methods=[ 'GET' ])
def check_alive():
    status = envoy_status(estats)

    if status['alive']:
        return "ambassador liveness check OK (%s)" % status['uptime'], 200
    else:
        return "ambassador seems to have died (%s)" % status['uptime'], 400

@app.route('/ambassador/v0/check_ready', methods=[ 'GET' ])
def check_ready():
    status = envoy_status(estats)

    if status['ready']:
        return "ambassador readiness check OK (%s)" % status['since_update'], 200
    else:
        return "ambassador not ready (%s)" % status['since_update'], 400

@app.route('/ambassador/v0/diag/', methods=[ 'GET' ])
def show_overview():
    logging.debug("showing overview")

    # Build a set of source _files_ rather than source _objects_.
    source_files = {}
    
    for key, source in aconf.sources.items():
        source_dict = source_files.setdefault(
            source['filename'],
            {
                'filename': source['filename'],
                'objects': {},
                'count': 0,
                'plural': "objects"
            }
        )

        source_dict['count'] += 1
        source_dict['plural'] = "object" if (source_dict['count'] == 1) else "objects"

        object_dict = source_dict['objects']
        object_dict[key] = {
            'key': key,
            'kind': source['kind'],
            'target': ambassador_targets.get(source['kind'].lower())
        }

    routes = [ route for route in aconf.envoy_config['routes']
               if route['_source'] != "--diagnostics--" ]

    clusters = aconf.envoy_config['clusters']

    configuration = { key: aconf.envoy_config[key] for key in aconf.envoy_config.keys()
                      if key != "routes" }

    return render_template('overview.html', system=system_info(), 
                           envoy_status=envoy_status(estats), 
                           cluster_stats=cluster_stats(clusters),
                           sources=sorted(source_files.values(), key=lambda x: x['filename']),
                           routes=routes,
                           **configuration)

@app.route('/ambassador/v0/diag/<path:source>', methods=[ 'GET' ])
def show_intermediate(source=None):
    logging.debug("getting intermediate for '%s'" % source)

    result = aconf.get_intermediate_for(source)

    logging.debug(json.dumps(result, indent=4))

    method = request.args.get('method', None)
    resource = request.args.get('resource', None)

    result['sources'] = sorted_sources(result['sources'])

    ambassador_targets = {
        'mapping': 'https://www.getambassador.io/reference/configuration#mappings',
        'module': 'https://www.getambassador.io/reference/configuration#modules',
    }

    envoy_targets = {
        'route': 'https://envoyproxy.github.io/envoy/configuration/http_conn_man/route_config/route.html',
        'cluster': 'https://envoyproxy.github.io/envoy/configuration/cluster_manager/cluster.html',
    }

    for source in result['sources']:
        if source['kind'].lower() in ambassador_targets:
            source['target'] = ambassador_targets[source['kind'].lower()]

    return render_template('diag.html', 
                           system=system_info(),
                           envoy_status=envoy_status(estats),                         
                           method=method, resource=resource,
                           cluster_stats=cluster_stats(result['clusters']),
                           **result)

@app.template_filter('pretty_json')
def pretty_json(obj):
    if isinstance(obj, dict):
        obj = dict(**obj)

        if '_source' in obj:
            del(obj['_source'])

        if '_referenced_by' in obj:
            del(obj['_referenced_by'])

    return json.dumps(obj, indent=4, sort_keys=True)

app.run(host='127.0.0.1', port=aconf.diag_port(), debug=True)
