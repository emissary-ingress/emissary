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

from pkg_resources import Requirement, resource_filename

import clize
from clize import Parameter
from flask import Flask, render_template, send_from_directory, request, jsonify
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
    verbose: bool
    notice_path: str
    logger: logging.Logger
    aconf: Config
    ir: Optional[IR]
    econf: EnvoyConfig
    diag: Diagnostics
    notices: 'Notices'
    scout: Scout
    # scout_args: Dict[str, Any]
    # scout_result: Dict[str, Any]
    watcher: 'AmbassadorEventWatcher'
    stats_updater: Optional[PeriodicTrigger]
    last_request_info: Dict[str, int]
    last_request_time: Optional[datetime.datetime]

    def setup(self, snapshot_path: str, bootstrap_path: str, ads_path: str,
              config_path: Optional[str], ambex_pid: int, kick: Optional[str],
              k8s=False, do_checks=True, no_envoy=False, reload=False, debug=False, verbose=False,
              notices=None):
        self.estats = EnvoyStats()
        self.health_checks = do_checks
        self.no_envoy = no_envoy
        self.debugging = reload
        self.verbose = verbose
        self.notice_path = notices
        self.notices = Notices(self.notice_path)
        self.notices.reset()
        self.k8s = k8s

        # This will raise an exception and crash if you pass it a string. That's intentional.
        self.ambex_pid = int(ambex_pid)
        self.kick = kick

        # This feels like overkill.
        self.logger = logging.getLogger("ambassador.diagd")
        self.logger.setLevel(logging.INFO)

        if debug:
            self.logger.setLevel(logging.DEBUG)
            logging.getLogger('ambassador').setLevel(logging.DEBUG)

        self.config_path = config_path
        self.bootstrap_path = bootstrap_path
        self.ads_path = ads_path
        self.snapshot_path = snapshot_path

        self.ir = None
        self.stats_updater = None

        self.last_request_info = {}
        self.last_request_time = None

        # self.scout = Scout(update_frequency=datetime.timedelta(seconds=10))
        self.scout = Scout()

    def check_scout(self, what: str) -> None:
        self.watcher.post("SCOUT", (what, self.ir))

# Get the Flask app defined early. Setup happens later.
app = DiagApp(__name__,
              template_folder=resource_filename(Requirement.parse("ambassador"), "templates"))


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

        # Default to the exception case
        result_to_log = "server error"
        status_to_log = 500
        result_log_level = logging.ERROR
        result = (result_to_log, status_to_log)

        try:
            result = f(*args, reqid=reqid, **kwds)
            if not isinstance(result, tuple):
                result = (result, 200)

            status_to_log = result[1]

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


def system_info():
    return {
        "version": __version__,
        "hostname": SystemInfo.MyHostName,
        "cluster_id": os.environ.get('AMBASSADOR_CLUSTER_ID',
                                     os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")),
        "boot_time": boot_time,
        "hr_uptime": td_format(datetime.datetime.now() - boot_time)
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


@app.route('/_internal/v0/ping', methods=[ 'GET' ])
def handle_ping():
    return "ACK", 200


@app.route('/_internal/v0/update', methods=[ 'POST' ])
def handle_kubewatch_update():
    url = request.args.get('url', None)

    if not url:
        app.logger.error("error: update requested with no URL")
        return "error: update requested with no URL", 400

    app.logger.info("Update requested: kubewatch, %s" % url)

    status, info = app.watcher.post('CONFIG', ( 'kw', url ))

    return info, status


@app.route('/_internal/v0/watt', methods=[ 'POST' ])
def handle_watt_update():
    url = request.args.get('url', None)

    if not url:
        app.logger.error("error: watt update requested with no URL")
        return "error: watt update requested with no URL", 400

    app.logger.info("Update requested: watt, %s" % url)

    status, info = app.watcher.post('CONFIG', ( 'watt', url ))

    return info, status


# @app.route('/_internal/v0/fs', methods=[ 'POST' ])
# def handle_fs():
#     path = request.args.get('path', None)
#
#     if not path:
#         app.logger.error("error: update requested with no PATH")
#         return "error: update requested with no PATH", 400
#
#     app.logger.info("Update requested from %s" % path)
#
#     status, info = app.watcher.post('CONFIG_FS', path)
#
#     return info, status


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
    if not app.ir:
        return "ambassador waiting for config", 503

    status = envoy_status(app.estats)

    if status['ready']:
        return "ambassador readiness check OK (%s)" % status['since_update'], 200
    else:
        return "ambassador not ready (%s)" % status['since_update'], 503


@app.route('/ambassador/v0/diag/', methods=[ 'GET' ])
@standard_handler
def show_overview(reqid=None):
    app.logger.debug("OV %s - showing overview" % reqid)

    diag = app.diag

    app.check_scout("overview")

    if app.verbose:
        app.logger.debug("OV %s: DIAG" % reqid)
        app.logger.debug("%s" % json.dumps(diag.as_dict(), sort_keys=True, indent=4))

    ov = diag.overview(request, app.estats)

    if app.verbose:
        app.logger.debug("OV %s: OV" % reqid)
        app.logger.debug("%s" % json.dumps(ov, sort_keys=True, indent=4))
        app.logger.debug("OV %s: collecting errors" % reqid)

    ddict = collect_errors_and_notices(request, reqid, "overview", diag)

    tvars = dict(system=system_info(),
                 envoy_status=envoy_status(app.estats), 
                 loginfo=app.estats.loginfo,
                 notices=app.notices.notices,
                 **ov, **ddict)

    if request.args.get('json', None):
        key = request.args.get('filter', None)

        if key:
            return jsonify(tvars.get(key, None))
        else:
            return jsonify(tvars)
    else:
        return render_template("overview.html", **tvars)


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

    app.check_scout("detail: %s" % source)

    method = request.args.get('method', None)
    resource = request.args.get('resource', None)

    result = diag.lookup(request, source, app.estats)

    if app.verbose:
        app.logger.debug("RESULT %s" % json.dumps(result, sort_keys=True, indent=4))

    ddict = collect_errors_and_notices(request, reqid, "detail %s" % source, diag)

    tvars = dict(system=system_info(),
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
        return render_template("diag.html", **tvars)


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


class AmbassadorEventWatcher(threading.Thread):
    def __init__(self, app: DiagApp) -> None:
        super().__init__(name="AmbassadorEventWatcher", daemon=True)
        self.app = app
        self.logger = self.app.logger
        self.events: queue.Queue = queue.Queue()

    def post(self, cmd: str, arg: Union[str, Tuple[str, Optional[IR]]]) -> Tuple[int, str]:
        rqueue: queue.Queue = queue.Queue()

        self.events.put((cmd, arg, rqueue))

        return rqueue.get()

    def update_estats(self) -> None:
        self.post('ESTATS', '')

    def run(self):
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
            elif cmd == 'CONFIG_FS':
                try:
                    self.load_config_fs(rqueue, arg)
                except Exception as e:
                    self.logger.error("could not reconfigure: %s" % e)
                    self.logger.exception(e)
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
            elif cmd == 'SCOUT':
                try:
                    self._respond(rqueue, 200, 'checking Scout')
                    self.check_scout(*arg)
                except Exception as e:
                    self.logger.error("could not reconfigure: %s" % e)
                    self.logger.exception(e)
            else:
                self.logger.error("unknown event type: '%s' '%s'" % (cmd, arg))

    def _respond(self, rqueue: queue.Queue, status: int, info='') -> None:
        self.logger.debug("responding to query with %s %s" % (status, info))
        rqueue.put((status, info))

    def load_config_fs(self, rqueue: queue.Queue, path: str) -> None:
        self.logger.info("loading configuration from disk: %s" % path)

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

        if not self.validate_envoy_config(config=ads_config):
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

        self.logger.info("configuration updated from snapshot %s" % snapshot)
        self._respond(rqueue, 200, 'configuration updated from snapshot %s' % snapshot)

        if app.health_checks and not app.stats_updater:
            app.logger.info("starting Envoy status updater")
            app.stats_updater = PeriodicTrigger(app.watcher.update_estats, period=5)
            # app.scout_updater = PeriodicTrigger(lambda: app.watcher.check_scout("30s"), period=30)

        # Don't use app.check_scout; it will deadlock. And don't bother doing the Scout
        # update until after we've taken care of Envoy.
        self.check_scout("update")

    def check_scout(self, what: str, ir: Optional[IR] = None) -> None:
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

        scout_result = self.app.scout.report(mode="diagd", action=what, **scout_args)
        scout_notices = scout_result.pop('notices', [])
        self.app.notices.extend(scout_notices)

        self.app.logger.info("Scout reports %s" % json.dumps(scout_result))
        self.app.logger.info("Scout notices: %s" % json.dumps(scout_notices))
        self.app.logger.debug("App notices after scout: %s" % json.dumps(app.notices.notices))

    def validate_envoy_config(self, config) -> bool:
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

        try:
            odict['output'] = subprocess.check_output(command, stderr=subprocess.STDOUT, timeout=5)
            odict['exit_code'] = 0
        except subprocess.CalledProcessError as e:
            odict['exit_code'] = e.returncode
            odict['output'] = e.output

        if odict['exit_code'] == 0:
            self.logger.info("successfully validated the resulting envoy configuration, continuing...")
            return True

        self.logger.info("{}\ncould not validate the envoy configuration above, failed with error \n{}\nAborting update...".format(config_json, odict['output']))
        return False


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
        # This is a little weird, but whatever.
        self.application.watcher = AmbassadorEventWatcher(self.application)
        self.application.watcher.start()

        if self.application.config_path:
            self.application.watcher.post("CONFIG_FS", self.application.config_path)

        return self.application


def _main(snapshot_path: Parameter.REQUIRED, bootstrap_path: Parameter.REQUIRED, ads_path: Parameter.REQUIRED,
          *, config_path=None, ambex_pid=0, kick=None, k8s=False,
          no_checks=False, no_envoy=False, reload=False, debug=False, verbose=False,
          workers=None, port=Constants.DIAG_PORT, host='0.0.0.0', notices=None):
    """
    Run the diagnostic daemon.

    :param snapshot_path: Path to directory in which to save configuration snapshots and dynamic secrets
    :param bootstrap_path: Path to which to write bootstrap Envoy configuration
    :param ads_path: Path to which to write ADS Envoy configuration
    :param config_path: Optional configuration path to scan for Ambassador YAML files
    :param k8s: If True, assume config_path contains Kubernetes resources (only relevant with config_path)
    :param ambex_pid: Optional PID to signal with HUP after updating Envoy configuration
    :param kick: Optional command to run after updating Envoy configuration
    :param no_checks: If True, don't do Envoy-cluster health checking
    :param no_envoy: If True, don't interact with Envoy at all
    :param reload: If True, run Flask in debug mode for live reloading
    :param debug: If True, do debug logging
    :param verbose: If True, do really verbose debug logging
    :param workers: Number of workers; default is based on the number of CPUs present
    :param host: Interface on which to listen
    :param port: Port on which to listen
    :param notices: Optional file to read for local notices
    """

    if no_envoy:
        no_checks = True

    # Create the application itself.
    app.setup(snapshot_path, bootstrap_path, ads_path, config_path, ambex_pid, kick,
              k8s, not no_checks, no_envoy, reload, debug, verbose, notices)

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
