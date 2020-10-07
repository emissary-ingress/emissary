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

from typing import Any, Dict, List, Optional, Union

import logging
import requests
import threading
import time

from dataclasses import dataclass
from dataclasses import field as dc_field

def percentage(x: float, y: float) -> int:
    if y == 0:
        return 0
    else:
        return int(((x * 100) / y) + 0.5)

@dataclass(frozen=True)
class EnvoyStats:
    max_live_age: int = 120
    max_ready_age: int = 120
    created: float = 0.0
    last_update: Optional[float] = None
    last_attempt: Optional[float] = None
    update_errors: int = 0

    # Yes yes yes I know -- the contents of these dicts are not immutable. 
    # That's OK for now, but realize that you mustn't go munging around altering
    # things in here once they're assigned!
    requests: Dict[str, Any] = dc_field(default_factory=dict)
    clusters: Dict[str, Any] = dc_field(default_factory=dict)
    envoy: Dict[str, Any] = dc_field(default_factory=dict)

    def is_alive(self) -> bool:
        """
        Make sure we've heard from Envoy within max_live_age seconds. 

        If we haven't yet heard from Envoy at all (we've just booted),
        consider Envoy alive if we haven't yet been running for max_live_age
        seconds -- basically, Envoy gets a grace period to start running at
        boot time.
        """

        epoch = self.last_update

        if not epoch:
            epoch = self.created

        return (time.time() - epoch) <= self.max_live_age

    def is_ready(self) -> bool:
        """
        Make sure we've heard from Envoy within max_ready_age seconds. 

        If we haven't yet heard from Envoy at all (we've just booted),
        then Envoy is not yet ready, and is_ready() returns False.
        """

        epoch = self.last_update

        if not epoch:
            return False

        return (time.time() - epoch) <= self.max_ready_age

    def time_since_boot(self) -> float:
        """ Return the number of seconds since Envoy booted. """
        return time.time() - self.created

    def time_since_update(self) -> Optional[float]:
        """
        Return the number of seconds since we last heard from Envoy, or None if
        we've never heard from Envoy.
        """
        
        if not self.last_update:
            return None
        else:
            return time.time() - self.last_update

    def cluster_stats(self, name: str) -> Dict[str, Union[str, bool]]:
        if not self.last_update:
            # No updates.
            return { 
                'valid': False,
                'reason': "No stats updates have succeeded",
                'health': "no stats yet",
                'hmetric': 'startup',
                'hcolor': 'grey'
            }

        # OK, we should be OK.
        when = self.last_update
        cstat = self.clusters

        if name not in cstat:
            return {
                'valid': False,
                'reason': "Cluster %s is not defined" % name,
                'health': "undefined cluster",
                'hmetric': 'undefined cluster',
                'hcolor': 'orange',
            }

        cstat = dict(**cstat[name])
        cstat.update({
            'valid': True,
            'reason': "Cluster %s updated at %d" % (name, when)
        })

        pct = cstat.get('healthy_percent', None)

        if pct != None:
            color = 'green'

            if pct < 70:
                color = 'red'
            elif pct < 90:
                color = 'yellow'

            cstat.update({
                'health': "%d%% healthy" % pct,
                'hmetric': int(pct),
                'hcolor': color
            })
        else:
            cstat.update({
                'health': "no requests yet",
                'hmetric': 'waiting',
                'hcolor': 'grey'
            })

        return cstat


class EnvoyStatsMgr:
    def __init__(self, logger: logging.Logger, max_live_age: int=120, max_ready_age: int=120,
        self.logger = logger
        self.loginfo: Dict[str, Union[str, List[str]]] = {}

        self.stats = EnvoyStats(
            created=time.time(), 
            max_live_age=max_live_age, 
            max_ready_age=max_ready_age
        )

    def update_log_levels(self, last_attempt, level=None):
        # self.logger.info("updating levels")

        failed = False

        try:
            url = "http://127.0.0.1:8001/logging"

            if level:
                url += "?level=%s" % level

            r = requests.post(url)

            # OMFG. Querying log levels returns with a 404 code.
            if (r.status_code != 200) and (r.status_code != 404):
                self.logger.warning("EnvoyStats.update_log_levels failed: %s" % r.text)
                failed = True
        except OSError as e:
            self.logger.warning("EnvoyStats.update_log_levels failed: %s" % e)
            failed = True
        
        if failed:
            # EnvoyStats is immutable, so...
            new_stats = EnvoyStats(
                max_live_age=self.stats.max_live_age,
                max_ready_age=self.stats.max_ready_age,
                created=self.stats.created,
                last_update=self.stats.last_update,
                last_attempt=last_attempt,                      # THIS IS A CHANGE
                update_errors=self.stats.update_errors + 1,     # THIS IS A CHANGE
                requests=self.stats.requests,
                clusters=self.stats.clusters,
                envoy=self.stats.envoy
            )

            self.stats = new_stats

            return False

        levels: Dict[str, Dict[str, bool]] = {}

        for line in r.text.split("\n"):
            if not line:
                continue

            if line.startswith('  '):
                ( logtype, level ) = line[2:].split(": ")

                x = levels.setdefault(level, {})
                x[logtype] = True

        # self.logger.info("levels: %s" % levels)

        loginfo: Dict[str, Union[str, List[str]]]

        if len(levels.keys()) == 1:
            loginfo = { 'all': list(levels.keys())[0] }
        else:
            loginfo = { x: list(levels[x].keys()) for x in levels.keys() }

        self.loginfo = loginfo

        # self.logger.info("loginfo: %s" % self.loginfo)
        return True

    def get_stats(self) -> EnvoyStats:
        return self.stats

    def get_prometheus_stats(self) -> str:
        try:
            r = requests.get("http://127.0.0.1:8001/stats/prometheus")
        except OSError as e:
            self.logger.warning("EnvoyStats.get_prometheus_state failed: %s" % e)
            return ''

        if r.status_code != 200:
            self.logger.warning("EnvoyStats.get_prometheus_state failed: %s" % r.text)
            return ''
        return r.text
        
    def update_envoy_stats(self, last_attempt: float) -> None:
        failed = False

        try:
            r = requests.get("http://127.0.0.1:8001/stats")

            if r.status_code != 200:
                self.logger.warning("EnvoyStats.update failed: %s" % r.text)
                failed = True
        except OSError as e:
            self.logger.warning("EnvoyStats.update failed: %s" % e)
            failed = True

        if failed:
            # EnvoyStats is immutable, so...
            new_stats = EnvoyStats(
                max_live_age=self.stats.max_live_age,
                max_ready_age=self.stats.max_ready_age,
                created=self.stats.created,
                last_update=self.stats.last_update,
                last_attempt=last_attempt,                    # THIS IS A CHANGE
                update_errors=self.stats.update_errors + 1,   # THIS IS A CHANGE
                requests=self.stats.requests,
                clusters=self.stats.clusters,
                envoy=self.stats.envoy
            )

            self.stats = new_stats
            return

        # Parse stats into a hierarchy.
        envoy_stats: Dict[str, Any] = {}    # Ew.

        for line in r.text.split("\n"):
            if not line:
                continue

            # self.logger.info('line: %s' % line)
            key, value = line.split(":")
            keypath = key.split('.')

            node = envoy_stats

            for key in keypath[:-1]:
                if key not in node:
                    node[key] = {}

                node = node[key]

            value = value.strip()

            # Skip histograms for the moment.
            # if value.startswith("P0("):
            #     continue
            #     # for field in value.split(' '):
            #     #     if field.startswith('P95('):
            #     #         value = field.split(',')

            try:
                node[keypath[-1]] = int(value)
            except:
                continue

        # Now dig into clusters a bit more.

        requests_info = {}
        active_clusters = {}

        if ("http" in envoy_stats) and ("ingress_http" in envoy_stats["http"]):
            ingress_stats = envoy_stats["http"]["ingress_http"]

            requests_total = ingress_stats.get("downstream_rq_total", 0)

            requests_4xx = ingress_stats.get('downstream_rq_4xx', 0)
            requests_5xx = ingress_stats.get('downstream_rq_5xx', 0)
            requests_bad = requests_4xx + requests_5xx

            requests_ok = requests_total - requests_bad

            requests_info = {
                "total": requests_total,
                "4xx": requests_4xx,
                "5xx": requests_5xx,
                "bad": requests_bad,
                "ok": requests_ok,
            }

        if "cluster" in envoy_stats:
            for cluster_name in envoy_stats['cluster']:
                cluster = envoy_stats['cluster'][cluster_name]

                # # Toss any _%d -- that's madness with our Istio code at the moment.
                # cluster_name = re.sub('_\d+$', '', cluster_name)

                # mapping_name = active_cluster_map[cluster_name]
                # active_mappings[mapping_name] = {}

                # self.logger.info("cluster %s stats: %s" % (cluster_name, cluster))

                healthy_percent: Optional[int]
                
                healthy_members = cluster['membership_healthy']
                total_members = cluster['membership_total']
                healthy_percent = percentage(healthy_members, total_members)

                update_attempts = cluster['update_attempt']
                update_successes = cluster['update_success']
                update_percent = percentage(update_successes, update_attempts)

                # Weird.
                # upstream_ok = cluster.get('upstream_rq_2xx', 0)
                # upstream_total = cluster.get('upstream_rq_pending_total', 0)
                upstream_total = cluster.get('upstream_rq_completed', 0)

                upstream_4xx = cluster.get('upstream_rq_4xx', 0)
                upstream_5xx = cluster.get('upstream_rq_5xx', 0)
                upstream_bad = upstream_5xx # used to include 4XX here, but that seems wrong.

                upstream_ok = upstream_total - upstream_bad

                # self.logger.info("%s total %s bad %s ok %s" % (cluster_name, upstream_total, upstream_bad, upstream_ok))

                if upstream_total > 0:
                    healthy_percent = percentage(upstream_ok, upstream_total)
                    # self.logger.debug("cluster %s is %d%% healthy" % (cluster_name, healthy_percent))
                else:
                    healthy_percent = None
                    # self.logger.debug("cluster %s has had no requests" % cluster_name)

                active_clusters[cluster_name] = {
                    'healthy_members': healthy_members,
                    'total_members': total_members,
                    'healthy_percent': healthy_percent,

                    'update_attempts': update_attempts,
                    'update_successes': update_successes,
                    'update_percent': update_percent,

                    'upstream_ok': upstream_ok,
                    'upstream_4xx': upstream_4xx,
                    'upstream_5xx': upstream_5xx,
                    'upstream_bad': upstream_bad
                }

        # OK, we're now officially finished with all the hard stuff.
        last_update = time.time()

        # Finally, set up the new EnvoyStats.
        new_stats = EnvoyStats(
            max_live_age=self.stats.max_live_age,
            max_ready_age=self.stats.max_ready_age,
            created=self.stats.created,
            last_update=last_update,                    # THIS IS A CHANGE
            last_attempt=last_attempt,                  # THIS IS A CHANGE
            update_errors=self.stats.update_errors,
            requests=requests_info,                     # THIS IS A CHANGE
            clusters=active_clusters,                   # THIS IS A CHANGE
            envoy=envoy_stats                           # THIS IS A CHANGE
        )

        # Make sure we hold the access_lock while messing with self.stats!
        self.stats = new_stats

        # self.logger.info("stats updated")

    def update(self) -> None:
        # self.logger.info("updating estats")

        try:
            # Remember when we started.
            last_attempt = time.time()

            self.update_log_levels(last_attempt)
            self.update_envoy_stats(last_attempt)
        except Exception as e:
            self.logger.error("could not update Envoy stats: %s" % e)
