import copy
import json
import logging
import time

import dpath
import requests


def percentage(x, y):
    if y == 0:
        return 0
    else:
        return int(((x * 100) / y) + 0.5)


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

    def add_service(self, name, prefix):
        self.services[name] = {
            'prefix': prefix,
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


class EnvoyStats (object):
    def __init__(self):
        self.update_errors = 0
        self.stats = {
            "last_update": 0,
            "last_attempt": 0,
            "update_errors": 0,
            "services": {},
            "envoy": {}
        }

    def update(self, active_service_names):
        # Remember how many update errors we had before...
        update_errors = self.stats['update_errors']

        # ...and remember when we started.
        last_attempt = time.time()

        r = requests.get("http://127.0.0.1:8001/stats")

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

            node[keypath[-1]] = int(value.strip())

        # Now dig into clusters a bit more.

        active_services = {}

        if "cluster" in envoy_stats:
            active_service_map = {
                x + '_cluster': x
                for x in active_service_names
            }

            for cluster_name in envoy_stats['cluster']:
                cluster = envoy_stats['cluster'][cluster_name]

                if cluster_name in active_service_map:
                    service_name = active_service_map[cluster_name]
                    active_services[service_name] = {}

                    logging.info("SVC %s => CLUSTER %s" % (service_name, json.dumps(cluster)))

                    healthy_members = cluster['membership_healthy']
                    total_members = cluster['membership_total']
                    healthy_percent = percentage(healthy_members, total_members)

                    update_attempts = cluster['update_attempt']
                    update_successes = cluster['update_success']
                    update_percent = percentage(update_successes, update_attempts)

                    upstream_ok = cluster.get('upstream_rq_2xx', 0)
                    upstream_4xx = cluster.get('upstream_rq_4xx', 0)
                    upstream_5xx = cluster.get('upstream_rq_5xx', 0)
                    upstream_bad = upstream_4xx + upstream_5xx

                    active_services[service_name] = {
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

        self.stats = {
            "last_update": last_update,
            "last_attempt": last_attempt,
            "update_errors": update_errors,
            "services": active_services,
            "envoy": envoy_stats
        }
