import copy
import json
import logging
import os
import re
import time

import dpath
import requests

TOKEN = open("/var/run/secrets/kubernetes.io/serviceaccount/token", "r").read()
SERVICE_URL = "https://kubernetes/api/v1/namespaces/default/services"

def percentage(x, y):
    if y == 0:
        return 0
    else:
        return int(((x * 100) / y) + 0.5)


class TLSConfig (object):
    def __init__(self, chain_path=None, privkey_path=None, cacert_chain_path=None):
        self.chain_path = self.check_path(chain_path)
        self.privkey_path = self.check_path(privkey_path)
        self.cacert_chain_path = self.check_path(cacert_chain_path)

    def check_path(self, pathinfo):
        if not pathinfo:
            logging.error("pathinfo missing??")
            return None

        if not "paths" in pathinfo:
            logging.error("pathinfo missing paths??")
            return None

        default = pathinfo["paths"][-1]

        if "env" in pathinfo:
            tmp = os.environ.get(pathinfo["env"], None)

            if tmp:
                return tmp

        if "paths" in pathinfo:
            for path in pathinfo["paths"]:
                if self.check_file(path):
                    return path

        return default

    def check_file(self, path):
        found = False

        try:
            statinfo = os.stat(path)
            found = True
        except FileNotFoundError:
            pass

        return found

    def config_block(self):
        config = {}

        if (self.check_file(self.chain_path) and
            self.check_file(self.privkey_path)):
            config["cert_chain_file"] = self.chain_path
            config["private_key_file"] = self.privkey_path

        if self.check_file(self.cacert_chain_path):
            config["ca_cert_file"] = self.cacert_chain_path

        return config

    def cacert_config_block(self):
        if self.check_file(self.cacert_chain_path):
            return {
              "type": "read",
              "name": "client_ssl_auth",
              "config": {
                "auth_api_cluster": "ambassador_cluster",
                "stat_prefix": "main_tls_auth"
              }
            }
        else:
            return None

class EnvoyConfig (object):
    route_template = '''
    {{
        "timeout_ms": 0,
        "prefix": "{url_prefix}",
        "prefix_rewrite": "{rewrite_prefix_as}",
        "cluster": "{cluster_name}"
    }}
    '''

    # We may append the 'features' element to the cluster definition, too.
    #
    # We can switch back to SDS later.
    # "type": "sds",
    # "service_name": "{service_name}",
    #
    # At that time we'll need to reinstate the SDS cluster in envoy-template.json:
    #
    # "sds": {
    #   "cluster": {
    #     "name": "ambassador-sds",
    #     "connect_timeout_ms": 250,
    #     "type": "strict_dns",
    #     "lb_type": "round_robin",
    #     "hosts": [
    #       {
    #         "url": "tcp://ambassador-sds:5000"
    #       }
    #     ]
    #   },
    #   "refresh_delay_ms": 15000
    # },

    cluster_template = '''
    {{
        "name": "{cluster_name}",
        "connect_timeout_ms": 250,
        "lb_type": "round_robin",
        "type": "strict_dns",
        "hosts": []
    }}
    '''

    host_template = '''
    {{
        "url": "tcp://{service_name}:{port}"
    }}
    '''

    ext_auth_filter_template = '''
    {{
        "type": "decoder",
        "name": "extauth",
        "config": {{
            "cluster": "ext_auth",
            "timeout_ms": 5000
        }}
    }}
    '''    

    ext_auth_cluster_template = '''
    {{
        "name": "ext_auth",
        "type": "logical_dns",
        "connect_timeout_ms": 5000,
        "lb_type": "random",
        "hosts": [
            {{
                "url": "tcp://{ext_auth_target}"
            }}
        ]
    }}
    '''

    self_routes = [
        # {
        #     "timeout_ms": 0,
        #     "prefix": "/ambassador/",
        #     "cluster": "ambassador_cluster"
        # },
        # {
        #     "timeout_ms": 0,
        #     "prefix": "/v1/",
        #     "cluster": "ambassador_cluster"
        # },
        # {
        #     "timeout_ms": 0,
        #     "prefix": "/ambassador-config/",
        #     "prefix_rewrite": "/",
        #     "cluster": "ambassador_config_cluster"
        # }
    ]

    self_clusters = [
        {
            "name": "ambassador_cluster",
            "connect_timeout_ms": 250,
            "type": "static",
            "lb_type": "round_robin",
            "hosts": [
                {
                    "url": "tcp://127.0.0.1:8888"
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

    def __init__(self, base_config, tls_config, current_modules):
        self.mappings = []
        self.base_config = base_config
        self.tls_config = tls_config
        self.current_modules = current_modules

        self.ext_auth_target = None

        auth_config = self.current_modules.get('authentication', None)

        if auth_config:
            try:
                ambassador_auth = auth_config.get('ambassador', None)

                if ambassador_auth:
                    # Use the auth module built in to Ambassador.
                    self.ext_auth_target = '127.0.0.1:5000'
                else:
                    # Look in the config itself for the target.
                    self.ext_auth_target = auth_config.get('auth_service', None)
            except Exception as e:
                # This can't really happen except for the case where auth_config
                # isn't a dict, and that's unsupported.
                logging.warning("authentication module has unsupported config '%s'" % json.dumps(auth_config))
                pass                

    def add_mapping(self, name, prefix, service, rewrite, modules):
        logging.debug("adding mapping %s (%s -> %s)" % (name, prefix, service))
        
        self.mappings.append({
            'name': name,
            'prefix': prefix,
            'service': service,
            'rewrite': rewrite,
            'modules': modules
        })

    def write_config(self, path):
        # Generate routes and clusters.
        routes = copy.deepcopy(EnvoyConfig.self_routes)
        clusters = copy.deepcopy(EnvoyConfig.self_clusters)
        in_istio = False
        istio_services = {}

        logging.info("writing Envoy config to %s" % path)
        logging.info("initial routes: %s" % routes)
        logging.info("initial clusters: %s" % clusters)

        # Grab service info from Kubernetes.
        r = requests.get(SERVICE_URL, headers={"Authorization": "Bearer " + TOKEN}, 
                         verify=False)

        if r.status_code != 200:
            # This can't be good.
            raise Exception("couldn't query Kubernetes for services! %s" % r)

        services = r.json()

        items = services.get('items', [])

        service_info = {}

        for item in items:
            service_name = None
            portspecs = []

            try:
                service_name = dpath.util.get(item, "/metadata/name")
            except KeyError:
                pass

            if service_name.startswith('istio-'):
                # This is an Istio service. Remember that we've seen it...
                istio_services[service_name] = True

                # ...and continue, we needn't do anything else.
                continue

            try:
                portspecs = dpath.util.get(item, "/spec/ports")
            except KeyError:
                pass

            if service_name and portspecs:
                service_info[service_name] = portspecs

        # OK. Are we running in an Istio cluster?
        if (('istio-ingress' in istio_services) and
            ('istio-manager' in istio_services) and
            ('istio-mixer' in istio_services)):
            in_istio = True

        for mapping in self.mappings:
            # Does this mapping refer to a service that we know about?
            mapping_name = mapping['name']
            prefix = mapping['prefix']
            service_name = mapping['service']
            rewrite = mapping['rewrite']
            modules = mapping.get('modules', {})

            if service_name in service_info:
                portspecs = service_info[service_name]

                istio_string = " (in Istio)" if in_istio else ""

                logging.info("mapping %s%s: pfx %s => svc %s, portspecs %s, modules %s" %
                             (mapping_name, istio_string, prefix, service_name, portspecs, modules))

                # OK, blatant hackery coming up here.
                #
                # Here's the problem: when we're running in Istio, at the moment we 
                # have to force the Host: header to match _exactly_ the destination, 
                # including the host name _and the port number_. This will change later
                # in Istio, but for support right now, it's what we need to do.
                #
                # The problem is that services might include multiple ports, and we'll
                # need to rewrite differently for the different ports. Right now, we'll
                # hack around this by using multiple clusters. Ohhhh the joy.

                cluster_number = 0

                for portspec in portspecs:
                    pspec = { "service_name": service_name }
                    pspec.update(portspec)

                    # OK, what's the target host spec here?
                    host_name_and_port = "%s:%d" % (service_name, pspec['port'])

                    # How should we write that in a cluster definition?
                    host_json = EnvoyConfig.host_template.format(**pspec)

                    # What is the cluster's name? (Note that we name clusters after the
                    # mapping, not the service.)
                    cluster_name = "%s_cluster_%d" % (mapping_name, cluster_number)

                    # What's the spec for our back-end service look like?
                    service_def = {
                        'service_name': service_name,
                        'url_prefix': prefix,
                        'rewrite_prefix_as': rewrite,
                        'cluster_name': cluster_name
                    }

                    # OK. We can build up the cluster definition now...
                    cluster_json = EnvoyConfig.cluster_template.format(**service_def)

                    cluster = json.loads(cluster_json)
                    cluster['hosts'] = [ json.loads(host_json) ]

                    if 'grpc' in modules:
                        cluster['features'] = 'http2'

                    # ...and we can write a routing entry that routes to that cluster...
                    route_json = EnvoyConfig.route_template.format(**service_def)

                    route = json.loads(route_json)

                    # ...including a host_rewrite rule if we need it.
                    if in_istio:
                        route['host_rewrite'] = host_name_and_port

                    # Once that's done, save the route...
                    logging.info("add route %s" % route)
                    routes.append(route)

                    # ...and save the cluster.
                    logging.info("add cluster %s" % cluster)
                    clusters.append(cluster)

                    # Finally, bump the cluster number.
                    cluster_number += 1

        # OK. Spin out the config.
        config = copy.deepcopy(self.base_config)

        if self.ext_auth_target:
            logging.info("enabling ext_auth to %s" % self.ext_auth_target)

            filt0name = dpath.util.get(config, "/listeners/0/filters/0/name")

            if filt0name != 'http_connection_manager':
                msg = "expected httpconnman as /listeners/0/filters/0, got %s?" % filt0name
                raise Exception(msg)

            ext_auth_def = {
                'ext_auth_target': self.ext_auth_target
            }

            ext_auth_filter_json = EnvoyConfig.ext_auth_filter_template.format(**ext_auth_def)

            logging.debug("ext_auth_filter %s" % ext_auth_filter_json)

            ext_auth_filter = json.loads(ext_auth_filter_json)
            filter_set = dpath.util.get(config, "/listeners/0/filters/0/config/filters")
            filter_set.insert(0, ext_auth_filter)

            ext_auth_cluster_json = EnvoyConfig.ext_auth_cluster_template.format(**ext_auth_def)

            logging.debug("ext_auth_cluster %s" % ext_auth_cluster_json)

            ext_auth_cluster = json.loads(ext_auth_cluster_json)
            clusters.append(ext_auth_cluster)

        logging.info("final routes: %s" % routes)
        logging.info("final clusters: %s" % clusters)

        # listeners = config.get("listeners", [])

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

        ssl_context = self.tls_config.config_block()

        if ssl_context:
            logging.info("configuring with TLS")

            dpath.util.new(
                config,
                "/listeners/0/ssl_context",
                ssl_context
            )

            dpath.util.set(
                config,
                "/listeners/0/address",
                "tcp://0.0.0.0:443"
            )

            client_cert_config = self.tls_config.cacert_config_block()

            if client_cert_config:
                dpath.util.get(config, "/listeners/0/filters").append(client_cert_config)
        else:
            logging.info("configuring plaintext-only")

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

    def update(self, active_mapping_names):
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

        active_mappings = {}

        if "cluster" in envoy_stats:
            active_cluster_map = {
                x + '_cluster': x
                for x in active_mapping_names
            }

            logging.info("active_cluster_map: %s" % json.dumps(active_cluster_map))

            for cluster_name in envoy_stats['cluster']:
                cluster = envoy_stats['cluster'][cluster_name]

                # Toss any _%d -- that's madness with our Istio code at the moment.
                cluster_name = re.sub('_\d+$', '', cluster_name)
                
                if cluster_name in active_cluster_map:
                    mapping_name = active_cluster_map[cluster_name]
                    active_mappings[mapping_name] = {}

                    logging.info("SVC %s has cluster" % mapping_name)

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

                    active_mappings[mapping_name] = {
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
            "mappings": active_mappings,
            "envoy": envoy_stats
        }
