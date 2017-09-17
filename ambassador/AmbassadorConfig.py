import sys

import collections
import json
import jsonschema
import logging
import os
import re
import yaml

from jinja2 import Environment, FileSystemLoader
from utils import RichStatus

class AmbassadorConfig (object):
    def __init__(self, config_dir_path,
                 schema_dir_path="schemas", template_dir_path="templates"):
        self.config_dir_path = config_dir_path
        self.schema_dir_path = schema_dir_path
        self.template_dir_path = template_dir_path

        self.schemas = {}
        self.config = {}
        self.envoy_config = {}
        self.envoy_clusters = {}

        for dirpath, dirnames, filenames in os.walk(self.config_dir_path):
            for filename in [ x for x in filenames if x.endswith(".yaml") ]:
                self.filename = filename

                filepath = os.path.join(dirpath, filename)

                try:
                    objects = yaml.safe_load_all(open(filepath, "r"))
                except Exception as e:
                    logging.error("%s: could not parse YAML: %s" % (filepath, e))
                    continue

                self.ocount = 0
                for obj in objects:
                    self.ocount += 1

                    self.process_object(obj)

        self.generate_envoy_config()

    def process_object(self, obj):
        # OK. What is this thing?
        obj_kind, obj_version = self.validate_object(obj)

        handler_name = "handle_%s" % obj_kind.lower()
        handler = getattr(self, handler_name, None)

        if not handler:
            handler = self.save_object
            logging.warning("%s[%d]: no handler for %s, just saving" % (self.filename, self.ocount, obj_kind))
        else:
            logging.debug("%s[%d]: handling %s..." % (self.filename, self.ocount, obj_kind))

        handler(obj, obj_kind, obj_version)

    def validate_object(self, obj):
        # Each object must be a dict, and must include "apiVersion"
        # and "type" at toplevel.

        if not isinstance(obj, collections.Mapping):
            raise Exception("%s[%d]: not a dictionary" % (self.filename, self.ocount))

        if not (("apiVersion" in obj) and ("kind" in obj)):
            raise Exception("%s[%d]: must have apiVersion and kind" % (self.filename, self.ocount))

        obj_version = obj['apiVersion']
        obj_kind = obj['kind']

        if obj_version.startswith("ambassador/"):
            obj_version = obj_version.split('/')[1]
        else:
            raise Exception("%s[%d]: apiVersion %s unsupported" % (self.filename, self.ocount, obj_version))

        schema_key = "%s-%s" % (obj_version, obj_kind)

        schema = self.schemas.get(schema_key, None)

        if not schema:
            schema_path = os.path.join(self.schema_dir_path, obj_version, "%s.schema" % obj_kind)

            try:
                schema = json.load(open(schema_path, "r"))
            except OSError:
                logging.debug("no schema at %s, skipping" % schema_path)
            except json.decoder.JSONDecodeError as e:
                logging.warning("corrupt schema at %s, skipping (%s)" % (schema_path, e))

        if schema:
            self.schemas[schema_key] = schema
            try:
                jsonschema.validate(obj, schema)
            except jsonschema.exceptions.ValidationError as e:
                raise Exception("%s[%d] is not a valid %s: %s" % 
                                (self.filename, self.ocount, obj_kind, e))

        return (obj_kind, obj_version)

    def save_object(self, obj, obj_kind, obj_version):
        logging.debug("%s[%d]: saving %s %s" %
                      (self.filename, self.ocount, obj_kind,  obj['name']))
        objects = self.config.setdefault(obj_kind, {})
        objects[obj['name']] = obj

    def handle_module(self, obj, obj_kind, obj_version):
        logging.debug("%s[%d]: saving module %s" % (self.filename, self.ocount, obj['name']))
        modules = self.config.setdefault("modules", {})
        modules[obj['name']] = obj['config']

    def handle_mapping(self, obj, obj_kind, obj_version):
        logging.debug("%s[%d]: saving mapping %s" % (self.filename, self.ocount, obj['name']))
        mappings = self.config.setdefault("mappings", {})
        mappings[obj['name']] = obj

    def generate_envoy_config(self):
        # First things first. Assume we'll listen on port 80.
        service_port = 80

        # OK. Is TLS configured?
        aconf = self.config.get('Ambassador', {})

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
        breakers = self.config.get("CircuitBreaker", {})
        outliers = self.config.get("OutlierDetection", {})

        for mapping_name in sorted(mappings.keys(), key=lambda x: len(mappings[x]['prefix'])):
            mapping = mappings[mapping_name]

            # OK. We need a cluster for this service. Derive it from the 
            # service name, plus things like circuit breaker and outlier 
            # detection settings.
            svc = mapping['service']

            # Use just the DNS name for the cluster -- if a port was included, drop it.
            svc_name_only = svc.split(':')[0]
            cluster_name_fields =[ svc_name_only ]

            cb_name = mapping.get('circuit_breaker', None)

            if cb_name:
                if cb_name in breakers:
                    cluster_name_fields.append("cb_%s" % cb_name)
                else:
                    logging.error("CircuitBreaker %s is not defined (mapping %s)" %
                                  (cb_name, mapping_name))

            od_name = mapping.get('outlier_detection', None)

            if od_name:
                if od_name in outliers:
                    cluster_name_fields.append("od_%s" % od_name)
                else:
                    logging.error("OutlierDetection %s is not defined (mapping %s)" %
                                  (od_name, mapping_name))

            cluster_name = '%s_cluster' % "_".join(cluster_name_fields)
            cluster_name = re.sub(r'[^0-9A-Za-z_]', '_', cluster_name)

            logging.debug("%s: svc %s -> cluster %s" % (mapping_name, svc, cluster_name))

            if cluster_name not in self.envoy_clusters:
                url = 'tcp://%s' % svc

                if ':' not in svc:
                    url += ':80'

                cluster_def = {
                    "name": cluster_name,
                    "type": "strict_dns",
                    "lb_type": "round_robin",
                    "urls": [ url ]
                }

                if cb_name and (cb_name in breakers):
                    cluster_def['circuit_breakers'] = breakers[cb_name]

                if od_name and (od_name in outliers):
                    cluster_def['outlier_detection'] = outliers[od_name]

                self.envoy_clusters[cluster_name] = cluster_def

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
