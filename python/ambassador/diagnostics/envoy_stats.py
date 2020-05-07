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
import logging
import threading
import time

import requests
from flask import Response

def percentage(x, y):
    if y == 0:
        return 0
    else:
        return int(((x * 100) / y) + 0.5)

class EnvoyStats (object):
    def __init__(self, stats, max_live_age=20, max_ready_age=20):
        self.stats = stats
        self.max_live_age = max_live_age
        self.max_ready_age = max_ready_age

    def is_alive(self):
        """
        Make sure we've heard from Envoy within max_live_age seconds. 

        If we haven't yet heard from Envoy at all (we've just booted),
        consider Envoy alive if we haven't yet been running for max_live_age
        seconds -- basically, Envoy gets a grace period to start running at
        boot time.
        """

        epoch = self.stats["last_update"]

        if not epoch:
            epoch = self.stats["created"]

        return (time.time() - epoch) <= self.max_live_age

    def is_ready(self):
        """
        Make sure we've heard from Envoy within max_ready_age seconds. 

        If we haven't yet heard from Envoy at all (we've just booted),
        then Envoy is not yet ready, and is_ready() returns False.
        """

        epoch = self.stats["last_update"]

        if not epoch:
            return False

        return (time.time() - epoch) <= self.max_ready_age

    def time_since_boot(self):
        """ Return the number of seconds since Envoy booted. """
        return time.time() - self.stats["created"]

    def time_since_update(self):
        """
        Return the number of seconds since we last heard from Envoy, or None if
        we've never heard from Envoy.
        """
        
        if self.stats["last_update"] == 0:
            return None
        else:
            return time.time() - self.stats["last_update"]

    def cluster_stats(self, name):
        if not self.stats['last_update']:
            # No updates.
            return { 
                'valid': False,
                'reason': "No stats updates have succeeded",
                'health': "no stats yet",
                'hmetric': 'startup',
                'hcolor': 'grey'
            }

        # OK, we should be OK.
        when = self.stats['last_update']
        cstat = self.stats['clusters']

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

class EnvoyStatsManager (object):
    def __init__(self, max_live_age=20, max_ready_age=20):
        self.max_live_age = max_live_age
        self.max_ready_age = max_ready_age
        self.loginfo = None
        self.lock = threading.Lock()

        self.stats = {
            "created": time.time(),
            "last_update": 0,
            "last_attempt": 0,
            "update_errors": 0,
            "services": {},
            "envoy": {}
        }

    def get_stats(self):
        with self.lock:
            return copy.deepcopy(self.stats)

    def update_log_levels(self, last_attempt, level=None):
        # logging.info("updating levels")

        try:
            url = "http://127.0.0.1:8001/logging"

            if level:
                url += "?level=%s" % level

            r = requests.post(url)
        except OSError as e:
            logging.warning("EnvoyStats.update_log_levels failed: %s" % e)
            self.stats['update_errors'] += 1
            return False

        # OMFG. Querying log levels returns with a 404 code.
        if (r.status_code != 200) and (r.status_code != 404):
            logging.warning("EnvoyStats.update_log_levels failed: %s" % r.text)
            self.stats['update_errors'] += 1
            return False   

        levels = {}

        for line in r.text.split("\n"):
            if not line:
                continue

            if line.startswith('  '):
                ( logtype, level ) = line[2:].split(": ")

                x = levels.setdefault(level, {})
                x[logtype] = True

        # logging.info("levels: %s" % levels)

        if len(levels.keys()) == 1:
            self.loginfo = { 'all': list(levels.keys())[0] }
        else:
            self.loginfo = { x: levels[x] for x in sorted(levels.keys()) }

        # logging.info("loginfo: %s" % self.loginfo)
        return True

    def get_prometheus_state(self):
        try:
            r = requests.get("http://127.0.0.1:8001/stats/prometheus")
        except OSError as e:
            logging.warning("EnvoyStats.get_prometheus_state failed: %s" % e)
            return Response("EnvoyStats.get_prometheus_state failed, OSError: %s" % e, 503)

        if r.status_code != 200:
            logging.warning("EnvoyStats.get_prometheus_state failed: %s" % r.text)
            return Response("EnvoyStats.get_prometheus_state failed: %s" % r.text, r.status_code)
        else:
            return Response(r.text, r.status_code, dict(r.headers))
        
    def _update_envoy_stats(self, last_attempt):
        # logging.info("updating stats")

        try:
            r = requests.get("http://127.0.0.1:8001/stats")
        except OSError as e:
            logging.warning("EnvoyStats.update failed: %s" % e)
            self.stats['update_errors'] += 1
            return

        if r.status_code != 200:
            logging.warning("EnvoyStats.update failed: %s" % r.text)
            self.stats['update_errors'] += 1
            return

        # Parse stats into a hierarchy.

        envoy_stats = {}

        for line in r.text.split("\n"):
            if not line:
                continue

            # logging.info('line: %s' % line)
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

                # logging.info("cluster %s stats: %s" % (cluster_name, cluster))

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

                # logging.info("%s total %s bad %s ok %s" % (cluster_name, upstream_total, upstream_bad, upstream_ok))

                if upstream_total > 0:
                    healthy_percent = percentage(upstream_ok, upstream_total)
                    # logging.debug("cluster %s is %d%% healthy" % (cluster_name, healthy_percent))
                else:
                    healthy_percent = None
                    # logging.debug("cluster %s has had no requests" % cluster_name)

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

        with self.lock:
            self.stats.update({
                "last_update": last_update,
                "last_attempt": last_attempt,
                "requests": requests_info,
                "clusters": active_clusters,
                "envoy": envoy_stats
            })

        # logging.info("stats updated")

    # def update(self, active_mapping_names):
    def update(self):
        try:
            # Remember when we started.
            last_attempt = time.time()

            self.update_log_levels(last_attempt)
            self._update_envoy_stats(last_attempt)
        except Exception as e:
            logging.error("could not update Envoy stats: %s" % e)
