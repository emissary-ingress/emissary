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
from typing import Any, Callable, Dict, List, Optional, Tuple, Type, Union, TYPE_CHECKING
from typing import cast as typecast

import datetime
import difflib
import functools
import json
import orjson
import logging
import multiprocessing
import os
import queue
import re
import signal
import sys
import threading
import time
import traceback
import uuid
import requests
import jsonpatch

from expiringdict import ExpiringDict
from prometheus_client import CollectorRegistry, ProcessCollector, generate_latest, Info, Gauge
from pythonjsonlogger import jsonlogger

import concurrent.futures

from pkg_resources import Requirement, resource_filename

import click
from flask import Flask, render_template, send_from_directory, request, jsonify, Response
from flask import json as flask_json
import gunicorn.app.base

from ambassador import Cache, Config, IR, EnvoyConfig, Diagnostics, Scout, Version
from ambassador.reconfig_stats import ReconfigStats
from ambassador.ir.irambassador import IRAmbassador
from ambassador.ir.irbasemapping import IRBaseMapping
from ambassador.utils import SystemInfo, Timer, PeriodicTrigger, SavedSecret, load_url_contents, parse_json, dump_json, parse_bool
from ambassador.utils import SecretHandler, KubewatchSecretHandler, FSSecretHandler, parse_bool
from ambassador.fetch import ResourceFetcher

from ambassador.diagnostics import EnvoyStatsMgr, EnvoyStats

from ambassador.constants import Constants

if TYPE_CHECKING:
    from ambassador.ir.irtlscontext import IRTLSContext # pragma: no cover

__version__ = Version

boot_time = datetime.datetime.now()

# allows 10 concurrent users, with a request timeout of 60 seconds
tvars_cache = ExpiringDict(max_len=10, max_age_seconds=60)

logHandler = None
if parse_bool(os.environ.get("AMBASSADOR_JSON_LOGGING", "false")):
    jsonFormatter = jsonlogger.JsonFormatter("%%(asctime)s %%(filename)s %%(lineno)d %%(process)d (threadName)s %%(levelname)s %%(message)s")
    logHandler = logging.StreamHandler()
    logHandler.setFormatter(jsonFormatter)

    # Set the root logger to INFO level and tell it to use the new log handler.
    logger = logging.getLogger()
    logger.setLevel(logging.INFO)
    logger.addHandler(logHandler)

    # Update all of the other loggers to also use the new log handler.
    loggingManager = getattr(logging.root, 'manager', None)
    if loggingManager is not None:
        for name in loggingManager.loggerDict:
            logging.getLogger(name).addHandler(logHandler)
    else:
        print("Could not find a logging manager. Some logging may not be properly JSON formatted!")
else:
    # Default log level
    level = logging.INFO

    # Check for env var log level
    if level_name := os.getenv("AES_LOG_LEVEL"):
        level_number = logging.getLevelName(level_name.upper())

        if isinstance(level_number, int):
            level = level_number

    # Set defauts for all loggers
    logging.basicConfig(
        level=level,
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
    cache: Optional[Cache]
    ambex_pid: int
    kick: Optional[str]
    estatsmgr: EnvoyStatsMgr
    config_path: Optional[str]
    snapshot_path: str
    bootstrap_path: str
    ads_path: str
    clustermap_path: str
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
    # self.diag is actually a property
    _diag: Optional[Diagnostics]
    notices: 'Notices'
    scout: Scout
    watcher: 'AmbassadorEventWatcher'
    stats_updater: Optional[PeriodicTrigger]
    scout_checker: Optional[PeriodicTrigger]
    last_request_info: Dict[str, int]
    last_request_time: Optional[datetime.datetime]
    latest_snapshot: str
    banner_endpoint: Optional[str]
    metrics_endpoint: Optional[str]

    # Reconfiguration stats
    reconf_stats: ReconfigStats

    # Custom metrics registry to weed-out default metrics collectors because the
    # default collectors can't be prefixed/namespaced with ambassador_.
    # Using the default metrics collectors would lead to name clashes between the Python and Go instrumentations.
    metrics_registry: CollectorRegistry

    config_lock: threading.Lock
    diag_lock: threading.Lock

    def setup(self, snapshot_path: str, bootstrap_path: str, ads_path: str,
              config_path: Optional[str], ambex_pid: int, kick: Optional[str], banner_endpoint: Optional[str],
              metrics_endpoint: Optional[str], k8s=False, do_checks=True, no_envoy=False, reload=False, debug=False,
              verbose=False, notices=None, validation_retries=5, allow_fs_commands=False, local_scout=False,
              report_action_keys=False, enable_fast_reconfigure=False, clustermap_path=None):
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
        self.metrics_endpoint = metrics_endpoint
        self.metrics_registry = CollectorRegistry(auto_describe=True)
        self.enable_fast_reconfigure = enable_fast_reconfigure

        # Init logger, inherits settings from default
        self.logger = logging.getLogger("ambassador.diagd")

        # Initialize the Envoy stats manager...
        self.estatsmgr = EnvoyStatsMgr(self.logger)

        # ...and the incremental-reconfigure stats.
        self.reconf_stats = ReconfigStats(self.logger)

        # This will raise an exception and crash if you pass it a string. That's intentional.
        self.ambex_pid = int(ambex_pid)
        self.kick = kick

        # Initialize the cache if we're allowed to.
        if self.enable_fast_reconfigure:
            self.logger.info("AMBASSADOR_FAST_RECONFIGURE enabled, initializing cache")
            self.cache = Cache(self.logger)
        else:
            self.logger.info("AMBASSADOR_FAST_RECONFIGURE disabled, not initializing cache")
            self.cache = None

        # Use Timers to keep some stats on reconfigurations
        self.config_timer = Timer("reconfiguration", self.metrics_registry)
        self.fetcher_timer = Timer("Fetcher", self.metrics_registry)
        self.aconf_timer = Timer("AConf", self.metrics_registry)
        self.ir_timer = Timer("IR", self.metrics_registry)
        self.econf_timer = Timer("EConf", self.metrics_registry)
        self.diag_timer = Timer("Diagnostics", self.metrics_registry)

        # Use gauges to keep some metrics on active config
        self.diag_errors = Gauge(f'diagnostics_errors', f'Number of configuration errors',
                                 namespace='ambassador', registry=self.metrics_registry)
        self.diag_notices = Gauge(f'diagnostics_notices', f'Number of configuration notices',
                                 namespace='ambassador', registry=self.metrics_registry)
        self.diag_log_level = Gauge(f'log_level', f'Debug log level enabled or not',
                                 ["level"],
                                 namespace='ambassador', registry=self.metrics_registry)

        if debug:
            self.logger.setLevel(logging.DEBUG)
            self.diag_log_level.labels('debug').set(1)
            logging.getLogger('ambassador').setLevel(logging.DEBUG)
        else:
            self.diag_log_level.labels('debug').set(0)


        # Assume that we will NOT update Mapping status.
        ksclass: Type[KubeStatus] = KubeStatusNoMappings

        if os.environ.get("AMBASSADOR_UPDATE_MAPPING_STATUS", "false").lower() == "true":
            self.logger.info("WILL update Mapping status")
            ksclass = KubeStatus
        else:
            self.logger.info("WILL NOT update Mapping status")

        self.kubestatus = ksclass(self)

        self.config_path = config_path
        self.bootstrap_path = bootstrap_path
        self.ads_path = ads_path
        self.snapshot_path = snapshot_path
        self.clustermap_path = clustermap_path or os.path.join(os.path.dirname(self.bootstrap_path), "clustermap.json")

        # You must hold config_lock when updating config elements (including diag!).
        self.config_lock = threading.Lock()

        # When generating new diagnostics, there's a dance around config_lock and
        # diag_lock -- see the diag() property.
        self.diag_lock = threading.Lock()

        # Why are we doing this? Aren't we sure we're singlethreaded here?
        # Well, yes. But self.diag is actually a property, and it will raise an
        # assertion failure if we're not holding self.config_lock... and once
        # the lock is in play at all, we're gonna time it, in case my belief
        # that grabbing the lock here is always effectively free turns out to
        # be wrong.

        with self.config_lock:
            self.ir = None      # don't update unless you hold config_lock
            self.econf = None   # don't update unless you hold config_lock
            self.diag = None    # don't update unless you hold config_lock

        self.stats_updater = None
        self.scout_checker = None

        self.last_request_info = {}
        self.last_request_time = None

        # self.scout = Scout(update_frequency=datetime.timedelta(seconds=10))
        self.scout = Scout(local_only=self.local_scout)

        ProcessCollector(namespace="ambassador", registry=self.metrics_registry)
        metrics_info = Info(name='diagnostics', namespace='ambassador', documentation='Ambassador diagnostic info', registry=self.metrics_registry)
        metrics_info.info({
            "version": __version__,
            "ambassador_id": Config.ambassador_id,
            "cluster_id": os.environ.get('AMBASSADOR_CLUSTER_ID',
                                         os.environ.get('AMBASSADOR_SCOUT_ID', "00000000-0000-0000-0000-000000000000")),
            "single_namespace": str(Config.single_namespace),
        })

    @property
    def diag(self) -> Optional[Diagnostics]:
        """
        It turns out to be expensive to generate the Diagnostics class, so
        app.diag is a property that does it on demand, handling Timers and
        the config lock for you.

        You MUST NOT already hold the config_lock or the diag_lock when
        trying to read app.diag.

        You MUST already have loaded an IR.
        """

        # The config_lock is meant to make sure that we don't ever update
        # self.diag in two places at once, so grab that first.
        with self.config_lock:
            # If we've already generated diagnostics...
            if app._diag:
                # ...then we're good to go.
                return app._diag

        # If here, we have _not_ generated diagnostics, and we've dropped the
        # config lock so as not to block anyone else. Next up: grab the diag
        # lock, because we'd rather not have two diag generations happening at
        # once.
        with self.diag_lock:
            # Did someone else sneak in between releasing the config lock and
            # grabbing the diag lock?
            if app._diag:
                # Yup. Use their work.
                return app._diag

            # OK, go generate diagnostics.
            _diag = self._generate_diagnostics()

            # If that didn't work, no point in messing with the config lock.
            if not _diag:
                return None

            # Once here, we need to - once again - grab the config lock to update
            # app._diag. This is safe because this is the only place we ever mess
            # with the diag lock, so nowhere else will try to grab the diag lock
            # while holding the config lock.
            with app.config_lock:
                app._diag = _diag

            # Finally, we can return app._diag to our caller.
            return app._diag

    @diag.setter
    def diag(self, diag: Optional[Diagnostics]) -> None:
        """
        It turns out to be expensive to generate the Diagnostics class, so
        app.diag is a property that does it on demand, handling Timers and
        the config lock for you.

        You MUST already hold the config_lock when trying to update app.diag.
        You MUST NOT hold the diag_lock.
        """
        self._diag = diag

    def _generate_diagnostics(self) -> Optional[Diagnostics]:
        """
        Do the heavy lifting of generating Diagnostics for our current configuration.
        Really, only the diag() property should be calling this method, but if you
        are convinced that you need to call it from elsewhere:

        1. You're almost certainly wrong about needing to call it from elsewhere.
        2. You MUST hold the diag_lock when calling this method.
        3. You MUST NOT hold the config_lock when calling this method.
        4. No, really, you're wrong. Don't call this method from anywhere but the
           diag() property.
        """

        # Make sure we have an IR and econf to work with.
        if not app.ir or not app.econf:
            # Nope, bail.
            return None

        # OK, go ahead and generate diagnostics. Use the diag_timer to time
        # this.
        with self.diag_timer:
            _diag = Diagnostics(app.ir, app.econf)

            # Update some metrics data points given the new generated Diagnostics
            diag_dict = _diag.as_dict()
            self.diag_errors.set(len(diag_dict.get("errors", [])))
            self.diag_notices.set(len(diag_dict.get("notices", [])))

            # Note that we've updated diagnostics, since that might trigger a
            # timer log.
            self.reconf_stats.mark("diag")

            return _diag

    def check_scout(self, what: str) -> None:
        self.watcher.post("SCOUT", (what, self.ir))

    def post_timer_event(self) -> None:
        # Post an event to do a timer check.
        self.watcher.post("TIMER", None)

    def check_timers(self) -> None:
        # Actually do the timer check.

        if self.reconf_stats.needs_timers():
            # OK! Log the timers...

            for t in [ self.config_timer,
                    self.fetcher_timer,
                    self.aconf_timer,
                    self.ir_timer,
                    self.econf_timer,
                    self.diag_timer ]:
                if t:
                    self.logger.info(t.summary())

            # ...and the cache statistics, if we can.
            if self.cache:
                self.cache.dump_stats()

            # Always dump the reconfiguration stats...
            self.reconf_stats.dump()

            # ...and mark that the timers have been logged.
            self.reconf_stats.mark_timers_logged()

        # In this case we need to check to see if it's time to do a configuration
        # check, too.
        if self.reconf_stats.needs_check():
            result = False

            try:
                result = self.check_cache()
            except Exception as e:
                tb = "\n".join(traceback.format_exception(*sys.exc_info()))
                self.logger.error("CACHE: CHECK FAILED: %s\n%s" % (e, tb))

            # Mark that the check has happened.
            self.reconf_stats.mark_checked(result)

    @staticmethod
    def json_diff(what: str, j1: str, j2: str) -> str:
        output = ""

        l1 = j1.split("\n")
        l2 = j2.split("\n")

        n1 = f"{what} incremental"
        n2 = f"{what} nonincremental"

        output += "\n--------\n"

        for line in difflib.context_diff(l1, l2, fromfile=n1, tofile=n2):
            line = line.rstrip()
            output += line
            output += "\n"

        return output


    def check_cache(self) -> bool:
        # We're going to build a shiny new IR and econf from our existing aconf, and make
        # sure everything matches. We will _not_ use the existing cache for this.
        #
        # For this, make sure we have an IR already...
        assert(self.aconf)
        assert(self.ir)
        assert(self.econf)

        # Compute this IR/econf with a new empty cache. It saves a lot of trouble with
        # having to delete cache keys from the JSON.

        self.logger.debug("CACHE: starting check")
        cache = Cache(self.logger)
        scc = SecretHandler(app.logger, "check_cache", app.snapshot_path, "check")
        ir = IR(self.aconf, secret_handler=scc, cache=cache)
        econf = EnvoyConfig.generate(ir, Config.envoy_api_version, cache=cache)

        # This is testing code.
        # name = list(ir.clusters.keys())[0]
        # del(ir.clusters[name])

        i1 = self.ir.as_json()
        i2 = ir.as_json()

        e1 = self.econf.as_json()
        e2 = econf.as_json()

        result = True
        errors = ""

        if i1 != i2:
            result = False
            self.logger.error("CACHE: IR MISMATCH")
            errors += "IR diffs:\n"
            errors += self.json_diff("IR", i1, i2)

        if e1 != e2:
            result = False
            self.logger.error("CACHE: ENVOY CONFIG MISMATCH")
            errors += "econf diffs:\n"
            errors += self.json_diff("econf", i1, i2)

        if not result:
            err_path = os.path.join(self.snapshot_path, "diff-tmp.txt")

            open(err_path, "w").write(errors)

            snapcount = int(os.environ.get('AMBASSADOR_SNAPSHOT_COUNT', "4"))
            snaplist: List[Tuple[str, str]] = []

            if snapcount > 0:
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
                from_path = os.path.join(app.snapshot_path, "diff{}.txt".format(from_suffix))
                to_path = os.path.join(app.snapshot_path, "diff{}.txt".format(to_suffix))

                try:
                    self.logger.debug("rotate: %s -> %s" % (from_path, to_path))
                    os.rename(from_path, to_path)
                except IOError as e:
                    self.logger.debug("skip %s -> %s: %s" % (from_path, to_path, e))
                    pass
                except Exception as e:
                    self.logger.debug("could not rename %s -> %s: %s" % (from_path, to_path, e))

        self.logger.info("CACHE: check %s" % ("succeeded" if result else "failed"))

        return result


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

        app.logger.debug("%s START" % prefix)

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


def internal_handler(f):
    """
    Reject requests where the remote address is not localhost. See the docstring
    for _is_local_request() for important caveats!
    """
    func_name = getattr(f, '__name__', '<anonymous>')

    @functools.wraps(f)
    def wrapper(*args, **kwds):
        if not _is_local_request():
            return "Forbidden\n", 403
        return f(*args, **kwds)

    return wrapper


######## UTILITIES


def _is_local_request() -> bool:
    """
    Determine if this request originated with localhost.

    We rely on healthcheck_server.go setting the X-Ambassador-Diag-IP header for us
    (and we rely on it overwriting anything that's already there!).

    It might be possible to consider the environment variables SERVER_NAME and
    SERVER_PORT instead, as those are allegedly required by WSGI... but attempting
    to do so in Flask/GUnicorn yielded a worse implementation that was still not
    portable.

    """

    remote_addr: Optional[str] = ""

    remote_addr = request.headers.get("X-Ambassador-Diag-IP")

    return remote_addr == "127.0.0.1"


def _allow_diag_ui() -> bool:
    """
    Helper function to check if diag ui traffic is allowed or not
    based on the different flags from the config:
    * diagnostics.enabled: Enable to diag UI by adding mappings
    * diagnostics.allow_non_local: Allow non local traffic
                                   even when diagnotics UI is disabled.
                                   Mappings are not added for the diag UI
                                   but the diagnotics UI is still exposed for
                                   the pod IP in the admin port.
    * local traffic or not: When diagnotics disagled and allow_non_local is false,
                            allow traffic only from localhost clients
    """
    enabled = False
    allow_non_local= False
    ir = app.ir
    if ir:
        enabled = ir.ambassador_module.diagnostics.get("enabled", False)
        allow_non_local = ir.ambassador_module.diagnostics.get("allow_non_local", False)
    if not enabled and not _is_local_request() and not allow_non_local:
        return False
    return True


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
            local_notices = parse_json(local_data)
        except OSError:
            pass
        except:
            local_notices.append({ 'level': 'ERROR', 'message': 'bad local notices: %s' % local_data })

        self.notices = local_notices
        # app.logger.info("Notices: after RESET: %s" % dump_json(self.notices))

    def post(self, notice):
        # app.logger.debug("Notices: POST %s" % notice)
        self.notices.append(notice)
        # app.logger.info("Notices: after POST: %s" % dump_json(self.notices))

    def prepend(self, notice):
        # app.logger.debug("Notices: PREPEND %s" % notice)
        self.notices.insert(0, notice)
        # app.logger.info("Notices: after PREPEND: %s" % dump_json(self.notices))

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

        app.logger.debug(f'DEBUG_MODE {debug_mode}')

    status_dict = {'config failure': [False, 'no configuration loaded']}

    env_status = getattr(app.watcher, 'env_status', None)

    if env_status:
        status_dict = env_status.to_dict()
        app.logger.debug(f"status_dict {status_dict}")

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


def envoy_status(estats: EnvoyStats):
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
@internal_handler
def handle_ping():
    return "ACK\n", 200


@app.route("/_internal/v0/features", methods=[ 'GET' ])
@internal_handler
def handle_features():
    # If we don't have an IR yet, do nothing.
    #
    # We don't bother grabbing the config_lock here because we're not changing
    # anything, and an features request hitting at exactly the same moment as
    # the first configure is a race anyway. If it fails, that's not a big deal,
    # they can try again.
    if not app.ir:
          app.logger.debug("Features: configuration required first")
          return "Can't do features before configuration", 503

    return jsonify(app.ir.features()), 200


@app.route('/_internal/v0/watt', methods=[ 'POST' ])
@internal_handler
def handle_watt_update():
    url = request.args.get('url', None)

    if not url:
        app.logger.error("error: watt update requested with no URL")
        return "error: watt update requested with no URL\n", 400

    app.logger.debug("Update requested: watt, %s" % url)

    status, info = app.watcher.post('CONFIG', ( 'watt', url ))

    return info, status


@app.route('/_internal/v0/fs', methods=[ 'POST' ])
@internal_handler
def handle_fs():
    path = request.args.get('path', None)

    if not path:
        app.logger.error("error: update requested with no PATH")
        return "error: update requested with no PATH\n", 400

    app.logger.debug("Update requested from %s" % path)

    status, info = app.watcher.post('CONFIG_FS', path)

    return info, status


@app.route('/_internal/v0/events', methods=[ 'GET' ])
@internal_handler
def handle_events():
    if not app.local_scout:
        return 'Local Scout is not enabled\n', 400

    event_dump = [
        ( x['local_scout_timestamp'], x['mode'], x['action'], x ) for x in app.scout._scout.events
    ]

    app.logger.debug(f'Event dump {event_dump}')

    return jsonify(event_dump)


@app.route('/ambassador/v0/favicon.ico', methods=[ 'GET' ])
def favicon():
    template_path = resource_filename(Requirement.parse("ambassador"), "templates")

    return send_from_directory(template_path, "favicon.ico")


@app.route('/ambassador/v0/check_alive', methods=[ 'GET' ])
def check_alive():
    status = envoy_status(app.estatsmgr.get_stats())

    if status['alive']:
        return "ambassador liveness check OK (%s)\n" % status['uptime'], 200
    else:
        return "ambassador seems to have died (%s)\n" % status['uptime'], 503


@app.route('/ambassador/v0/check_ready', methods=[ 'GET' ])
def check_ready():
    if not app.ir:
        return "ambassador waiting for config\n", 503

    status = envoy_status(app.estatsmgr.get_stats())

    if status['ready']:
        return "ambassador readiness check OK (%s)\n" % status['since_update'], 200
    else:
        return "ambassador not ready (%s)\n" % status['since_update'], 503


@app.route('/ambassador/v0/diag/', methods=[ 'GET' ])
@standard_handler
def show_overview(reqid=None):
    # If we don't have an IR yet, do nothing.
    #
    # We don't bother grabbing the config_lock here because we're not changing
    # anything, and an overview request hitting at exactly the same moment as
    # the first configure is a race anyway. If it fails, that's not a big deal,
    # they can try again.
    if not app.ir:
          app.logger.debug("OV %s - can't do overview before configuration" % reqid)
          return "Can't do overview before configuration", 503

    if not _allow_diag_ui():
        return Response("Not found\n", 404)

    app.logger.debug("OV %s - showing overview" % reqid)

    # Remember that app.diag is a property that can involve some real expense
    # to compute -- we don't want to call it more than once here, so we cache
    # its value.
    diag = app.diag

    if app.verbose:
        app.logger.debug("OV %s: DIAG" % reqid)
        app.logger.debug("%s" % dump_json(diag.as_dict(), pretty=True))

    estats = app.estatsmgr.get_stats()
    ov = diag.overview(request, estats)

    if app.verbose:
        app.logger.debug("OV %s: OV" % reqid)
        app.logger.debug("%s" % dump_json(ov, pretty=True))
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
                 envoy_status=envoy_status(estats),
                 loginfo=app.estatsmgr.loginfo,
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
            # parameter of the json.loads(...) call. We have to use python's
            # json library instead of orjson, because orjson does not support
            # the object_hook feature.

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
        app.diag_log_level.labels('debug').set(1 if loglevel == 'debug' else 0)

        if not app.estatsmgr.update_log_levels(time.time(), level=loglevel):
            notice = { 'level': 'WARNING', 'message': "Could not update log level!" }
        # else:
        #     return redirect("/ambassador/v0/diag/", code=302)

    # We need to grab errors and notices from diag.as_dict(), process the errors so
    # they work for the HTML rendering, and post the notices to app.notices. Then we
    # return the dict representation that our caller should work with.

    ddict = diag.as_dict()

    # app.logger.debug("ddict %s" % dump_json(ddict, pretty=True))

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
    # If we don't have an IR yet, do nothing.
    #
    # We don't bother grabbing the config_lock here because we're not changing
    # anything, and an overview request hitting at exactly the same moment as
    # the first configure is a race anyway. If it fails, that's not a big deal,
    # they can try again.
    if not app.ir:
          app.logger.debug("SRC %s - can't do intermediate for %s before configuration" % (reqid, source))
          return "Can't do overview before configuration", 503

    if not _allow_diag_ui():
        return Response("Not found\n", 404)

    app.logger.debug("SRC %s - getting intermediate for '%s'" % (reqid, source))

    # Remember that app.diag is a property that can involve some real expense
    # to compute -- we don't want to call it more than once here, so we cache
    # its value.
    diag = app.diag

    method = request.args.get('method', None)
    resource = request.args.get('resource', None)

    estats = app.estatsmgr.get_stats()
    result = diag.lookup(request, source, estats)

    if app.verbose:
        app.logger.debug("RESULT %s" % dump_json(result, pretty=True))

    ddict = collect_errors_and_notices(request, reqid, "detail %s" % source, diag)

    tvars = dict(system=system_info(app),
                 envoy_status=envoy_status(estats),
                 loginfo=app.estatsmgr.loginfo,
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

    return dump_json(obj, pretty=True)


@app.template_filter('sort_clusters_by_service')
def sort_clusters_by_service(clusters):
    return sorted(clusters, key=lambda x: x['service'])
    # return sorted([ c for c in clusters.values() ], key=lambda x: x['service'])


@app.template_filter('source_lookup')
def source_lookup(name, sources):
    app.logger.debug("%s => sources %s" % (name, sources))

    source = sources.get(name, {})

    app.logger.debug("%s => source %s" % (name, source))

    return source.get('_source', name)


@app.route('/metrics', methods=['GET'])
@standard_handler
def get_prometheus_metrics(*args, **kwargs):
    # Envoy metrics
    envoy_metrics = app.estatsmgr.get_prometheus_stats()

    # Ambassador OSS metrics
    ambassador_metrics = generate_latest(registry=app.metrics_registry).decode('utf-8')

    # Extra metrics endpoint
    extra_metrics_content = ''
    if app.metrics_endpoint and app.ir and app.ir.edge_stack_allowed:
        try:
            response = requests.get(app.metrics_endpoint)
            if response.status_code == 200:
                extra_metrics_content = response.text
        except Exception as e:
            app.logger.error("could not get metrics_endpoint: %s" % e)

    return Response(''.join([envoy_metrics, ambassador_metrics, extra_metrics_content]).encode('utf-8'),
                    200, mimetype="text/plain")


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

    def __init__(self, app) -> None:
        self.app = app
        self.logger = app.logger
        self.live: Dict[str,  bool] = {}
        self.current_status: Dict[str, str] = {}
        self.pool = concurrent.futures.ProcessPoolExecutor(max_workers=5)

    def mark_live(self, kind: str, name: str, namespace: str) -> None:
        key = f"{kind}/{name}.{namespace}"

        # self.logger.debug(f"KubeStatus MASTER {os.getpid()}: mark_live {key}")
        self.live[key] = True

    def prune(self) -> None:
        drop: List[str] = []

        for key in self.current_status.keys():
            if not self.live.get(key, False):
                drop.append(key)

        for key in drop:
            # self.logger.debug(f"KubeStatus MASTER {os.getpid()}: prune {key}")
            del(self.current_status[key])

        self.live = {}

    def post(self, kind: str, name: str, namespace: str, text: str) -> None:
        key = f"{kind}/{name}.{namespace}"
        extant = self.current_status.get(key, None)

        if extant == text:
            # self.logger.info(f"KubeStatus MASTER {os.getpid()}: {key} == {text}")
            pass
        else:
            # self.logger.info(f"KubeStatus MASTER {os.getpid()}: {key} needs {text}")

            # For now we're going to assume that this works.
            self.current_status[key] = text
            f = self.pool.submit(kubestatus_update, kind, name, namespace, text)
            f.add_done_callback(kubestatus_update_done)


# The KubeStatusNoMappings class clobbers the mark_live() method of the
# KubeStatus class, so that updates to Mappings don't actually have any
# effect, but updates to Ingress (for example) do.
class KubeStatusNoMappings (KubeStatus):
    def mark_live(self, kind: str, name: str, namespace: str) -> None:
        pass

    def post(self, kind: str, name: str, namespace: str, text: str) -> None:
        # There's a path (via IRBaseMapping.check_status) where a Mapping
        # can be added directly to ir.k8s_status_updates, which will come
        # straight here without mark_live being involved -- so short-circuit
        # here for Mappings, too.

        if kind == 'Mapping':
            return

        super().post(kind, name, namespace, text)

def kubestatus_update(kind: str, name: str, namespace: str, text: str) -> str:
    cmd = [ 'kubestatus', '--cache-dir', '/tmp/client-go-http-cache', kind, name, '-n', namespace, '-u', '/dev/fd/0' ]
    # print(f"KubeStatus UPDATE {os.getpid()}: running command: {cmd}")

    try:
        rc = subprocess.run(cmd, input=text.encode('utf-8'), stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=5)
        if rc.returncode == 0:
            return f"{name}.{namespace}: update OK"
        else:
            return f"{name}.{namespace}: error {rc.returncode}"

    except subprocess.TimeoutExpired as e:
        return f"{name}.{namespace}: timed out\n\n{e.output}"

def kubestatus_update_done(f: concurrent.futures.Future) -> None:
    # print(f"KubeStatus DONE {os.getpid()}: result {f.result()}")
    pass

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

    reCompressed = re.compile(r'-\d+$')

    def __init__(self, app: DiagApp) -> None:
        super().__init__(name="AEW", daemon=True)
        self.app = app
        self.logger = self.app.logger
        self.events: queue.Queue = queue.Queue()

        self.chimed = False         # Have we ever sent a chime about the environment?
        self.last_chime = False     # What was the status of our last chime? (starts as False)
        self.env_good = False       # Is our environment currently believed to be OK?
        self.failure_list: List[str] = [ 'unhealthy at boot' ]     # What's making our environment not OK?

    def post(self, cmd: str, arg: Optional[Union[str, Tuple[str, Optional[IR]]]]) -> Tuple[int, str]:
        rqueue: queue.Queue = queue.Queue()

        self.events.put((cmd, arg, rqueue))

        return rqueue.get()

    def update_estats(self) -> None:
        self.app.estatsmgr.update()

    def run(self):
        self.logger.info("starting Scout checker and timer logger")
        self.app.scout_checker = PeriodicTrigger(lambda: self.check_scout("checkin"), period=86400)     # Yup, one day.
        self.app.timer_logger = PeriodicTrigger(self.app.post_timer_event, period=1)

        self.logger.info("starting event watcher")

        while True:
            cmd, arg, rqueue = self.events.get()
            # self.logger.info("EVENT: %s" % cmd)

            if cmd == 'CONFIG_FS':
                try:
                    self.load_config_fs(rqueue, arg)
                except Exception as e:
                    self.logger.error("could not reconfigure: %s" % e)
                    self.logger.exception(e)
                    self._respond(rqueue, 500, 'configuration from filesystem failed')
            elif cmd == 'CONFIG':
                version, url = arg

                try:
                    if version == 'watt':
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
            elif cmd == 'TIMER':
                try:
                    self._respond(rqueue, 200, 'done')
                    self.app.check_timers()
                except Exception as e:
                    self.logger.error("could not check timers? %s" % e)
                    self.logger.exception(e)
            else:
                self.logger.error(f"unknown event type: '{cmd}' '{arg}'")
                self._respond(rqueue, 400, f"unknown event type '{cmd}' '{arg}'")

    def _respond(self, rqueue: queue.Queue, status: int, info='') -> None:
        # self.logger.debug("responding to query with %s %s" % (status, info))
        rqueue.put((status, info))

    # load_config_fs reconfigures from the filesystem. It's _mostly_ legacy
    # code, but not entirely, since Docker demo mode still uses it.
    #
    # BE CAREFUL ABOUT STOPPING THE RECONFIGURATION TIMER ONCE IT IS STARTED.
    def load_config_fs(self, rqueue: queue.Queue, path: str) -> None:
        self.logger.debug("loading configuration from disk: %s" % path)

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

        # OK, we're starting a reconfiguration. BE CAREFUL TO STOP THE TIMER
        # BEFORE YOU RESPOND TO THE CALLER.
        self.app.config_timer.start()

        snapshot = re.sub(r'[^A-Za-z0-9_-]', '_', path)
        scc = FSSecretHandler(app.logger, path, app.snapshot_path, "0")

        with self.app.fetcher_timer:
            aconf = Config()
            fetcher = ResourceFetcher(app.logger, aconf)
            fetcher.load_from_filesystem(path, k8s=app.k8s, recurse=True)

        if not fetcher.elements:
            self.logger.debug("no configuration resources found at %s" % path)
            # Don't bail from here -- go ahead and reload the IR.
            #
            # XXX This is basically historical logic, honestly. But if you try
            # to respond from here and bail, STOP THE RECONFIGURATION TIMER.

        self._load_ir(rqueue, aconf, fetcher, scc, snapshot)

    # load_config_watt reconfigures from the filesystem. It's the one true way of
    # reconfiguring these days.
    #
    # BE CAREFUL ABOUT STOPPING THE RECONFIGURATION TIMER ONCE IT IS STARTED.
    def load_config_watt(self, rqueue: queue.Queue, url: str):
        snapshot = url.split('/')[-1]
        ss_path = os.path.join(app.snapshot_path, "snapshot-tmp.yaml")

        # OK, we're starting a reconfiguration. BE CAREFUL TO STOP THE TIMER
        # BEFORE YOU RESPOND TO THE CALLER.
        self.app.config_timer.start()

        self.logger.debug("copying configuration: watt, %s to %s" % (url, ss_path))

        # Grab the serialization, and save it to disk too.
        serialization = load_url_contents(self.logger, url, stream2=open(ss_path, "w"))

        if not serialization:
            self.logger.debug("no data loaded from snapshot %s" % snapshot)
            # We never used to return here. I'm not sure if that's really correct?
            #
            # IF YOU CHANGE THIS, BE CAREFUL TO STOP THE RECONFIGURATION TIMER.

        # Weirdly, we don't need a special WattSecretHandler: parse_watt knows how to handle
        # the secrets that watt sends.
        scc = SecretHandler(app.logger, url, app.snapshot_path, snapshot)

        # OK. Time the various configuration sections separately.

        with self.app.fetcher_timer:
            aconf = Config()
            fetcher = ResourceFetcher(app.logger, aconf)

            if serialization:
                fetcher.parse_watt(serialization)

        if not fetcher.elements:
            self.logger.debug("no configuration found in snapshot %s" % snapshot)

            # Don't actually bail here. If they send over a valid config that happens
            # to have nothing for us, it's still a legit config.
            #
            # IF YOU CHANGE THIS, BE CAREFUL TO STOP THE RECONFIGURATION TIMER.

        self._load_ir(rqueue, aconf, fetcher, scc, snapshot)

    # _load_ir is where the heavy lifting of a reconfigure happens.
    #
    # AT THE POINT OF ENTRY, THE RECONFIGURATION TIMER IS RUNNING. DO NOT LEAVE
    # THIS METHOD WITHOUT STOPPING THE RECONFIGURATION TIMER.
    def _load_ir(self, rqueue: queue.Queue, aconf: Config, fetcher: ResourceFetcher,
                 secret_handler: SecretHandler, snapshot: str) -> None:
        with self.app.aconf_timer:
            aconf.load_all(fetcher.sorted())

            # TODO(Flynn): This is an awful hack. Have aconf.load(fetcher) that does
            # this correctly.
            #
            # I'm not doing this at this moment because aconf.load_all() is called in a
            # lot of places, and I don't want to destablize 2.2.2.
            aconf.load_invalid(fetcher)

        aconf_path = os.path.join(app.snapshot_path, "aconf-tmp.json")
        open(aconf_path, "w").write(aconf.as_json())

        # OK. What kind of reconfiguration are we doing?
        config_type, reset_cache, invalidate_groups_for = IR.check_deltas(self.logger, fetcher, self.app.cache)

        if reset_cache:
            self.logger.debug("RESETTING CACHE")
            self.app.cache = Cache(self.logger)

        with self.app.ir_timer:
            ir = IR(aconf, secret_handler=secret_handler,
                    invalidate_groups_for=invalidate_groups_for, cache=self.app.cache)

        ir_path = os.path.join(app.snapshot_path, "ir-tmp.json")
        open(ir_path, "w").write(ir.as_json())

        with self.app.econf_timer:
            self.logger.debug("generating envoy configuration with api version %s" % Config.envoy_api_version)
            econf = EnvoyConfig.generate(ir, Config.envoy_api_version, cache=self.app.cache)

        # DON'T generate the Diagnostics here, because that turns out to be expensive.
        # Instead, we'll just reset app.diag to None, then generate it on-demand when
        # we need it.
        #
        # DO go ahead and split the Envoy config into its components for later, though.
        bootstrap_config, ads_config, clustermap = econf.split_config()

        # OK. Assume that the Envoy config is valid...
        econf_is_valid = True
        econf_bad_reason = ""

        # ...then look for reasons it's not valid.
        if not econf.has_listeners():
            # No listeners == something in the Ambassador config is totally horked.
            # Probably this is the user not defining any Hosts that match
            # the Listeners in the system.
            #
            # As it happens, Envoy is OK running a config with no listeners, and it'll
            # answer on port 8001 for readiness checks, so... log a notice, but run with it.
            self.logger.warning("No active listeners at all; check your Listener and Host configuration")
        elif not self.validate_envoy_config(ir, config=ads_config, retries=self.app.validation_retries):
            # Invalid Envoy config probably indicates a bug in Emissary itself. Sigh.
            econf_is_valid = False
            econf_bad_reason = "invalid envoy configuration generated"

        # OK. Is the config invalid?
        if not econf_is_valid:
            # BZzzt. Don't post this update.
            self.logger.info("no update performed (%s), continuing with current configuration..." % econf_bad_reason)

            # Don't use app.check_scout; it will deadlock.
            self.check_scout("attempted bad update")

            # DO stop the reconfiguration timer before leaving.
            self.app.config_timer.stop()
            self._respond(rqueue, 500, 'ignoring (%s) in snapshot %s' % (econf_bad_reason, snapshot))
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

                # Make sure we don't leave this method on error! The reconfiguration
                # timer is still running, but also, the snapshots are a debugging aid:
                # if we can't rotate them, meh, whatever.

                try:
                    self.logger.debug("rotate: %s -> %s" % (from_path, to_path))
                    os.rename(from_path, to_path)
                except IOError as e:
                    self.logger.debug("skip %s -> %s: %s" % (from_path, to_path, e))
                    pass
                except Exception as e:
                    self.logger.debug("could not rename %s -> %s: %s" % (from_path, to_path, e))

        app.latest_snapshot = snapshot
        self.logger.debug("saving Envoy configuration for snapshot %s" % snapshot)

        with open(app.bootstrap_path, "w") as output:
            output.write(dump_json(bootstrap_config, pretty=True))

        with open(app.ads_path, "w") as output:
            output.write(dump_json(ads_config, pretty=True))

        with open(app.clustermap_path, "w") as output:
            output.write(dump_json(clustermap, pretty=True))

        with app.config_lock:
            app.aconf = aconf
            app.ir = ir
            app.econf = econf

            # Force app.diag to None so that it'll be regenerated on-demand.
            app.diag = None

        # We're finally done with the whole configuration process.
        self.app.config_timer.stop()

        if app.kick:
            self.logger.debug("running '%s'" % app.kick)
            os.system(app.kick)
        elif app.ambex_pid != 0:
            self.logger.debug("notifying PID %d ambex" % app.ambex_pid)
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
                text = dump_json(update)

                # self.logger.debug(f"K8s status update: {kind} {resource_name}.{namespace}, {text}...")

                app.kubestatus.post(kind, resource_name, namespace, text)


        group_count = len(app.ir.groups)
        cluster_count = len(app.ir.clusters)
        listener_count = len(app.ir.listeners)
        service_count = len(app.ir.services)

        self._respond(rqueue, 200,
                      'configuration updated (%s) from snapshot %s' % (config_type, snapshot))

        self.logger.info("configuration updated (%s) from snapshot %s (S%d L%d G%d C%d)" %
                         (config_type, snapshot, service_count, listener_count, group_count, cluster_count))

        # Remember that we've reconfigured.
        self.app.reconf_stats.mark(config_type)

        if app.health_checks and not app.stats_updater:
            app.logger.debug("starting Envoy status updater")
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

                # Include features about the cache and incremental reconfiguration,
                # too.

                if self.app.cache is not None:
                    # Fast reconfigure is on. Supply the real info.
                    feat['frc_enabled'] = True
                    feat['frc_cache_hits'] = self.app.cache.hits
                    feat['frc_cache_misses'] = self.app.cache.misses
                    feat['frc_inv_calls'] = self.app.cache.invalidate_calls
                    feat['frc_inv_objects'] = self.app.cache.invalidated_objects
                else:
                    # Fast reconfigure is off.
                    feat['frc_enabled'] = False

                # Whether the cache is on or off, we can talk about reconfigurations.
                feat['frc_incr_count'] = self.app.reconf_stats.counts["incremental"]
                feat['frc_complete_count'] = self.app.reconf_stats.counts["complete"]
                feat['frc_check_count'] = self.app.reconf_stats.checks
                feat['frc_check_errors'] = self.app.reconf_stats.errors

                request_data = app.estatsmgr.get_stats().requests

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

        self.app.logger.debug("Scout reports %s" % dump_json(scout_result))
        self.app.logger.debug("Scout notices: %s" % dump_json(scout_notices))
        self.app.logger.debug("App notices after scout: %s" % dump_json(app.notices.notices))

    def validate_envoy_config(self, ir: IR, config, retries) -> bool:
        if self.app.no_envoy:
            self.app.logger.debug("Skipping validation")
            return True

        # We want to keep the original config untouched
        validation_config = copy.deepcopy(config)

        # Envoy fails to validate with @type field in envoy config, so removing that
        validation_config.pop('@type')

        if os.environ.get("AMBASSADOR_DEBUG_CLUSTER_CONFIG", "false").lower() == "true":
            vconf_clusters = validation_config['static_resources']['clusters']

            if len(vconf_clusters) > 10:
                vconf_clusters.append(copy.deepcopy(vconf_clusters[10]))

            # Check for cluster-name weirdness.
            _v2_clusters = {}
            _problems = []

            for name in sorted(ir.clusters.keys()):
                if AmbassadorEventWatcher.reCompressed.search(name):
                    _problems.append(f"IR pre-compressed cluster {name}")

            for cluster in validation_config['static_resources']['clusters']:
                name = cluster['name']

                if name in _v2_clusters:
                    _problems.append(f"V2 dup cluster {name}")
                _v2_clusters[name] = True

            if _problems:
                self.logger.error("ENVOY CONFIG PROBLEMS:\n%s", "\n".join(_problems))
                stamp = datetime.datetime.now().isoformat()

                bad_snapshot = open(os.path.join(app.snapshot_path, "snapshot-tmp.yaml"), "r").read()

                cache_dict: Dict[str, Any] = {}
                cache_links: Dict[str, Any] = {}

                if self.app.cache:
                    for k, c in self.app.cache.cache.items():
                        v: Any = c[0]

                        if getattr(v, 'as_dict', None):
                            v = v.as_dict()

                        cache_dict[k] = v

                    cache_links = { k: list(v) for k, v in self.app.cache.links.items() }

                bad_dict = {
                    "ir": ir.as_dict(),
                    "v2": config,
                    "validation": validation_config,
                    "problems": _problems,
                    "snapshot": bad_snapshot,
                    "cache": cache_dict,
                    "links": cache_links
                }

                bad_dict_str = dump_json(bad_dict, pretty=True)
                with open(os.path.join(app.snapshot_path, f"problems-{stamp}.json"), "w") as output:
                    output.write(bad_dict_str)

        config_json = dump_json(validation_config, pretty=True)

        econf_validation_path = os.path.join(app.snapshot_path, "econf-tmp.json")

        with open(econf_validation_path, "w") as output:
            output.write(config_json)

        command = ['envoy', '--service-node', 'test-id', '--service-cluster', ir.ambassador_nodename, '--config-path', econf_validation_path, '--mode', 'validate']

        v_exit = 0
        v_encoded = ''.encode('utf-8')

        # Try to validate the Envoy config. Short circuit and fall through
        # immediately on concrete success or failure, and retry (up to the
        # limit) on timeout.
        #
        # The default timeout is 5s, but this can be overridden in the Ambassador
        # module.

        amod = ir.ambassador_module
        timeout = amod.envoy_validation_timeout if amod else IRAmbassador.default_validation_timeout

        # If the timeout is zero, don't do the validation.
        if timeout == 0:
            self.logger.debug("not validating Envoy configuration since timeout is 0")
            return True

        self.logger.debug(f"validating Envoy configuration with timeout {timeout}")

        for retry in range(retries):
            try:
                v_encoded = subprocess.check_output(command, stderr=subprocess.STDOUT, timeout=timeout)
                v_exit = 0
                break
            except subprocess.CalledProcessError as e:
                v_exit = e.returncode
                v_encoded = e.output
                break
            except subprocess.TimeoutExpired as e:
                v_exit = 1
                v_encoded = e.output or ''.encode('utf-8')

                self.logger.warn("envoy configuration validation timed out after {} seconds{}\n{}".format(
                    timeout,', retrying...' if retry < retries - 1 else '', v_encoded.decode('utf-8'))
                )

                # Don't break here; continue on to the next iteration of the loop.

        if v_exit == 0:
            self.logger.debug("successfully validated the resulting envoy configuration, continuing...")
            return True

        v_str = typecast(str, v_encoded)

        try:
            v_str = v_encoded.decode('utf-8')
        except:
            pass

        self.logger.error("{}\ncould not validate the envoy configuration above after {} retries, failed with error \n{}\n(exit code {})\nAborting update...".format(config_json, retries, v_str, v_exit))
        return False


class StandaloneApplication(gunicorn.app.base.BaseApplication):
    def __init__(self, app, options=None):
        self.options = options or {}
        self.application = app
        super(StandaloneApplication, self).__init__()

        # Boot chime. This is basically the earliest point at which we can consider an Ambassador
        # to be "running".
        scout_result = self.application.scout.report(mode="boot", action="boot1", no_cache=True)
        self.application.logger.debug(f'BOOT: Scout result {dump_json(scout_result)}')
        self.application.logger.info(f'Ambassador {__version__} booted')

    def load_config(self):
        config = dict([ (key, value) for key, value in self.options.items()
                        if key in self.cfg.settings and value is not None ])

        for key, value in config.items():
            self.cfg.set(key.lower(), value)

    def load(self):
        # This is a little weird, but whatever.
        self.application.watcher = AmbassadorEventWatcher(self.application)
        self.application.watcher.start()

        if self.application.config_path:
            self.application.watcher.post("CONFIG_FS", self.application.config_path)

        return self.application


@click.command()
@click.argument('snapshot-path',      type=click.Path(), required=False)
@click.argument('bootstrap-path',     type=click.Path(), required=False)
@click.argument('ads-path',           type=click.Path(), required=False)
@click.option('--config-path',        type=click.Path(),                                 help="Optional configuration path to scan for Ambassador YAML files")
@click.option('--k8s',                is_flag=True,                                      help="If True, assume config_path contains Kubernetes resources (only relevant with config_path)")
@click.option('--ambex-pid',          type=int, default=0,                               help="Optional PID to signal with HUP after updating Envoy configuration", show_default=True)
@click.option('--kick',               type=str,                                          help="Optional command to run after updating Envoy configuration")
@click.option('--banner-endpoint',    type=str, default="http://127.0.0.1:8500/banner",  help="Optional endpoint of extra banner to include", show_default=True)
@click.option('--metrics-endpoint',   type=str, default="http://127.0.0.1:8500/metrics", help="Optional endpoint of extra prometheus metrics to include", show_default=True)
@click.option('--no-checks',          is_flag=True,                                      help="If True, don't do Envoy-cluster health checking")
@click.option('--no-envoy',           is_flag=True,                                      help="If True, don't interact with Envoy at all")
@click.option('--reload',             is_flag=True,                                      help="If True, run Flask in debug mode for live reloading")
@click.option('--debug',              is_flag=True,                                      help="If True, do debug logging")
@click.option('--dev-magic',          is_flag=True,                                      help="If True, override a bunch of things for Datawire dev-loop stuff")
@click.option('--verbose',            is_flag=True,                                      help="If True, do really verbose debug logging")
@click.option('--workers',            type=int,                                          help="Number of workers; default is based on the number of CPUs present")
@click.option('--host',               type=str,                                          help="Interface on which to listen")
@click.option('--port',               type=int, default=-1,                              help="Port on which to listen", show_default=True)
@click.option('--notices',            type=click.Path(),                                 help="Optional file to read for local notices")
@click.option('--validation-retries', type=int, default=5,                               help="Number of times to retry Envoy configuration validation after a timeout", show_default=True)
@click.option('--allow-fs-commands',  is_flag=True,                                      help="If true, allow CONFIG_FS to support debug/testing commands")
@click.option('--local-scout',        is_flag=True,                                      help="Don't talk to remote Scout at all; keep everything purely local")
@click.option('--report-action-keys', is_flag=True,                                      help="Report action keys when chiming")
def main(snapshot_path=None, bootstrap_path=None, ads_path=None,
          *, dev_magic=False, config_path=None, ambex_pid=0, kick=None,
          banner_endpoint="http://127.0.0.1:8500/banner", metrics_endpoint="http://127.0.0.1:8500/metrics", k8s=False,
          no_checks=False, no_envoy=False, reload=False, debug=False, verbose=False,
          workers=None, port=-1, host="", notices=None,
          validation_retries=5, allow_fs_commands=False, local_scout=False,
          report_action_keys=False):
    """
    Run the diagnostic daemon.

    Arguments:

      SNAPSHOT_PATH
        Path to directory in which to save configuration snapshots and dynamic secrets

      BOOTSTRAP_PATH
        Path to which to write bootstrap Envoy configuration

      ADS_PATH
        Path to which to write ADS Envoy configuration
    """

    enable_fast_reconfigure = parse_bool(os.environ.get("AMBASSADOR_FAST_RECONFIGURE", "true"))

    if port < 0:
        port = Constants.DIAG_PORT if not enable_fast_reconfigure else Constants.DIAG_PORT_ALT
        # port = Constants.DIAG_PORT

    if not host:
        host = '0.0.0.0' if not enable_fast_reconfigure else '127.0.0.1'

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
              metrics_endpoint, k8s, not no_checks, no_envoy, reload, debug, verbose, notices,
              validation_retries, allow_fs_commands, local_scout, report_action_keys,
              enable_fast_reconfigure)

    if not workers:
        workers = number_of_workers()

    gunicorn_config = {
        'bind': '%s:%s' % (host, port),
        # 'workers': 1,
        'threads': workers,
    }

    app.logger.info("thread count %d, listening on %s" % (gunicorn_config['threads'], gunicorn_config['bind']))

    StandaloneApplication(app, gunicorn_config).run()


if __name__ == "__main__":
    main()
