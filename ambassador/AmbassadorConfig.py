import sys

import collections
import json
import logging
import os
import re
import yaml

from jinja2 import Environment, FileSystemLoader
from utils import RichStatus

class AmbassadorConfig (object):
    def __init__(self, config_dir_path):
        self.config_dir_path = config_dir_path
        self.config = {}
        self.envoy_config = {}
        self.envoy_clusters = {}

        for dirpath, dirnames, filenames in os.walk(self.config_dir_path):
            for filename in [ x for x in filenames if x.endswith(".yaml") ]:
                filepath = os.path.join(dirpath, filename)

                try:
                    obj = yaml.safe_load(open(filepath, "r"))
                except Exception as e:
                    logging.error("%s: could not parse YAML: %s" % (filepath, e))
                    continue

                # The object must be a dict with a single toplevel key.
                if not isinstance(obj, collections.Mapping):
                    raise Exception("%s: not a dictionary" % filepath)

                if len(obj.keys()) != 1:
                    raise Exception("%s: need exactly one toplevel key, found %d" %
                                    (filepath, len(obj.keys())))

                toplevel = list(obj.keys())[0]
                handler_name = "handle_%s" % toplevel
                handler = getattr(self, handler_name, None)

                if not handler:
                    logging.warning("%s: no handler for %s, skipping" % (filepath, toplevel))
                    continue

                logging.debug("%s: handling %s..." % (filename, toplevel))

                handler(filename, toplevel, obj)

        self.generate_envoy_config()

    def handle_ambassador(self, filename, toplevel, obj):
        # Just save this for now.
        self.config['ambassador'] = obj['ambassador']

    def handle_module(self, filename, toplevel, obj):
        # This is a single module definition. The name of the module
        # comes from the name of the file.

        m = re.match(r'^module-([a-zA-Z][a-zA-Z0-9_]*)\.[^.]*$', filename)

        if not m:
            raise Exception("parsing %s: cannot infer module name" % filename)

        module_name = m.group(1)

        logging.debug("%s: module name is %s" % (filename, module_name))

        modules = self.config.setdefault("modules", {})
        modules[module_name] = obj['module']

    def handle_mappings(self, filename, toplevel, obj):
        mappings = self.config.setdefault("mappings", {})

        # Support both 'mapping' and 'mappings'...
        for mapping_name in obj[toplevel].keys():
            logging.debug("%s: saving mapping %s" % (filename, mapping_name))

            mappings[mapping_name] = obj[toplevel][mapping_name]

    def handle_mapping(self, filename, toplevel, obj):
        return self.handle_mappings(filename, toplevel, obj)

    def generate_envoy_config(self):
        # First things first. Assume we'll listen on port 80.
        service_port = 80

        # OK. Is TLS configured?
        aconf = self.config.get('ambassador', {})

        if 'tls' in aconf:
            # Yes. Switch to port 443 by default...
            service_port = 443

            # ...and copy the TLS config to the envoy_config dict.
            self.envoy_config['tls'] = aconf['tls']

        # OK. Copy listener data into the envoy_config dict.
        self.envoy_config['listeners'] = {
            "service_port": service_port,
            "admin_port": aconf.get('admin_port', 8001)
        }

        # Next up: let's define initial clusters, routes, and filters.
        #
        # Our initial set of clusters is just the ambassador config interface.
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.
        self.envoy_clusters = {
            "ambassador_config_cluster": {
                "name": "ambassador_config_cluster",
                "type": "static",
                "urls": [ "tcp://127.0.0.1:8001" ]
            }
        }

        # ...our initial set of routes is empty...
        self.envoy_config['routes'] = []

        # ...and our initial set of filters is just the 'router' filter.
        #
        # !!!! WARNING WARNING WARNING !!!! Filters are actually ORDER-DEPENDENT.
        # We're kind of punting on that so far since we'll only ever add one filter
        # right now.
        self.envoy_config['filters'] = [
            {
                "type": "decoder",
                "name": "router",
                "config": {}
            }
        ]

        # OK. Given those initial sets, let's look over our global modules.
        modules = self.config.get('modules', {})

        for module_name in modules.keys():
            handler_name = "module_config_%s" % module_name
            handler = getattr(self, handler_name, None)

            if not handler:
                print("module %s: no configuration generator, skipping" % module_name)
                continue

            handler(module_name, modules[module_name])

        # Once that's done, it's time to wrangle mappings. These must be sorted by prefix
        # length...
        mappings = self.config.get("mappings", [])

        for mapping_name in sorted(mappings.keys(), key=lambda x: len(mappings[x]['prefix'])):
            mapping = mappings[mapping_name]

            # OK. We need a cluster for this service. Derive it from the 
            # service name.
            svc = mapping['service']

            # Use just the DNS name for the cluster -- if a port was included, drop it.
            svc_name_only = svc.split(':')[0]
            cluster_name = '%s_cluster' % svc_name_only
            cluster_name = re.sub(r'[^0-9A-Za-z_]', '_', cluster_name)

            logging.debug("%s: svc %s -> cluster %s" % (mapping_name, svc, cluster_name))

            if cluster_name not in self.envoy_clusters:
                url = 'tcp://%s' % svc

                if ':' not in svc:
                    url += ':80'

                self.envoy_clusters[cluster_name] = {
                    "name": cluster_name,
                    "type": "strict_dns",
                    "lb_type": "round_robin",
                    "urls": [ url ]
                }

            route = {
                "prefix": mapping['prefix'],
                "prefix_rewrite": mapping.get('rewrite', '/'),
                "cluster": cluster_name
            }

            if 'method' in mapping:
                route['method'] = mapping['method']
                route['method_regex'] = route.get('method_regex', False)

            if 'timeout_ms' in mapping:
                route['timeout_ms'] = mapping['timeout_ms']

            self.envoy_config['routes'].append(route)

        # OK. When all is said and done, map the cluster set back into a list.
        self.envoy_config['clusters'] = [
            self.envoy_clusters[name] for name in sorted(self.envoy_clusters.keys())
        ]

    def module_config_authentication(self, name, module):
        filter = {
            "type": "decoder",
            "name": "extauth",
            "config": {
                "cluster": "ext_auth_cluster",
                "timeout_ms": 5000
            }
        }

        path_prefix = module.get("path_prefix", None)

        if path_prefix:
            filter['config']['path_prefix'] = path_prefix

        allowed_headers = module.get("allowed_headers", None)

        if allowed_headers:
            filter['config']['allowed_headers'] = allowed_headers

        self.envoy_config['filters'].insert(0, filter)

        if 'ext_auth_cluster' not in self.envoy_clusters:
            svc = module.get('auth_service', '127.0.0.1:5000')

            self.envoy_clusters['ext_auth_cluster'] = {
                "name": "ext_auth_cluster",
                "type": "logical_dns",
                "connect_timeout_ms": 5000,
                "lb_type": "random",
                "urls": [ "tcp://%s" % svc ]
            }

    def pretty(self, obj, out=sys.stdout):
        json.dump(obj, out, indent=4, separators=(',',':'), sort_keys=True)
        out.write("\n")

    def to_json(self, template=None, template_dir="templates"):
        if not template:
            env = Environment(loader=FileSystemLoader("templates"))
            template = env.get_template("envoy.j2")

        return(template.render(**self.envoy_config))

    def envoy_config_object(self, **kwargs):
        envoy_json = self.to_json(**kwargs)
        rc = RichStatus.fromError("impossible")

        try:
            obj = json.loads(envoy_json)
            rc = RichStatus.OK(msg="Envoy configuration OK", envoy_config=obj)
        except json.decoder.JSONDecodeError as e:
            rc = RichStatus.fromError("Invalid Envoy configuration: %s" % str(e),
                                      raw=envoy_json, exception=e)

        return rc

    def dump(self):
        print("==== config")
        self.pretty(self.config)

        print("==== envoy_config")
        self.pretty(self.envoy_config)
