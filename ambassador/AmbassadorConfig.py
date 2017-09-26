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

        self.default_liveness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_alive",
            "rewrite": "/server_info"
            # "service" gets added later
        }

        self.liveness_probe = { "enabled": True }
        # self.liveness_probe.update(self.default_liveness_probe)

        self.default_readiness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_ready",
            "rewrite": "/server_info"
            # "service" gets added later
        }

        self.readiness_probe = { "enabled": True }
        # self.readiness_probe.update(self.default_readiness_probe)

        self.tls_config = None

        self.logger = logging.getLogger("ambassador.config")

        self.syntax_errors = 0
        self.object_errors = 0

        for dirpath, dirnames, filenames in os.walk(self.config_dir_path, topdown=True):
            # Modify dirnames in-place (dirs[:]) to remove any weird directories
            # whose names start with '.' -- why? because my GKE cluster mounts my
            # ConfigMap with a self-referential directory named 
            # /etc/ambassador-config/..9989_25_09_15_43_06.922818753, and if we don't
            # ignore that, we end up trying to read the same config files twice, which
            # triggers the collision checks. Sigh.

            dirnames[:] = [ d for d in dirnames if not d.startswith('.') ]

            # self.logger.debug("WALK %s: dirs %s, files %s" % (dirpath, dirnames, filenames))

            for filename in [ x for x in filenames if x.endswith(".yaml") ]:
                self.filename = filename

                filepath = os.path.join(dirpath, filename)

                try:
                    # XXX This is a bit of a hack -- yaml.safe_load_all returns a
                    # generator, and if we don't use list() here, any exception
                    # dealing with the actual object gets deferred 
                    objects = list(yaml.safe_load_all(open(filepath, "r")))
                except Exception as e:
                    self.logger.error("%s: could not parse YAML: %s" % (filepath, e))
                    self.syntax_errors += 1
                    continue

                self.ocount = 0
                for obj in objects:
                    self.ocount += 1

                    try:
                        if not self.process_object(obj):
                            # Object error. Not good but we'll allow the system to start.
                            # We assume that the processor already logged appropriately.
                            self.object_errors += 1
                    except Exception as e:
                        # Bzzzt.
                        self.logger.error("%s[%d]: could not process object: %s" % 
                                          (self.filename, self.ocount, e))
                        self.syntax_errors += 1
                        continue

        if self.syntax_errors:
            # Kaboom.
            raise Exception("ERROR ERROR ERROR Unparseable configuration; exiting")

        if object_errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

        self.generate_envoy_config()

    def process_object(self, obj):
        # OK. What is this thing?
        obj_kind, obj_version = self.validate_object(obj)
        obj_name = obj['name']

        handler_name = "handle_%s" % obj_kind.lower()
        handler = getattr(self, handler_name, None)

        if not handler:
            handler = self.save_object
            self.logger.warning("%s[%d]: no handler for %s, just saving" %
                                (self.filename, self.ocount, obj_kind))
        else:
            self.logger.debug("%s[%d]: handling %s..." %
                              (self.filename, self.ocount, obj_kind))

        return handler(obj, obj_name, obj_kind, obj_version)

    def validate_object(self, obj):
        # Each object must be a dict, and must include "apiVersion"
        # and "type" at toplevel.

        if not isinstance(obj, collections.Mapping):
            raise TypeException("%s[%d]: not a dictionary" %
                                (self.filename, self.ocount))

        if not (("apiVersion" in obj) and ("kind" in obj)):
            raise TypeException("%s[%d]: must have apiVersion and kind" %
                                (self.filename, self.ocount))

        obj_version = obj['apiVersion']
        obj_kind = obj['kind']

        if obj_version.startswith("ambassador/"):
            obj_version = obj_version.split('/')[1]
        else:
            raise ValueException("%s[%d]: apiVersion %s unsupported" %
                                 (self.filename, self.ocount, obj_version))

        schema_key = "%s-%s" % (obj_version, obj_kind)

        schema = self.schemas.get(schema_key, None)

        if not schema:
            schema_path = os.path.join(self.schema_dir_path, obj_version,
                                       "%s.schema" % obj_kind)

            try:
                schema = json.load(open(schema_path, "r"))
            except OSError:
                self.logger.debug("no schema at %s, skipping" % schema_path)
            except json.decoder.JSONDecodeError as e:
                self.logger.warning("corrupt schema at %s, skipping (%s)" %
                                    (schema_path, e))

        if schema:
            self.schemas[schema_key] = schema
            try:
                jsonschema.validate(obj, schema)
            except jsonschema.exceptions.ValidationError as e:
                raise TypeException("%s[%d] is not a valid %s: %s" % 
                                    (self.filename, self.ocount, obj_kind, e))

        return (obj_kind, obj_version)

    def safe_store(self, storage_name, obj_name, obj_kind, value, allow_log=True):
        storage = self.config.setdefault(storage_name, {})

        if obj_name in storage:
            # Oooops.
            raise Exception("%s[%d] defines %s %s, which is already present" % 
                            (self.filename, self.ocount, obj_kind, obj_name))

        if allow_log:
            self.logger.debug("%s[%d]: saving %s %s" %
                          (self.filename, self.ocount, obj_kind, obj_name))

        storage[obj_name] = value
        return True

    def save_object(self, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(obj_kind, obj_name, obj_kind, obj)

    def handle_module(self, obj, obj_name, obj_kind, obj_version):
        return self.safe_store("modules", obj_name, obj_kind, obj['config'])

    def handle_mapping(self, obj, obj_name, obj_kind, obj_version):
        method = obj.get("method", "GET")
        mapping_key = "%s:%s" % (method, obj['prefix'])

        if not self.safe_store("mapping_prefixes", mapping_key, obj_kind, True):
            return False

        return self.safe_store("mappings", obj_name, obj_kind, obj)

    def generate_envoy_config(self):
        # First things first. Assume we'll listen on port 80, with an admin port
        # on 8001.
        self.service_port = 80
        self.admin_port = 8001

        # Next up: let's define initial clusters, routes, and filters.
        #
        # Our initial set of clusters is empty. Note that we use a map for
        # clusters, not a list -- the reason is that multiple mappings can use the
        # same service, and we don't want multiple clusters.
        self.envoy_clusters = {
            # "cluster_ambassador_config": {
            #     "name": "cluster_ambassador_config",
            #     "type": "static",
            #     "urls": [ "tcp://127.0.0.1:8001" ]
            # }
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

        # For mappings, start with empty sets for everything.
        mappings = self.config.get("mappings", {})
        breakers = self.config.get("CircuitBreaker", {})
        outliers = self.config.get("OutlierDetection", {})

        # OK. Given those initial sets, let's look over our global modules.
        modules = self.config.get('modules', {})

        for module_name in modules.keys():
            handler_name = "module_config_%s" % module_name
            handler = getattr(self, handler_name, None)

            if not handler:
                print("module %s: no configuration generator, skipping" % module_name)
                continue

            handler(module_name, modules[module_name])

        # Once modules are handled, we can set up our listeners...
        self.envoy_config['listeners'] = {
            "service_port": self.service_port,
            "admin_port": self.admin_port
        }

        self.default_liveness_probe['service'] = '127.0.0.1:%d' % self.admin_port
        self.default_readiness_probe['service'] = '127.0.0.1:%d' % self.admin_port

        self.logger.debug("LIVENESS: cur  %s" % json.dumps(self.liveness_probe))
        self.logger.debug("LIVENESS: dflt %s" % json.dumps(self.default_liveness_probe))
        self.logger.debug("READINESS: cur  %s" % json.dumps(self.readiness_probe))
        self.logger.debug("READINESS: dflt %s" % json.dumps(self.default_readiness_probe))

        # ...TLS config, if necessary...
        if self.tls_config:
            self.envoy_config['tls'] = self.tls_config

        # ...and probes, if configured.
        for name, cur, dflt in [ ("liveness", self.liveness_probe, self.default_liveness_probe), 
                                 ("readiness", self.readiness_probe, self.default_readiness_probe) ]:
            if cur and cur.get("enabled", False):
                prefix = cur.get("prefix", dflt['prefix'])
                rewrite = cur.get("rewrite", dflt['rewrite'])
                service = cur.get("service", dflt['service'])

                # Push a fake mapping to handle this.
                name = "internal_%s_probe_mapping" % name

                mappings[name] = {
                    'name': name,
                    'prefix': prefix,
                    'rewrite': rewrite,
                    'service': service
                }

                self.logger.debug("PROBE %s: %s -> %s%s" % (name, prefix, service, rewrite))


        # OK! We have all the mappings we need. Process them sorted by decreasing
        # length of prefix.
        for mapping_name in reversed(sorted(mappings.keys(),
                                            key=lambda x: len(mappings[x]['prefix']))):
            mapping = mappings[mapping_name]

            # OK. We need a cluster for this service. Derive it from the 
            # service name, plus things like circuit breaker and outlier 
            # detection settings.
            svc = mapping['service']

            cluster_name_fields =[ svc ]

            cb_name = mapping.get('circuit_breaker', None)

            if cb_name:
                if cb_name in breakers:
                    cluster_name_fields.append("cb_%s" % cb_name)
                else:
                    self.logger.error("CircuitBreaker %s is not defined (mapping %s)" %
                                  (cb_name, mapping_name))

            od_name = mapping.get('outlier_detection', None)

            if od_name:
                if od_name in outliers:
                    cluster_name_fields.append("od_%s" % od_name)
                else:
                    self.logger.error("OutlierDetection %s is not defined (mapping %s)" %
                                  (od_name, mapping_name))

            cluster_name = 'cluster_%s' % "_".join(cluster_name_fields)
            cluster_name = re.sub(r'[^0-9A-Za-z_]', '_', cluster_name)

            self.logger.debug("%s: svc %s -> cluster %s" % (mapping_name, svc, cluster_name))

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

    def module_config_ambassador(self, name, module):
        # Toplevel Ambassador configuration. First up: is TLS configured?

        if 'tls' in module:
            # Yes. Switch to port 443 by default...
            self.service_port = 443

            # ...and save the TLS config.
            self.tls_config = module['tls']

        # After that, is a service port defined?
        if 'service_port' in module:
            # Yes. It overrides the default.
            self.service_port = module['service_port']

        if 'liveness_probe' in module:
            self.liveness_probe.update(module['liveness_probe'])

        if 'readiness_probe' in module:
            self.readiness_probe.update(module['readiness_probe'])

    def module_config_authentication(self, name, module):
        filter = {
            "type": "decoder",
            "name": "extauth",
            "config": {
                "cluster": "cluster_ext_auth",
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

            self.envoy_clusters['cluster_ext_auth'] = {
                "name": "cluster_ext_auth",
                "type": "logical_dns",
                "connect_timeout_ms": 5000,
                "lb_type": "random",
                "urls": [ "tcp://%s" % svc ]
            }

    def pretty(self, obj, out=sys.stdout):
        json.dump(obj, out, indent=4, separators=(',',':'), sort_keys=True)
        out.write("\n")

    def to_json(self, template=None, template_dir=None):
        template_paths = [ self.config_dir_path, self.template_dir_path ]

        if template_dir:
            template_paths.insert(0, template_dir)

        if not template:
            env = Environment(loader=FileSystemLoader(template_paths))
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
