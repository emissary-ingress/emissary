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

class SourcedDict (dict):
    def __init__(self, _source="--internal--", _from=None, **kwargs):
        super().__init__(self, **kwargs)

        if _from and ('_source' in _from):
            self['_source'] = _from['_source']
        else:
            self['_source'] = _source

class AmbassadorConfig (object):
    def __init__(self, config_dir_path, schema_dir_path="schemas", template_dir_path="templates"):
        self.config_dir_path = config_dir_path
        self.schema_dir_path = schema_dir_path
        self.template_dir_path = template_dir_path

        self.schemas = {}
        self.config = {}
        self.envoy_config = {}
        self.envoy_clusters = {}
        self.sources = {}

        self.default_liveness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_alive",
            "rewrite": "/server_info"
            # "service" gets added later
        }

        self.default_readiness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_ready",
            "rewrite": "/server_info"
            # "service" gets added later
        }

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

        if self.object_errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

        self.generate_intermediate_config()

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

        source_key = "%s.%d" % (self.filename, self.ocount)
        self.sources[source_key] = {
            'filename': self.filename,
            'index': self.ocount,
            'yaml': yaml.safe_dump(obj, default_flow_style=False)
        }

        handler(source_key, obj, obj_name, obj_kind, obj_version)

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

    def safe_store(self, source_key, storage_name, obj_name, obj_kind, value, allow_log=True):
        storage = self.config.setdefault(storage_name, {})

        if obj_name in storage:
            # Oooops.
            raise Exception("%s[%d] defines %s %s, which is already present" % 
                            (self.filename, self.ocount, obj_kind, obj_name))

        if allow_log:
            self.logger.debug("%s[%d]: saving %s %s" %
                          (self.filename, self.ocount, obj_kind, obj_name))

        storage[obj_name] = SourcedDict(_source=source_key, **value)
        return storage[obj_name]

    def save_object(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, obj_kind, obj_name, obj_kind, obj)

    def handle_module(self, source_key, obj, obj_name, obj_kind, obj_version):
        return self.safe_store(source_key, "modules", obj_name, obj_kind, obj['config'])

    def handle_mapping(self, source_key, obj, obj_name, obj_kind, obj_version):
        method = obj.get("method", "GET")
        mapping_key = "%s:%s" % (method, obj['prefix'])

        if not self.safe_store(source_key, "mapping_prefixes", mapping_key, obj_kind, obj):
            return False

        return self.safe_store(source_key, "mappings", obj_name, obj_kind, obj)

    def add_intermediate_cluster(self, _source, name, urls, 
                                 type="strict_dns", lb_type="round_robin",
                                 cb_name=None, od_name=None):
        if name not in self.envoy_clusters:
            cluster = SourcedDict(
                _source=_source,
                _referenced_by=[ _source ],
                name=name,
                type=type,
                lb_type=lb_type,
                urls=urls
            )

            if cb_name and (cb_name in self.breakers):
                cluster['circuit_breakers'] = self.breakers[cb_name]

            if od_name and (od_name in self.outliers):
                cluster['outlier_detection'] = self.outliers[od_name]

            self.envoy_clusters[name] = cluster
        else:
            self.envoy_clusters[name]['_referenced_by'].append(_source)

    def add_intermediate_route(self, _source, mapping, cluster_name):
        route = SourcedDict(
            _source=_source,
            prefix=mapping['prefix'],
            prefix_rewrite=mapping.get('rewrite', '/'),
            cluster=cluster_name
        )

        if 'method' in mapping:
            route['method'] = mapping['method']
            route['method_regex'] = route.get('method_regex', False)

        if 'timeout_ms' in mapping:
            route['timeout_ms'] = mapping['timeout_ms']

        self.envoy_config['routes'].append(route)

    def generate_intermediate_config(self):
        # First things first. Assume we'll listen on port 80, with an admin port
        # on 8001.

        self.ambassador_module = SourcedDict(
            service_port=80,
            admin_port = 8001,
            liveness_probe = { "enabled": True },
            readiness_probe = { "enabled": True },
            tls_config = None
        )

        # Next up: let's define initial clusters, routes, and filters.
        #
        # Our initial set of clusters just contains the one for our Courier container.
        # We start with the empty set and use add_intermediate_cluster() to make sure 
        # that all the source-tracking stuff works out.
        #
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.
        self.envoy_clusters = {}
        self.add_intermediate_cluster('--diagnostics--',
                                      'cluster_diagnostics', [ "tcp://127.0.0.1:8877" ],
                                      type="logical_dns", lb_type="random")

        # ...our initial set of routes is empty...
        self.envoy_config['routes'] = []

        # ...and our initial set of filters is just the 'router' filter.
        #
        # !!!! WARNING WARNING WARNING !!!! Filters are actually ORDER-DEPENDENT.
        # We're kind of punting on that so far since we'll only ever add one filter
        # right now.
        self.envoy_config['filters'] = [
            SourcedDict(type="decoder", name="router", config={})
        ]

        # For mappings, start with empty sets for everything.
        mappings = self.config.get("mappings", {})
        self.breakers = self.config.get("CircuitBreaker", {})
        self.outliers = self.config.get("OutlierDetection", {})

        # OK. Given those initial sets, let's look over our global modules.
        modules = self.config.get('modules', {})

        for module_name in modules.keys():
            handler_name = "module_config_%s" % module_name
            handler = getattr(self, handler_name, None)

            if not handler:
                print("module %s: no configuration generator, skipping" % module_name)
                continue

            handler(module_name, modules[module_name])

        # # Once modules are handled, we can set up our listeners...
        self.envoy_config['listeners'] = SourcedDict(
            _from=self.ambassador_module,
            service_port=self.ambassador_module["service_port"],
            admin_port=self.ambassador_module["admin_port"]
        )

        self.default_liveness_probe['service'] = '127.0.0.1:%d' % self.ambassador_module["admin_port"]
        self.default_readiness_probe['service'] = '127.0.0.1:%d' % self.ambassador_module["admin_port"]

        # ...TLS config, if necessary...
        if self.ambassador_module['tls_config']:
            self.envoy_config['tls'] = self.ambassador_module['tls_config']

        # ...and probes, if configured.
        for name, cur, dflt in [ 
            ("liveness", self.ambassador_module['liveness_probe'], self.default_liveness_probe), 
            ("readiness", self.ambassador_module['readiness_probe'], self.default_readiness_probe) ]:

            if cur and cur.get("enabled", False):
                prefix = cur.get("prefix", dflt['prefix'])
                rewrite = cur.get("rewrite", dflt['rewrite'])
                service = cur.get("service", dflt['service'])

                # Push a fake mapping to handle this.
                name = "internal_%s_probe_mapping" % name

                mappings[name] = SourcedDict(
                    _from=self.ambassador_module,
                    name=name,
                    prefix=prefix,
                    rewrite=rewrite,
                    service=service
                )

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
                if cb_name in self.breakers:
                    cluster_name_fields.append("cb_%s" % cb_name)
                else:
                    self.logger.error("CircuitBreaker %s is not defined (mapping %s)" %
                                  (cb_name, mapping_name))

            od_name = mapping.get('outlier_detection', None)

            if od_name:
                if od_name in self.outliers:
                    cluster_name_fields.append("od_%s" % od_name)
                else:
                    self.logger.error("OutlierDetection %s is not defined (mapping %s)" %
                                  (od_name, mapping_name))

            cluster_name = 'cluster_%s' % "_".join(cluster_name_fields)
            cluster_name = re.sub(r'[^0-9A-Za-z_]', '_', cluster_name)

            self.logger.debug("%s: svc %s -> cluster %s" % (mapping_name, svc, cluster_name))

            url = 'tcp://%s' % svc

            if ':' not in svc:
                url += ':80'

            self.add_intermediate_cluster(mapping['_source'], cluster_name, [ url ],
                                          cb_name=cb_name, od_name=od_name)

            self.add_intermediate_route(mapping['_source'], mapping, cluster_name)

            # Also add a diag route.

            source = mapping['_source']
            method = mapping.get("method", "GET").lower()
            prefix = mapping['prefix']

            if prefix.startswith('/'):
                prefix = prefix[1:]

            diag_prefix = '/ambassador/v0/diag/%s/%s' % (method, prefix)
            diag_rewrite = '/ambassador/v0/diag/%s/%s' % (method, source)

            self.add_intermediate_route(
                '--diagnostics--',
                {
                    'prefix': diag_prefix,
                    'rewrite': diag_rewrite,
                    'service': 'cluster_diagnostics'
                },
                'cluster_diagnostics'
            )

        # OK. When all is said and done, map the cluster set back into a list.
        self.envoy_config['clusters'] = [
            self.envoy_clusters[name] for name in sorted(self.envoy_clusters.keys())
        ]

    def _get_intermediate_for(self, element_list, source_keys, value):
        if not isinstance(value, dict):
            return

        value_source = value.get("_source", None)
        value_referenced_by = value.get("_referenced_by", [])

        if ((value_source in source_keys) or
            (source_keys & set(value_referenced_by))):
            element_list.append(value)

    def get_intermediate_for(self, source_key):
        source_keys = []

        if source_key in self.sources:
            source_keys.append(source_key)
        elif not re.search(r'\.\d+$', source_key):
            source_key_with_dot = source_key + "."

            source_keys = [ key for key in self.sources.keys()
                            if key.startswith(source_key_with_dot) ]

        source_keys = set(source_keys)

        result = {
            "sources": [ self.sources[key] for key in source_keys ],
        }

        for key in self.envoy_config.keys():
            result[key] = []

            value = self.envoy_config[key]

            if isinstance(value, list):
                for v2 in value:
                    self._get_intermediate_for(result[key], source_keys, v2)
            else:
                self._get_intermediate_for(result[key], source_keys, value)

        return result

    def generate_envoy_config(self, template=None, template_dir=None):
        # Finally! Render the template to JSON...
        envoy_json = self.to_json(template=template, template_dir=template_dir)
        rc = RichStatus.fromError("impossible")

        # ...and use the JSON parser as a final sanity check.
        try:
            obj = json.loads(envoy_json)
            rc = RichStatus.OK(msg="Envoy configuration OK", envoy_config=obj)
        except json.decoder.JSONDecodeError as e:
            rc = RichStatus.fromError("Invalid Envoy configuration: %s" % str(e),
                                      raw=envoy_json, exception=e)

        return rc

    def set_config_ambassador(self, module, key, value, merge=False):
        if not merge:
            self.ambassador_module[key] = value
        else:
            self.ambassador_module[key].update(value)

        self.ambassador_module['_source'] = module['_source']

    def update_config_ambassador(self, module, key, value):
        self.set_config_ambassador(module, key, value, merge=True)

    def module_config_ambassador(self, name, module):
        # Toplevel Ambassador configuration. First up: is TLS configured?

        if 'tls' in module:            
            # Yes. Switch to port 443 by default...
            self.set_config_ambassador(module, 'service_port', 443)

            # ...save the TLS config...
            self.set_config_ambassador(module, 'tls_config', module['tls'])

        # After that, is a service port defined?
        if 'service_port' in module:
            # Yes. It overrides the default.
            self.set_config_ambassador(module, 'service_port', module['service_port'])

        # How about an admin port?
        if 'admin_port' in module:
            # Yes. It overrides the default.
            self.set_config_ambassador(module, 'admin_port', module['admin_port'])

        if 'liveness_probe' in module:
            self.update_config_ambassador(module, 'liveness_probe', module['liveness_probe'])

        if 'readiness_probe' in module:
            self.update_config_ambassador(module, 'readiness_probe', module['readiness_probe'])

    def module_config_authentication(self, name, module):
        filter = SourcedDict(
            _from=module,
            type="decoder",
            name="extauth",
            config={
                "cluster": "cluster_ext_auth",
                "timeout_ms": 5000
            }
        )

        path_prefix = module.get("path_prefix", None)

        if path_prefix:
            filter['config']['path_prefix'] = path_prefix

        allowed_headers = module.get("allowed_headers", None)

        if allowed_headers:
            filter['config']['allowed_headers'] = allowed_headers

        self.envoy_config['filters'].insert(0, filter)

        if 'ext_auth_cluster' not in self.envoy_clusters:
            svc = module.get('auth_service', '127.0.0.1:5000')

            self.add_intermediate_cluster(module['_source'],
                                          'cluster_ext_auth', [ "tcp://%s" % svc ],
                                          type="logical_dns", lb_type="random")

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

    def dump(self):
        print("==== config")
        self.pretty(self.config)

        print("==== envoy_config")
        self.pretty(self.envoy_config)
