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
import copy
import subprocess
from typing import Any, Callable, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

import datetime
import functools
import json
import logging
import multiprocessing
import os
import queue
import re
import signal
import threading
import time
import uuid
import requests
import jsonpatch

from expiringdict import ExpiringDict

import concurrent.futures

from pkg_resources import Requirement, resource_filename

import clize
from clize import Parameter
from flask import Flask, render_template, send_from_directory, request, jsonify, Response
from flask import json as flask_json
import gunicorn.app.base
from gunicorn.six import iteritems

from ambassador import Config, IR, EnvoyConfig, Diagnostics, Scout, Version
from ambassador.utils import SystemInfo, PeriodicTrigger, SavedSecret, load_url_contents
from ambassador.utils import SecretHandler, KubewatchSecretHandler, FSSecretHandler
from ambassador.config.resourcefetcher import ResourceFetcher

from ambassador.diagnostics import EnvoyStats

from ambassador.constants import Constants

if TYPE_CHECKING:
    from ambassador.ir.irtlscontext import IRTLSContext

__version__ = Version

boot_time = datetime.datetime.now()

# allows 10 concurrent users, with a request timeout of 60 seconds
tvars_cache = ExpiringDict(max_len=10, max_age_seconds=60)

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

# envoy_targets = {
#     'route': 'https://envoyproxy.github.io/envoy/configuration/http_conn_man/route_config/route.html',
#     'cluster': 'https://envoyproxy.github.io/envoy/configuration/cluster_manager/cluster.html',
# }


def number_of_workers():
    return (multiprocessing.cpu_count() * 2) + 1


class DiagApp (Flask):
    ambex_pid: int
    kick: Optional[str]
    estats: EnvoyStats
    config_path: Optional[str]
    snapshot_path: str
    bootstrap_path: str
    ads_path: str
    health_checks: bool
    no_envoy: bool
    debugging: bool
    allow_fs_commands: bool
    report_action_keys: bool
    verbose: bool
    notice_path: str
    logger: logging.Logger
    aconf: Config
    ir: Optional[IR]
    econf: Optional[EnvoyConfig]
    diag: Optional[Diagnostics]
    notices: 'Notices'
    scout: Scout
    watcher: 'AmbassadorEventWatcher'
    stats_updater: Optional[PeriodicTrigger]
    scout_checker: Optional[PeriodicTrigger]
    last_request_info: Dict[str, int]
    last_request_time: Optional[datetime.datetime]
    latest_snapshot: str
    banner_endpoint: Optional[str]

    def setup(self, snapshot_path: str, bootstrap_path: str, ads_path: str,
              config_path: Optional[str], ambex_pid: int, kick: Optional[str], banner_endpoint: Optional[str],
              k8s=False, do_checks=True, no_envoy=False, reload=False, debug=False, verbose=False,
              notices=None, validation_retries=5, allow_fs_commands=False, local_scout=False,
              report_action_keys=False):
        self.estats = EnvoyStats()
        self.health_checks = do_checks
        self.no_envoy = no_envoy
        self.debugging = reload
        self.verbose = verbose
        self.notice_path = notices
        self.notices = Notices(self.notice_path)
        self.notices.reset()
        self.k8s = k8s
        self.validation_retries = validation_retries
        self.allow_fs_commands = allow_fs_commands
        self.local_scout = local_scout
        self.report_action_keys = report_action_keys
        self.banner_endpoint = banner_endpoint

        # This will raise an exception and crash if you pass it a string. That's intentional.
        self.ambex_pid = int(ambex_pid)
        self.kick = kick

        # This feels like overkill.
        self.logger = logging.getLogger("ambassador.diagd")
        self.logger.setLevel(logging.INFO)

        self.kubestatus = KubeStatus()

        if debug:
            self.logger.setLevel(logging.DEBUG)
            logging.getLogger('ambassador').setLevel(logging.DEBUG)

        self.config_path = config_path
        self.bootstrap_path = bootstrap_path
        self.ads_path = ads_path
        self.snapshot_path = snapshot_path

        self.ir = None
        self.econf = None
        self.diag = None

        self.stats_updater = None
        self.scout_checker = None

        self.last_request_info = {}
        self.last_request_time = None

        # self.scout = Scout(update_frequency=datetime.timedelta(seconds=10))
        self.scout = Scout(local_only=self.local_scout)

    def check_scout(self, what: str) -> None:
        self.watcher.post("SCOUT", (what, self.ir))


# get the "templates" directory, or raise "FileNotFoundError" if not found
def get_templates_dir():
    res_dir = None
    try:
        # this will fail when not in a distribution
        res_dir = resource_filename(Requirement.parse("ambassador"), "templates")
    except:
        pass

    maybe_dirs = [
        res_dir,
        os.path.join(os.path.dirname(__file__), "..", "templates")
    ]
    for d in maybe_dirs:
        if d and os.path.isdir(d):
            return d
    raise FileNotFoundError


# Get the Flask app defined early. Setup happens later.
app = DiagApp(__name__, template_folder=get_templates_dir())


######## DECORATORS

def standard_handler(f):
    func_name = getattr(f, '__name__', '<anonymous>')

    @functools.wraps(f)
    def wrapper(*args, **kwds):
        reqid = str(uuid.uuid4()).upper()
        prefix = "%s: %s \"%s %s\"" % (reqid, request.remote_addr, request.method, request.path)

        app.logger.info("%s START" % prefix)

        start = datetime.datetime.now()

        app.logger.debug("%s handler %s" % (prefix, func_name))

        # Getting elements in the `tvars_cache` will make sure eviction happens on `max_age_seconds` TTL
        # for removed patch_client rather than waiting to fill `max_len`.
        # Looping over a copied list of keys, to prevent mutating tvars_cache while iterating.
        for k in list(tvars_cache.keys()):
            tvars_cache.get(k)

        # Default to the exception case
        result_to_log = "server error"
        status_to_log = 500
        result_log_level = logging.ERROR
        result = Response(result_to_log, status_to_log)

        try:
            result = f(*args, reqid=reqid, **kwds)
            if not isinstance(result, Response):
                result = Response(f"Invalid handler result {result}", status_to_log)

            status_to_log = result.status_code

            if (status_to_log // 100) == 2:
                result_log_level = logging.INFO
                result_to_log = "success"
            else:
                result_log_level = logging.ERROR
                result_to_log = "failure"
        except Exception as e:
            app.logger.exception(e)

        end = datetime.datetime.now()
        ms = int(((end - start).total_seconds() * 1000) + .5)

        app.logger.log(result_log_level, "%s %dms %d %s" % (prefix, ms, status_to_log, result_to_log))

        return result

    return wrapper


######## UTILITIES


class Notices:
    def __init__(self, local_config_path: str) -> None:
        self.local_path = local_config_path
        self.notices: List[Dict[str, str]] = []

    def reset(self):
        local_notices: List[Dict[str, str]] = []
        local_data = ''

        try:
            local_stream = open(self.local_path, "r")
            local_data = local_stream.read()
            local_notices = json.loads(local_data)
        except OSError:
            pass
        except:
            local_notices.append({ 'level': 'ERROR', 'message': 'bad local notices: %s' % local_data })

        self.notices = local_notices
        # app.logger.info("Notices: after RESET: %s" % json.dumps(self.notices))

    def post(self, notice):
        # app.logger.debug("Notices: POST %s" % notice)
        self.notices.append(notice)
        # app.logger.info("Notices: after POST: %s" % json.dumps(self.notices))

    def prepend(self, notice):
        # app.logger.debug("Notices: PREPEND %s" % notice)
        self.notices.insert(0, notice)
        # app.logger.info("Notices: after PREPEND: %s" % json.dumps(self.notices))

    def extend(self, notices):
        for notice in notices:
            self.post(notice)


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

    strings = []
    for period_name, period_seconds in periods:
        if seconds > period_seconds:
            period_value, seconds = divmod(seconds, period_seconds)

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


def system_info(app):
    ir = app.ir
    debug_mode = False
    
    if ir:
        amod = ir.ambassador_module
        debug_mode = amod.get('debug_mode', False)

        app.logger.info(f'DEBUG_MODE {debug_mode}')

    status_dict = {'config failure': [False, 'no configuration loaded']}

    env_status = getattr(app.watcher, 'env_status', None)

    if env_status:
        status_dict = env_status.to_dict()
        print(f"status_dict {status_dict}")

    return {
        "version": __version__,
        "hostname": SystemInfo.MyHostName,
        "ambassador_id": Config.ambassador_id,
        "ambassador_namespace": Config.ambassador_namespace,
        "single_namespace": Config.single_namespace,
        "knative_enabled": os.environ.get('AMBASSADOR_KNATIVE_SUPPORT', '').lower() == 'true',
        "statsd_enabled": os.environ.get('STATSD_ENABLED', '').lower() == 'true',
        "endpoints_enabled": Config.enable_endpoints,
        "cluster_id": os.environ.get('AMBASSADOR_CLUSTER_ID',
                                     os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")),
        "boot_time": boot_time,
        "hr_uptime": td_format(datetime.datetime.now() - boot_time),
        "latest_snapshot": app.latest_snapshot,
        "env_good": getattr(app.watcher, 'env_good', False),
        "env_failures": getattr(app.watcher, 'failure_list', [ 'no IR loaded' ]),
        "env_status": status_dict,
        "debug_mode": debug_mode
    }


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


def drop_serializer_key(d: Dict[Any, Any]) -> Dict[Any, Any]:
    """
    Delete the "serialization" key (if present) in any dictionary passed in and
    return that dictionary. This function is intended to be used as the
    object_hook value for json.load[s].
    """
    _ = d.pop("serialization", None)
    return d


def filter_keys(d: Dict[Any, Any], keys_to_keep):
    unwanted_keys = set(d) - set(keys_to_keep)
    for unwanted_key in unwanted_keys: del d[unwanted_key]


def filter_webui(d: Dict[Any, Any]):
    filter_keys(d, ['system', 'route_info', 'source_map',
                    'ambassador_resolvers', 'ambassador_services',
                    'envoy_status', 'cluster_stats', 'loginfo', 'errors'])
    for ambassador_resolver in d['ambassador_resolvers']:
        filter_keys(ambassador_resolver, ['_source', 'kind'])
    for route_info in d['route_info']:
        filter_keys(route_info, ['diag_class', 'key', 'headers',
                                 'precedence', 'clusters'])
        for cluster in route_info['clusters']:
            filter_keys(cluster, ['_hcolor', 'type_label', 'service', 'weight'])


@app.route('/_internal/v0/ping', methods=[ 'GET' ])
def handle_ping():
    return "ACK\n", 200


@app.route('/_internal/v0/update', methods=[ 'POST' ])
def handle_kubewatch_update():
    url = request.args.get('url', None)

    if not url:
        app.logger.error("error: update requested with no URL")
        return "error: update requested with no URL\n", 400

    app.logger.info("Update requested: kubewatch, %s" % url)

    status, info = app.watcher.post('CONFIG', ( 'kw', url ))

    return info, status


@app.route('/_internal/v0/watt', methods=[ 'POST' ])
def handle_watt_update():
    url = request.args.get('url', None)

    if not url:
        app.logger.error("error: watt update requested with no URL")
        return "error: watt update requested with no URL\n", 400

    app.logger.info("Update requested: watt, %s" % url)

    status, info = app.watcher.post('CONFIG', ( 'watt', url ))

    return info, status


@app.route('/_internal/v0/fs', methods=[ 'POST' ])
def handle_fs():
    path = request.args.get('path', None)

    if not path:
        app.logger.error("error: update requested with no PATH")
        return "error: update requested with no PATH\n", 400

    app.logger.info("Update requested from %s" % path)

    status, info = app.watcher.post('CONFIG_FS', path)

    return info, status


@app.route('/_internal/v0/events', methods=[ 'GET' ])
def handle_events():
    if not app.local_scout:
        return 'Local Scout is not enabled\n', 400

    event_dump = [
        ( x['local_scout_timestamp'], x['mode'], x['action'], x ) for x in app.scout._scout.events
    ]

    app.logger.info(f'Event dump {event_dump}')

    return jsonify(event_dump)


@app.route('/ambassador/v0/favicon.ico', methods=[ 'GET' ])
def favicon():
    template_path = resource_filename(Requirement.parse("ambassador"), "templates")

    return send_from_directory(template_path, "favicon.ico")


@app.route('/ambassador/v0/check_alive', methods=[ 'GET' ])
def check_alive():
    status = envoy_status(app.estats)

    if status['alive']:
        return "ambassador liveness check OK (%s)\n" % status['uptime'], 200
    else:
        return "ambassador seems to have died (%s)\n" % status['uptime'], 503


@app.route('/ambassador/v0/check_ready', methods=[ 'GET' ])
def check_ready():
    if not (app.ir and app.diag):
        return "ambassador waiting for config\n", 503

    status = envoy_status(app.estats)

    if status['ready']:
        return "ambassador readiness check OK (%s)\n" % status['since_update'], 200
    else:
        return "ambassador not ready (%s)\n" % status['since_update'], 503


@app.route('/ambassador/v0/diag/', methods=[ 'GET' ])
@standard_handler
def show_overview(reqid=None):
    app.logger.debug("OV %s - showing overview" % reqid)

    diag = app.diag

    if app.verbose:
        app.logger.debug("OV %s: DIAG" % reqid)
        app.logger.debug("%s" % json.dumps(diag.as_dict(), sort_keys=True, indent=4))

    ov = diag.overview(request, app.estats)

    if app.verbose:
        app.logger.debug("OV %s: OV" % reqid)
        app.logger.debug("%s" % json.dumps(ov, sort_keys=True, indent=4))
        app.logger.debug("OV %s: collecting errors" % reqid)

    ddict = collect_errors_and_notices(request, reqid, "overview", diag)

    banner_content = None
    if app.banner_endpoint and app.ir and app.ir.edge_stack_allowed:
        try:
            response = requests.get(app.banner_endpoint)
            if response.status_code == 200:
                banner_content = response.text
        except Exception as e:
            app.logger.error("could not get banner_content: %s" % e)

    tvars = dict(system=system_info(app),
                 envoy_status=envoy_status(app.estats),
                 loginfo=app.estats.loginfo,
                 notices=app.notices.notices,
                 banner_content=banner_content,
                 **ov, **ddict)

    patch_client = request.args.get('patch_client', None)
    if request.args.get('json', None):
        filter_key = request.args.get('filter', None)

        if filter_key == 'webui':
            filter_webui(tvars)
        elif filter_key:
            return jsonify(tvars.get(filter_key, None))

        if patch_client:
            # Assume this is the Admin UI. Recursively drop all "serialization"
            # keys. This avoids leaking secrets and generally makes the
            # snapshot a lot smaller without losing information that the Admin
            # UI cares about. We do this below by setting the object_hook
            # parameter of the json.loads(...) call.

            # Get the previous full representation
            cached_tvars_json = tvars_cache.get(patch_client, dict())
            # Serialize the tvars into a json-string using the same jsonify Flask serializer, then load the json object
            response_content = json.loads(flask_json.dumps(tvars), object_hook=drop_serializer_key)
            # Diff between the previous representation and the current full representation  (http://jsonpatch.com/)
            patch = jsonpatch.make_patch(cached_tvars_json, response_content)
            # Save the current full representation in memory
            tvars_cache[patch_client] = response_content

            # Return only the diff
            return Response(patch.to_string(), mimetype="application/json")
        else:
            return jsonify(tvars)
    else:
        app.check_scout("overview")
        return Response(render_template("overview.html", **tvars))


def collect_errors_and_notices(request, reqid, what: str, diag: Diagnostics) -> Dict:
    loglevel = request.args.get('loglevel', None)
    notice = None

    if loglevel:
        app.logger.debug("%s %s -- requesting loglevel %s" % (what, reqid, loglevel))

        if not app.estats.update_log_levels(time.time(), level=loglevel):
            notice = { 'level': 'WARNING', 'message': "Could not update log level!" }
        # else:
        #     return redirect("/ambassador/v0/diag/", code=302)

    # We need to grab errors and notices from diag.as_dict(), process the errors so
    # they work for the HTML rendering, and post the notices to app.notices. Then we
    # return the dict representation that our caller should work with.

    ddict = diag.as_dict()

    # app.logger.debug("ddict %s" % json.dumps(ddict, indent=4, sort_keys=True))

    derrors = ddict.pop('errors', {})

    errors = []

    for err_key, err_list in derrors.items():
        if err_key == "-global-":
            err_key = ""

        for err in err_list:
            errors.append((err_key, err[ 'error' ]))

    dnotices = ddict.pop('notices', {})

    # Make sure that anything about the loglevel gets folded into this set.
    if notice:
        app.notices.prepend(notice)

    for notice_key, notice_list in dnotices.items():
        for notice in notice_list:
            app.notices.post({'level': 'NOTICE', 'message': "%s: %s" % (notice_key, notice)})

    ddict['errors'] = errors

    return ddict


@app.route('/ambassador/v0/diag/<path:source>', methods=[ 'GET' ])
@standard_handler
def show_intermediate(source=None, reqid=None):
    app.logger.debug("SRC %s - getting intermediate for '%s'" % (reqid, source))

    diag = app.diag

    method = request.args.get('method', None)
    resource = request.args.get('resource', None)

    result = diag.lookup(request, source, app.estats)

    if app.verbose:
        app.logger.debug("RESULT %s" % json.dumps(result, sort_keys=True, indent=4))

    ddict = collect_errors_and_notices(request, reqid, "detail %s" % source, diag)

    tvars = dict(system=system_info(app),
                 envoy_status=envoy_status(app.estats),
                 loginfo=app.estats.loginfo,
                 notices=app.notices.notices,
                 method=method, resource=resource,
                 **result, **ddict)

    if request.args.get('json', None):
        key = request.args.get('filter', None)

        if key:
            return jsonify(tvars.get(key, None))
        else:
            return jsonify(tvars)
    else:
        app.check_scout("detail: %s" % source)
        return Response(render_template("diag.html", **tvars))


@app.template_filter('sort_by_key')
def sort_by_key(objects):
    return sorted(objects, key=lambda x: x['key'])


@app.template_filter('pretty_json')
def pretty_json(obj):
    if isinstance(obj, dict):
        obj = dict(**obj)

        keys_to_drop = [ key for key in obj.keys() if key.startswith('_') ]

        for key in keys_to_drop:
            del(obj[key])

    return json.dumps(obj, indent=4, sort_keys=True)


@app.template_filter('sort_clusters_by_service')
def sort_clusters_by_service(clusters):
    return sorted(clusters, key=lambda x: x['service'])
    # return sorted([ c for c in clusters.values() ], key=lambda x: x['service'])


@app.template_filter('source_lookup')
def source_lookup(name, sources):
    app.logger.info("%s => sources %s" % (name, sources))

    source = sources.get(name, {})

    app.logger.info("%s => source %s" % (name, source))

    return source.get('_source', name)


@app.route('/metrics', methods=['GET'])
@standard_handler
def get_prometheus_metrics(*args, **kwargs):
    return app.estats.get_prometheus_state()


def bool_fmt(b: bool) -> str:
    return 'T' if b else 'F'


class StatusInfo:
    def __init__(self) -> None:
        self.status = True
        self.specifics: List[Tuple[bool, str]] = []

    def failure(self, message: str) -> None:
        self.status = False
        self.specifics.append((False, message))

    def OK(self, message: str) -> None:
        self.specifics.append((True, message))

    def to_dict(self) -> Dict[str, Union[bool, List[Tuple[bool, str]]]]:
        return {
            'status': self.status,
            'specifics': self.specifics
        }

class SystemStatus:
    def __init__(self) -> None:
        self.status: Dict[str, StatusInfo] = {}

    def failure(self, key: str, message: str) -> None:
        self.info_for_key(key).failure(message)

    def OK(self, key: str, message: str) -> None:
        self.info_for_key(key).OK(message)

    def info_for_key(self, key) -> StatusInfo:
        if key not in self.status:
            self.status[key] = StatusInfo()

        return self.status[key]

    def to_dict(self) -> Dict[str, Dict[str, Union[bool, List[Tuple[bool, str]]]]]:
        return { key: info.to_dict() for key, info in self.status.items() }


class KubeStatus:
    pool: concurrent.futures.ProcessPoolExecutor

    def __init__(self) -> None:
        self.live: Dict[str,  bool] = {}
        self.current_status: Dict[str, str] = {}
        self.pool = concurrent.futures.ProcessPoolExecutor(max_workers=5)

    def mark_live(self, kind: str, name: str, namespace: str) -> None:
        key = f"{kind}/{name}.{namespace}"

        # print(f"KubeStatus MASTER {os.getpid()}: mark_live {key}")
        self.live[key] = True

    def prune(self) -> None:
        drop: List[str] = []

        for key in self.current_status.keys():
            if not self.live.get(key, False):
                drop.append(key)

        for key in drop:
            # print(f"KubeStatus MASTER {os.getpid()}: prune {key}")
            del(self.current_status[key])

        self.live = {}

    def post(self, kind: str, name: str, namespace: str, text: str) -> None:
        key = f"{kind}/{name}.{namespace}"
        extant = self.current_status.get(key, None)

        if extant == text:
            # print(f"KubeStatus MASTER {os.getpid()}: {key} == {text}")
            pass
        else:
            # print(f"KubeStatus MASTER {os.getpid()}: {key} needs {text}")

            # For now we're going to assume that this works.
            self.current_status[key] = text
            f = self.pool.submit(kubestatus_update, kind, name, namespace, text)
            f.add_done_callback(kubestatus_update_done)


def kubestatus_update(kind: str, name: str, namespace: str, text: str) -> str:
    cmd = [ 'kubestatus', kind, '-f', f'metadata.name={name}', '-n', namespace, '-u', '/dev/fd/0' ]
    print(f"KubeStatus UPDATE {os.getpid()}: running command: {cmd}")

    try:
        rc = subprocess.run(cmd, input=text.encode('utf-8'), timeout=5)

        if rc.returncode == 0:
            return f"{name}.{namespace}: update OK"
        else:
            return f"{name}.{namespace}: error {rc.returncode}"

    except subprocess.TimeoutExpired as e:
        return f"{name}.{namespace}: timed out"

def kubestatus_update_done(f: concurrent.futures.Future) -> None:
    print(f"KubeStatus DONE {os.getpid()}: result {f.result()}")


class AmbassadorEventWatcher(threading.Thread):
    # The key for 'Actions' is chimed - chimed_ok - env_good. This will make more sense
    # if you read through the _load_ir method.

    Actions = {
        'F-F-F': ( 'unhealthy',     True  ),    # make sure the first chime always gets out
        'F-F-T': ( 'now-healthy',   True  ),    # make sure the first chime always gets out
        'F-T-F': ( 'now-unhealthy', True  ),    # this is actually impossible
        'F-T-T': ( 'healthy',       True  ),    # this is actually impossible
        'T-F-F': ( 'unhealthy',     False ),
        'T-F-T': ( 'now-healthy',   True  ),
        'T-T-F': ( 'now-unhealthy', True  ),
        'T-T-T': ( 'update',        False ),
    }

    def __init__(self, app: DiagApp) -> None:
        super().__init__(name="AEW", daemon=True)
        self.app = app
        self.logger = self.app.logger
        self.events: queue.Queue = queue.Queue()

        self.chimed = False         # Have we ever sent a chime about the environment?
        self.last_chime = False     # What was the status of our last chime? (starts as False)
        self.env_good = False       # Is our environment currently believed to be OK?
        self.failure_list: List[str] = [ 'unhealthy at boot' ]     # What's making our environment not OK?

    def post(self, cmd: str, arg: Union[str, Tuple[str, Optional[IR]]]) -> Tuple[int, str]:
        rqueue: queue.Queue = queue.Queue()

        self.events.put((cmd, arg, rqueue))

        return rqueue.get()

    def update_estats(self) -> None:
        self.post('ESTATS', '')

    def run(self):
        self.logger.info("starting Scout checker")
        self.app.scout_checker = PeriodicTrigger(lambda: self.check_scout("checkin"), period=86400)     # Yup, one day.

        self.logger.info("starting event watcher")

        while True:
            cmd, arg, rqueue = self.events.get()
            # self.logger.info("EVENT: %s" % cmd)

            if cmd == 'ESTATS':
                # self.logger.info("updating estats")
                try:
                    self._respond(rqueue, 200, 'updating')
                    self.app.estats.update()
                except Exception as e:
                    self.logger.error("could not update estats: %s" % e)
                    self.logger.exception(e)
                    self._respond(rqueue, 500, 'Envoy stats update failed')
            elif cmd == 'CONFIG_FS':
                try:
                    self.load_config_fs(rqueue, arg)
                except Exception as e:
                    self.logger.error("could not reconfigure: %s" % e)
                    self.logger.exception(e)
                    self._respond(rqueue, 500, 'configuration from filesystem failed')
            elif cmd == 'CONFIG':
                version, url = arg

                try:
                    if version == 'kw':
                        self.load_config_kubewatch(rqueue, url)
                    elif version == 'watt':
                        self.load_config_watt(rqueue, url)
                    else:
                        raise RuntimeError("config from %s not supported" % version)
                except Exception as e:
                    self.logger.error("could not reconfigure: %s" % e)
                    self.logger.exception(e)
                    self._respond(rqueue, 500, 'configuration failed')
            elif cmd == 'SCOUT':
                try:
                    self._respond(rqueue, 200, 'checking Scout')
                    self.check_scout(*arg)
                except Exception as e:
                    self.logger.error("could not reconfigure: %s" % e)
                    self.logger.exception(e)
                    self._respond(rqueue, 500, 'scout check failed')
            else:
                self.logger.error(f"unknown event type: '{cmd}' '{arg}'")
                self._respond(rqueue, 400, f"unknown event type '{cmd}' '{arg}'")

    def _respond(self, rqueue: queue.Queue, status: int, info='') -> None:
        self.logger.debug("responding to query with %s %s" % (status, info))
        rqueue.put((status, info))

    def load_config_fs(self, rqueue: queue.Queue, path: str) -> None:
        self.logger.info("loading configuration from disk: %s" % path)

        # The "path" here can just be a path, but it can also be a command for testing,
        # if the user has chosen to allow that.

        if self.app.allow_fs_commands and (':' in path):
            pfx, rest = path.split(':', 1)

            if pfx.lower() == 'cmd':
                fields = rest.split(':', 1)

                cmd = fields[0].upper()

                args = fields[1:] if (len(fields) > 1) else None

                if cmd.upper() == 'CHIME':
                    self.logger.info('CMD: Chiming')

                    self.chime()

                    self._respond(rqueue, 200, 'Chimed')
                elif cmd.upper() == 'CHIME_RESET':
                    self.chimed = False
                    self.last_chime = False
                    self.env_good = False

                    self.app.scout.reset_events()
                    self.app.scout.report(mode="boot", action="boot1", no_cache=True)

                    self.logger.info('CMD: Reset chime state')
                    self._respond(rqueue, 200, 'CMD: Reset chime state')
                elif cmd.upper() == 'SCOUT_CACHE_RESET':
                    self.app.scout.reset_cache_time()

                    self.logger.info('CMD: Reset Scout cache time')
                    self._respond(rqueue, 200, 'CMD: Reset Scout cache time')
                elif cmd.upper() == 'ENV_OK':
                    self.env_good = True
                    self.failure_list = []

                    self.logger.info('CMD: Marked environment good')
                    self._respond(rqueue, 200, 'CMD: Marked environment good')
                elif cmd.upper() == 'ENV_BAD':
                    self.env_good = False
                    self.failure_list = [ 'failure forced' ]

                    self.logger.info('CMD: Marked environment bad')
                    self._respond(rqueue, 200, 'CMD: Marked environment bad')
                else:
                    self.logger.info(f'CMD: no such command "{cmd}"')
                    self._respond(rqueue, 400, f'CMD: no such command "{cmd}"')

                return
            else:
                self.logger.info(f'CONFIG_FS: invalid prefix "{pfx}"')
                self._respond(rqueue, 400, f'CONFIG_FS: invalid prefix "{pfx}"')

            return

        snapshot = re.sub(r'[^A-Za-z0-9_-]', '_', path)
        scc = FSSecretHandler(app.logger, path, app.snapshot_path, "0")

        aconf = Config()
        fetcher = ResourceFetcher(app.logger, aconf)
        fetcher.load_from_filesystem(path, k8s=app.k8s, recurse=True)

        if not fetcher.elements:
            self.logger.debug("no configuration resources found at %s" % path)
            # self._respond(rqueue, 204, 'ignoring empty configuration')
            # return

        self._load_ir(rqueue, aconf, fetcher, scc, snapshot)

    def load_config_kubewatch(self, rqueue: queue.Queue, url: str):
        snapshot = url.split('/')[-1]
        ss_path = os.path.join(app.snapshot_path, "snapshot-tmp.yaml")

        self.logger.info("copying configuration: kubewatch, %s to %s" % (url, ss_path))

        # Grab the serialization, and save it to disk too.
        elements: List[str] = []

        serialization = load_url_contents(self.logger, "%s/services" % url, stream2=open(ss_path, "w"))

        if serialization:
            elements.append(serialization)
        else:
            self.logger.debug("no services loaded from snapshot %s" % snapshot)

        if Config.enable_endpoints:
            serialization = load_url_contents(self.logger, "%s/endpoints" % url, stream2=open(ss_path, "a"))

            if serialization:
                elements.append(serialization)
            else:
                self.logger.debug("no endpoints loaded from snapshot %s" % snapshot)

        serialization = "---\n".join(elements)

        if not serialization:
            self.logger.debug("no data loaded from snapshot %s" % snapshot)
            # We never used to return here. I'm not sure if that's really correct?
            # self._respond(rqueue, 204, 'ignoring: no data loaded from snapshot %s' % snapshot)
            # return

        scc = KubewatchSecretHandler(app.logger, url, app.snapshot_path, snapshot)

        aconf = Config()
        fetcher = ResourceFetcher(app.logger, aconf)
        fetcher.parse_yaml(serialization, k8s=True)

        if not fetcher.elements:
            self.logger.debug("no configuration found in snapshot %s" % snapshot)

            # Don't actually bail here. If they send over a valid config that happens
            # to have nothing for us, it's still a legit config.
            # self._respond(rqueue, 204, 'ignoring: no configuration found in snapshot %s' % snapshot)
            # return

        self._load_ir(rqueue, aconf, fetcher, scc, snapshot)

    def load_config_watt(self, rqueue: queue.Queue, url: str):
        snapshot = url.split('/')[-1]
        ss_path = os.path.join(app.snapshot_path, "snapshot-tmp.yaml")

        self.logger.info("copying configuration: watt, %s to %s" % (url, ss_path))

        # Grab the serialization, and save it to disk too.
        serialization = load_url_contents(self.logger, url, stream2=open(ss_path, "w"))

        if not serialization:
            self.logger.debug("no data loaded from snapshot %s" % snapshot)
            # We never used to return here. I'm not sure if that's really correct?
            # self._respond(rqueue, 204, 'ignoring: no data loaded from snapshot %s' % snapshot)
            # return

        # Weirdly, we don't need a special WattSecretHandler: parse_watt knows how to handle
        # the secrets that watt sends.
        scc = SecretHandler(app.logger, url, app.snapshot_path, snapshot)

        aconf = Config()
        fetcher = ResourceFetcher(app.logger, aconf)

        if serialization:
            fetcher.parse_watt(serialization)

        if not fetcher.elements:
            self.logger.debug("no configuration found in snapshot %s" % snapshot)

            # Don't actually bail here. If they send over a valid config that happens
            # to have nothing for us, it's still a legit config.
            # self._respond(rqueue, 204, 'ignoring: no configuration found in snapshot %s' % snapshot)
            # return

        self._load_ir(rqueue, aconf, fetcher, scc, snapshot)

    def _load_ir(self, rqueue: queue.Queue, aconf: Config, fetcher: ResourceFetcher,
                 secret_handler: SecretHandler, snapshot: str) -> None:
        aconf.load_all(fetcher.sorted())

        aconf_path = os.path.join(app.snapshot_path, "aconf-tmp.json")
        open(aconf_path, "w").write(aconf.as_json())

        ir = IR(aconf, secret_handler=secret_handler)

        ir_path = os.path.join(app.snapshot_path, "ir-tmp.json")
        open(ir_path, "w").write(ir.as_json())

        econf = EnvoyConfig.generate(ir, "V2")
        diag = Diagnostics(ir, econf)

        bootstrap_config, ads_config = econf.split_config()

        if not self.validate_envoy_config(config=ads_config, retries=self.app.validation_retries):
            self.logger.info("no updates were performed due to invalid envoy configuration, continuing with current configuration...")
            # Don't use app.check_scout; it will deadlock.
            self.check_scout("attempted bad update")
            self._respond(rqueue, 500, 'ignoring: invalid Envoy configuration in snapshot %s' % snapshot)
            return

        snapcount = int(os.environ.get('AMBASSADOR_SNAPSHOT_COUNT', "4"))
        snaplist: List[Tuple[str, str]] = []

        if snapcount > 0:
            self.logger.debug("rotating snapshots for snapshot %s" % snapshot)

            # If snapcount is 4, this range statement becomes range(-4, -1)
            # which gives [ -4, -3, -2 ], which the list comprehension turns
            # into [ ( "-3", "-4" ), ( "-2", "-3" ), ( "-1", "-2" ) ]...
            # which is the list of suffixes to rename to rotate the snapshots.

            snaplist += [ (str(x+1), str(x)) for x in range(-1 * snapcount, -1) ]

            # After dealing with that, we need to rotate the current file into -1.
            snaplist.append(( '', '-1' ))

        # Whether or not we do any rotation, we need to cycle in the '-tmp' file.
        snaplist.append(( '-tmp', '' ))

        for from_suffix, to_suffix in snaplist:
            for fmt in [ "aconf{}.json", "econf{}.json", "ir{}.json", "snapshot{}.yaml" ]:
                from_path = os.path.join(app.snapshot_path, fmt.format(from_suffix))
                to_path = os.path.join(app.snapshot_path, fmt.format(to_suffix))

                try:
                    self.logger.debug("rotate: %s -> %s" % (from_path, to_path))
                    os.rename(from_path, to_path)
                except IOError as e:
                    self.logger.debug("skip %s -> %s: %s" % (from_path, to_path, e))
                    pass
                except Exception as e:
                    self.logger.debug("could not rename %s -> %s: %s" % (from_path, to_path, e))

        app.latest_snapshot = snapshot
        self.logger.info("saving Envoy configuration for snapshot %s" % snapshot)

        with open(app.bootstrap_path, "w") as output:
            output.write(json.dumps(bootstrap_config, sort_keys=True, indent=4))

        with open(app.ads_path, "w") as output:
            output.write(json.dumps(ads_config, sort_keys=True, indent=4))

        app.aconf = aconf
        app.ir = ir
        app.econf = econf
        app.diag = diag

        if app.kick:
            self.logger.info("running '%s'" % app.kick)
            os.system(app.kick)
        elif app.ambex_pid != 0:
            self.logger.info("notifying PID %d ambex" % app.ambex_pid)
            os.kill(app.ambex_pid, signal.SIGHUP)

        # don't worry about TCPMappings yet
        mappings = app.aconf.get_config('mappings')

        if mappings:
            for mapping_name, mapping in mappings.items():
                app.kubestatus.mark_live("Mapping", mapping_name, mapping.get('namespace', Config.ambassador_namespace))

        app.kubestatus.prune()

        if app.ir.k8s_status_updates:
            update_count = 0

            for name in app.ir.k8s_status_updates.keys():
                update_count += 1
                # Strip off any namespace in the name.
                resource_name = name.split('.', 1)[0]
                kind, namespace, update = app.ir.k8s_status_updates[name]
                text = json.dumps(update)

                # self.logger.info(f"K8s status update: {kind} {resource_name}.{namespace}, {text}...")

                app.kubestatus.post(kind, resource_name, namespace, text)

        self.logger.info("configuration updated from snapshot %s" % snapshot)
        self._respond(rqueue, 200, 'configuration updated from snapshot %s' % snapshot)

        if app.health_checks and not app.stats_updater:
            app.logger.info("starting Envoy status updater")
            app.stats_updater = PeriodicTrigger(app.watcher.update_estats, period=5)

        # Check our environment...
        self.check_environment()

        self.chime()

    def chime(self):
        # In general, our reports here should be action "update", and they should honor the
        # Scout cache, but we need to tweak that depending on whether we've done this before
        # and on whether the environment looks OK.

        already_chimed = bool_fmt(self.chimed)
        was_ok = bool_fmt(self.last_chime)
        now_ok = bool_fmt(self.env_good)

        # Poor man's state machine...
        action_key = f'{already_chimed}-{was_ok}-{now_ok}'
        action, no_cache = AmbassadorEventWatcher.Actions[action_key]

        self.logger.debug(f'CHIME: {action_key}')

        chime_args = {
            'no_cache': no_cache,
            'failures': self.failure_list
        }

        if self.app.report_action_keys:
            chime_args['action_key'] = action_key

        # Don't use app.check_scout; it will deadlock.
        self.check_scout(action, **chime_args)

        # Remember that we have now chimed...
        self.chimed = True

        # ...and remember what we sent for that chime.
        self.last_chime = self.env_good

    def check_environment(self, ir: Optional[IR]=None) -> None:
        env_good = True
        chime_failures = {}
        env_status = SystemStatus()

        error_count = 0
        tls_count = 0
        mapping_count = 0

        if not ir:
            ir = app.ir

        if not ir:
            chime_failures['no config loaded'] = True
            env_good = False
        else:
            if not ir.aconf:
                chime_failures['completely empty config'] = True
                env_good = False
            else:
                for err_key, err_list in ir.aconf.errors.items():
                    if err_key == "-global-":
                        err_key = ""

                    for err in err_list:
                        error_count += 1
                        err_text = err['error']

                        self.app.logger.info(f'error {err_key} {err_text}')

                        if err_text.find('CRD') >= 0:
                            if err_text.find('core') >= 0:
                                chime_failures['core CRDs'] = True
                                env_status.failure("CRDs", "Core CRD type definitions are missing")
                            else:
                                chime_failures['other CRDs'] = True
                                env_status.failure("CRDs", "Resolver CRD type definitions are missing")

                            env_good = False
                        elif err_text.find('TLS') >= 0:
                            chime_failures['TLS errors'] = True
                            env_status.failure('TLS', err_text)

                            env_good = False

            for context in ir.tls_contexts:
                if context:
                    tls_count += 1
                    break

            for group in ir.groups.values():
                for mapping in group.mappings:
                    pfx = mapping.get('prefix', None)
                    name = mapping.get('name', None)

                    if pfx:
                        if not pfx.startswith('/ambassador/v0') or not name.startswith('internal_'):
                            mapping_count += 1

        if error_count:
            env_status.failure('Error check', f'{error_count} total error{"" if (error_count == 1) else "s"} logged')
            env_good = False
        else:
            env_status.OK('Error check', "No errors logged")

        if tls_count:
            env_status.OK('TLS', f'{tls_count} TLSContext{" is" if (tls_count == 1) else "s are"} active')
        else:
            chime_failures['no TLS contexts'] = True
            env_status.failure('TLS', "No TLSContexts are active")

            env_good = False

        if mapping_count:
            env_status.OK('Mappings', f'{mapping_count} Mapping{" is" if (mapping_count == 1) else "s are"} active')
        else:
            chime_failures['no Mappings'] = True
            env_status.failure('Mappings', "No Mappings are active")
            env_good = False

        failure_list: List[str] = []

        if not env_good:
            failure_list = list(sorted(chime_failures.keys()))

        self.env_good = env_good
        self.env_status = env_status
        self.failure_list = failure_list

    def check_scout(self, what: str, no_cache: Optional[bool]=False,
                    ir: Optional[IR]=None, failures: Optional[List[str]]=None,
                    action_key: Optional[str]=None) -> None:
        now = datetime.datetime.now()
        uptime = now - boot_time
        hr_uptime = td_format(uptime)

        if not ir:
            ir = app.ir

        self.app.notices.reset()

        scout_args = {
            "uptime": int(uptime.total_seconds()),
            "hr_uptime": hr_uptime
        }

        if failures:
            scout_args['failures'] = failures

        if action_key:
            scout_args['action_key'] = action_key

        if ir:
            self.app.logger.debug("check_scout: we have an IR")

            if not os.environ.get("AMBASSADOR_DISABLE_FEATURES", None):
                self.app.logger.debug("check_scout: including features")
                feat = ir.features()

                request_data = app.estats.stats.get('requests', None)

                if request_data:
                    self.app.logger.debug("check_scout: including requests")

                    for rkey in request_data.keys():
                        cur = request_data[rkey]
                        prev = app.last_request_info.get(rkey, 0)
                        feat[f'request_{rkey}_count'] = max(cur - prev, 0)

                    lrt = app.last_request_time or boot_time
                    since_lrt = now - lrt
                    elapsed = since_lrt.total_seconds()
                    hr_elapsed = td_format(since_lrt)

                    app.last_request_time = now
                    app.last_request_info = request_data

                    feat['request_elapsed'] = elapsed
                    feat['request_hr_elapsed'] = hr_elapsed

                scout_args["features"] = feat

        scout_result = self.app.scout.report(mode="diagd", action=what, no_cache=no_cache, **scout_args)
        scout_notices = scout_result.pop('notices', [])

        global_loglevel = self.app.logger.getEffectiveLevel()

        self.app.logger.debug(f'Scout section: global loglevel {global_loglevel}')

        for notice in scout_notices:
            notice_level_name = notice.get('level') or 'INFO'
            notice_level = logging.getLevelName(notice_level_name)

            if notice_level >= global_loglevel:
                self.app.logger.debug(f'Scout section: include {notice}')
                self.app.notices.post(notice)
            else:
                self.app.logger.debug(f'Scout section: skip {notice}')

        self.app.logger.info("Scout reports %s" % json.dumps(scout_result))
        self.app.logger.info("Scout notices: %s" % json.dumps(scout_notices))
        self.app.logger.debug("App notices after scout: %s" % json.dumps(app.notices.notices))

    def validate_envoy_config(self, config, retries) -> bool:
        if self.app.no_envoy:
            self.app.logger.debug("Skipping validation")
            return True

        # We want to keep the original config untouched
        validation_config = copy.deepcopy(config)
        # Envoy fails to validate with @type field in envoy config, so removing that
        validation_config.pop('@type')
        config_json = json.dumps(validation_config, sort_keys=True, indent=4)

        econf_validation_path = os.path.join(app.snapshot_path, "econf-tmp.json")

        with open(econf_validation_path, "w") as output:
            output.write(config_json)

        command = ['envoy', '--config-path', econf_validation_path, '--mode', 'validate']
        odict = {
            'exit_code': 0,
            'output': ''
        }

        # Try to validate the Envoy config. Short circuit and fall through
        # immediately on concrete success or failure, and retry (up to the
        # limit) on timeout.
        timeout = 5
        for retry in range(retries):
            try:
                odict['output'] = subprocess.check_output(command, stderr=subprocess.STDOUT, timeout=timeout)
                odict['exit_code'] = 0
                break
            except subprocess.CalledProcessError as e:
                odict['exit_code'] = e.returncode
                odict['output'] = e.output
                break
            except subprocess.TimeoutExpired as e:
                odict['exit_code'] = 1
                odict['output'] = e.output
                self.logger.warn("envoy configuration validation timed out after {} seconds{}\n{}",
                    timeout, ', retrying...' if retry < retries - 1 else '', e.output)
                continue

        if odict['exit_code'] == 0:
            self.logger.info("successfully validated the resulting envoy configuration, continuing...")
            return True

        try:
            decoded_error = odict['output'].decode('utf-8')
            odict['output'] = decoded_error
        except:
            pass

        self.logger.error("{}\ncould not validate the envoy configuration above after {} retries, failed with error \n{}\nAborting update...".format(config_json, retries, odict['output']))
        return False


class StandaloneApplication(gunicorn.app.base.BaseApplication):
    def __init__(self, app, options=None):
        self.options = options or {}
        self.application = app
        super(StandaloneApplication, self).__init__()

        # Boot chime. This is basically the earliest point at which we can consider an Ambassador
        # to be "running".
        scout_result = self.application.scout.report(mode="boot", action="boot1", no_cache=True)
        self.application.logger.info(f'BOOT: Scout result {json.dumps(scout_result)}')

    def load_config(self):
        config = dict([(key, value) for key, value in iteritems(self.options)
                       if key in self.cfg.settings and value is not None])
        for key, value in iteritems(config):
            self.cfg.set(key.lower(), value)

    def load(self):
        # This is a little weird, but whatever.
        self.application.watcher = AmbassadorEventWatcher(self.application)
        self.application.watcher.start()

        if self.application.config_path:
            self.application.watcher.post("CONFIG_FS", self.application.config_path)

        return self.application


def _main(snapshot_path=None, bootstrap_path=None, ads_path=None,
          *, dev_magic=False, config_path=None, ambex_pid=0, kick=None,
          banner_endpoint="http://127.0.0.1:8500/banner", k8s=False,
          no_checks=False, no_envoy=False, reload=False, debug=False, verbose=False,
          workers=None, port=Constants.DIAG_PORT, host='0.0.0.0', notices=None,
          validation_retries=5, allow_fs_commands=False, local_scout=False,
          report_action_keys=False):
    """
    Run the diagnostic daemon.

    :param snapshot_path: Path to directory in which to save configuration snapshots and dynamic secrets
    :param bootstrap_path: Path to which to write bootstrap Envoy configuration
    :param ads_path: Path to which to write ADS Envoy configuration
    :param config_path: Optional configuration path to scan for Ambassador YAML files
    :param k8s: If True, assume config_path contains Kubernetes resources (only relevant with config_path)
    :param ambex_pid: Optional PID to signal with HUP after updating Envoy configuration
    :param kick: Optional command to run after updating Envoy configuration
    :param banner_endpoint: Optional endpoint of extra banner to include
    :param no_checks: If True, don't do Envoy-cluster health checking
    :param no_envoy: If True, don't interact with Envoy at all
    :param reload: If True, run Flask in debug mode for live reloading
    :param debug: If True, do debug logging
    :param dev_magic: If True, override a bunch of things for Datawire dev-loop stuff
    :param verbose: If True, do really verbose debug logging
    :param workers: Number of workers; default is based on the number of CPUs present
    :param host: Interface on which to listen
    :param port: Port on which to listen
    :param notices: Optional file to read for local notices
    :param validation_retries: Number of times to retry Envoy configuration validation after a timeout
    :param allow_fs_commands: If true, allow CONFIG_FS to support debug/testing commands
    :param local_scout: Don't talk to remote Scout at all; keep everything purely local
    :param report_action_keys: Report action keys when chiming
    """

    if dev_magic:
        # Override the world.
        os.environ['SCOUT_HOST'] = '127.0.0.1:9999'
        os.environ['SCOUT_HTTPS'] = 'no'

        no_checks = True
        no_envoy = True

        os.makedirs('/tmp/snapshots', mode=0o755, exist_ok=True)

        snapshot_path = '/tmp/snapshots'
        bootstrap_path = '/tmp/boot.json'
        ads_path = '/tmp/ads.json'

        port = 9998

        allow_fs_commands = True
        local_scout = True
        report_action_keys = True

    if no_envoy:
        no_checks = True

    # Create the application itself.
    app.setup(snapshot_path, bootstrap_path, ads_path, config_path, ambex_pid, kick, banner_endpoint,
              k8s, not no_checks, no_envoy, reload, debug, verbose, notices,
              validation_retries, allow_fs_commands, local_scout, report_action_keys)

    if not workers:
        workers = number_of_workers()

    gunicorn_config = {
        'bind': '%s:%s' % (host, port),
        # 'workers': 1,
        'threads': workers,
    }

    app.logger.info("thread count %d, listening on %s" % (gunicorn_config['threads'], gunicorn_config['bind']))

    StandaloneApplication(app, gunicorn_config).run()


def main():
    clize.run(_main)


if __name__ == "__main__":
    main()
