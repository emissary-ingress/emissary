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

        self.logger = logging.getLogger("ambassador.config")

        self.schemas = {}
        self.config = {}
        self.envoy_config = {}
        self.envoy_clusters = {}

        self.sources = {
            "--internal--": {
                "kind": "Internal",
                "version": "v0",
                "name": "Ambassador Internals",
                "filename": "--internal--",
                "index": 0,
                "description": "The '--internal--' source marks objects created by Ambassador's internal logic."
            },
            "--diagnostics--": {
                "kind": "diagnostics",
                "version": "v0",
                "name": "Ambassador Diagnostics",
                "filename": "--diagnostics--",
                "index": 0,
                "description": "The '--diagnostics--' source marks objects created by Ambassador to assist with diagnostic output."
            }            
        }

        self.source_map = {
            '--internal--': { '--internal--': True }
        }

        self.default_liveness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_alive",
            "rewrite": "/ambassador/v0/check_alive",
            # "service" gets added later
        }

        self.default_readiness_probe = {
            "enabled": True,
            "prefix": "/ambassador/v0/check_ready",
            "rewrite": "/ambassador/v0/check_ready",
            # "service" gets added later
        }

        self.default_tls_config = {
            "server": {
                "cert_chain_file": "/etc/certs/tls.crt",
                "private_key_file": "/etc/certs/tls.key"
            },
            "client": {
                "cacert_chain_file": "/etc/cacert/fullchain.pem"
            }
        }

        self.tls_config = None

        self.errors = {}
        self.fatal_errors = 0
        self.object_errors = 0

        if not os.path.isdir(self.config_dir_path):
            raise Exception("ERROR ERROR ERROR configuration directory %s does not exist; exiting" % self.config_dir_path)

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
                    self.fatal_errors += 1
                    continue

                self.ocount = 0
                for obj in objects:
                    self.ocount += 1

                    rc = self.process_object(obj)

                    if not rc:
                        # Object error. Not good but we'll allow the system to start.
                        self.post_error(rc)

        if self.fatal_errors:
            # Kaboom.
            raise Exception("ERROR ERROR ERROR Unparseable configuration; exiting")

        if self.errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

        self.generate_intermediate_config()

    def current_source_key(self):
        return("%s.%d" % (self.filename, self.ocount))

    def post_error(self, rc):
        source_map = self.source_map.setdefault(self.filename, {})
        source_map[self.current_source_key()] = True

        errors = self.errors.setdefault(self.current_source_key(), [])
        errors.append(rc.toDict())
        self.logger.error("%s: %s" % (self.current_source_key(), rc))

    def process_object(self, obj):
        obj_version = obj['apiVersion']
        obj_kind = obj['kind']
        obj_name = obj['name']

        # ...save the source info...
        source_key = "%s.%d" % (self.filename, self.ocount)
        self.sources[source_key] = {
            'kind': obj_kind,
            'version': obj_version,
            'name': obj_name,
            'filename': self.filename,
            'index': self.ocount,
            'yaml': yaml.safe_dump(obj, default_flow_style=False)
        }

        source_map = self.source_map.setdefault(self.filename, {})
        source_map[source_key] = True

        # OK. What is this thing?
        rc = self.validate_object(obj)

        if not rc:
            # Well that's no good.
            return rc

        # OK, so far so good. Grab the handler for this object type.
        handler_name = "handle_%s" % obj_kind.lower()
        handler = getattr(self, handler_name, None)

        if not handler:
            handler = self.save_object
            self.logger.warning("%s[%d]: no handler for %s, just saving" %
                                (self.filename, self.ocount, obj_kind))
        else:
            self.logger.debug("%s[%d]: handling %s..." %
                              (self.filename, self.ocount, obj_kind))

        try:
            handler(source_key, obj, obj_name, obj_kind, obj_version)
        except Exception as e:
            # Bzzzt.
            return RichStatus.fromError("could not process %s object: %s" % (obj_kind, e))

        # OK, all's well.
        return RichStatus.OK(msg="%s object processed successfully" % obj_kind)

    def validate_object(self, obj):
        # Each object must be a dict, and must include "apiVersion"
        # and "type" at toplevel.

        if not isinstance(obj, collections.Mapping):
            return RichStatus.fromError("not a dictionary")

        if not (("apiVersion" in obj) and ("kind" in obj) and ("name" in obj)):
            return RichStatus.fromError("must have apiVersion, kind, and name")

        obj_version = obj['apiVersion']
        obj_kind = obj['kind']
        obj_name = obj['name']

        if obj_version.startswith("ambassador/"):
            obj_version = obj_version.split('/')[1]
        else:
            return RichStatus.fromError("apiVersion %s unsupported" % obj_version)

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
                return RichStatus.fromError("not a valid %s: %s" % (obj_kind, e))

        return RichStatus.OK(msg="valid %s" % obj_kind,
                             details=(obj_kind, obj_version, obj_name))

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
        mapping_key = "%s:%s->%s" % (method, obj['prefix'], obj['service'])

        if not self.safe_store(source_key, "mapping_prefixes", mapping_key, obj_kind, obj):
            return False

        return self.safe_store(source_key, "mappings", obj_name, obj_kind, obj)

    def diag_port(self):
        modules = self.config.get("modules", {})
        amod = modules.get("ambassador", {})

        return amod.get("diag_port", 8877)

    def diag_service(self):
        return "127.0.0.1:%d" % self.diag_port()

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
                self.breakers[cb_name]['_referenced_by'].append(_source)

            if od_name and (od_name in self.outliers):
                cluster['outlier_detection'] = self.outliers[od_name]
                self.outliers[od_name]['_referenced_by'].append(_source)

            self.envoy_clusters[name] = cluster
        else:
            self.envoy_clusters[name]['_referenced_by'].append(_source)

    def add_intermediate_route(self, _source, mapping, cluster_name):
        routes = self.envoy_config['routes']
        group = None
        for r in routes:
            if r['prefix'] == mapping['prefix'] and r.get('method') == mapping.get('method'):
                group = r
                break

        if group is None:
            route = SourcedDict(
                _source=_source,
                prefix=mapping['prefix'],
                prefix_rewrite=mapping.get('rewrite', '/'),
                clusters=[{"name": cluster_name,
                           "weight": mapping.get("weight", None)}]
            )

            if 'method' in mapping:
                route['method'] = mapping['method']
                route['method_regex'] = route.get('method_regex', False)

            if 'timeout_ms' in mapping:
                route['timeout_ms'] = mapping['timeout_ms']

            routes.append(route)
        else:
            group["clusters"].append({"name": cluster_name,
                                      "weight": mapping.get("weight", None)})

    def generate_intermediate_config(self):
        # First things first. Define the default "Ambassador" module...

        self.ambassador_module = SourcedDict(
            service_port = 80,
            admin_port = 8001,
            diag_port = 8877,
            liveness_probe = { "enabled": True },
            readiness_probe = { "enabled": True },
            tls_config = None
        )

        # ...pull our defined modules from our config...
        modules = self.config.get('modules', {})

        # ...and then use process whatever the user has to say in the "ambassador" module.
        if 'ambassador' in modules:
            self.module_config_ambassador("ambassador", modules['ambassador'])        

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
        # self.add_intermediate_cluster('--diagnostics--',
        #                               'cluster_diagnostics', 
        #                               [ "tcp://%s" % self.diag_service() ],
        #                               type="logical_dns", lb_type="random")

        # Our initial set of routes is empty...
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

        for key, breaker in self.breakers.items():
            breaker['_referenced_by'] = []

        self.outliers = self.config.get("OutlierDetection", {})

        for key, outlier in self.outliers.items():
            outlier['_referenced_by'] = []

        # OK. Given those initial sets, let's look over our global modules.
        for module_name in modules.keys():
            if module_name == 'ambassador':
                continue

            handler_name = "module_config_%s" % module_name
            handler = getattr(self, handler_name, None)

            if not handler:
                self.logger.error("module %s: no configuration generator, skipping" % module_name)
                continue

            handler(module_name, modules[module_name])

        # # Once modules are handled, we can set up our listeners...
        self.envoy_config['listeners'] = SourcedDict(
            _from=self.ambassador_module,
            service_port=self.ambassador_module["service_port"],
            admin_port=self.ambassador_module["admin_port"]
        )

        self.default_liveness_probe['service'] = self.diag_service()
        self.default_readiness_probe['service'] = self.diag_service()

        # ...TLS config, if necessary...
        if self.ambassador_module['tls_config']:
            self.logger.debug("USING TLS")
            self.envoy_config['tls'] = self.ambassador_module['tls_config']

        # ...and probes, if configured.
        for name, cur, dflt in [ 
            ("liveness", self.ambassador_module['liveness_probe'], 
                         self.default_liveness_probe), 
            ("readiness", self.ambassador_module['readiness_probe'], 
                         self.default_readiness_probe) ]:

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

        # OK! We have all the mappings we need. Process them (don't worry about sorting
        # yet, we'll do that on routes).

        for mapping_name in mappings.keys():
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

            # # Also add a diag route.

            # source = mapping['_source']

            # method = mapping.get("method", "GET")
            # dmethod = method.lower()

            # prefix = mapping['prefix']
            # dprefix = prefix[1:] if prefix.startswith('/') else prefix

            # diag_prefix = '/ambassador/v0/diag/%s/%s' % (dmethod, dprefix)
            # diag_rewrite = '/ambassador/v0/diag/%s?method=%s&resource=%s' % (source, method, prefix)

            # self.add_intermediate_route(
            #     '--diagnostics--',
            #     {
            #         'prefix': diag_prefix,
            #         'rewrite': diag_rewrite,
            #         'service': 'cluster_diagnostics'
            #     },
            #     'cluster_diagnostics'
            # )

        # # Also push a fallback diag route, so that one can easily ask for diagnostics 
        # # by source file.

        # self.add_intermediate_route(
        #     '--diagnostics--',
        #     {
        #         'prefix': "/ambassador/v0/diag/",
        #         'rewrite': "/ambassador/v0/diag/",
        #         'service': 'cluster_diagnostics'
        #     },
        #     'cluster_diagnostics'
        # )

        # We need to default any unspecified weights and renormalize to 100
        for r in self.envoy_config['routes']:
            clusters = r["clusters"]
            total = 0.0
            unspecified = 0
            for c in clusters:
                if c["weight"] is None:
                    unspecified += 1
                else:
                    total += c["weight"]
            if unspecified:
                for c in clusters:
                    if c["weight"] is None:
                        c["weight"] = (100.0 - total)/unspecified
            elif total != 100.0:
                for c in clusters:
                    c["weight"] *= 100.0/total

        # OK. When all is said and done, sort the list of routes by descending 
        # legnth of prefix, then prefix itself, then method...
        self.envoy_config['routes'].sort(reverse=True,
                                         key=lambda x: (len(x['prefix']), 
                                                        x['prefix'],
                                                        x.get('method', 'GET')))

        # ...map clusters back into a list...
        self.envoy_config['clusters'] = [
            self.envoy_clusters[name] for name in sorted(self.envoy_clusters.keys())
        ]

        # ...and then repeat for breakers and outliers, but copy them in the process so we
        # can mess with the originals.
        #
        # What's going on here is that circuit-breaker and outlier-detection configs aren't
        # included as independent objects in envoy.json, but we want to be able to discuss 
        # them in diag. We also don't need to keep the _source and _referenced_by elements
        # in their real Envoy appearances.

        self.envoy_config['breakers'] = self.clean_and_copy(self.breakers)
        self.envoy_config['outliers'] = self.clean_and_copy(self.outliers)

    def clean_and_copy(self, d):
        out = []

        for key in sorted(d.keys()):
            original = d[key]
            copy = dict(**original)

            if '_source' in original:
                del(original['_source'])

            if '_referenced_by' in original:
                del(original['_referenced_by'])

            out.append(copy)

        return out

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

        if source_key in self.source_map:
            # Exact match for a file in the source map: include all the objects
            # in the file.
            source_keys = self.source_map[source_key]
        elif source_key in self.sources:
            # Exact match for an object in a file: include only that object.
            source_keys.append(source_key)
        else: 
            # No match at all. Weird.
            return {
                "error": "No source matches %s" % source_key
            }

        source_keys = set(source_keys)

        sources = []

        for key in source_keys:
            source_dict = dict(self.sources[key])
            source_dict['errors'] = [
                {
                    'summary': error['error'].split('\n', 1)[0],
                    'text': error['error']
                }
                for error in self.errors.get(key, [])
            ]

            sources.append(source_dict)

        result = {
            "sources": sources
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
        return RichStatus.OK(msg="Envoy configuration OK", envoy_config=envoy_json)
#        rc = RichStatus.fromError("impossible")

        # ...and use the JSON parser as a final sanity check.
#        try:
#            obj = json.loads(envoy_json)
#            rc = RichStatus.OK(msg="Envoy configuration OK", envoy_config=obj)
#        except json.decoder.JSONDecodeError as e:
#            rc = RichStatus.fromError("Invalid Envoy configuration: %s" % str(e),
#                                      raw=envoy_json, exception=e)

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
            tmod = module['tls']
            tmp_config = {}
            some_enabled = False

            if ('server' in tmod) and tmod['server'].get('enabled', True):
                # Server-side TLS is enabled. 
                self.logger.debug("TLS termination enabled!")
                some_enabled = True

                # Yes. Switch to port 443 by default...
                self.set_config_ambassador(module, 'service_port', 443)

                # ...and merge in the server-side defaults.
                tmp_config.update(self.default_tls_config['server'])
                tmp_config.update(tmod['server'])

            if ('client' in tmod) and tmod['client'].get('enabled', True):
                # Client-side TLS is enabled. 
                self.logger.debug("TLS client certs enabled!")
                some_enabled = True

                # Merge in the client-side defaults.
                tmp_config.update(self.default_tls_config['client'])
                tmp_config.update(tmod['client'])

            if some_enabled:
                if 'enabled' in tmp_config:
                    del(tmp_config['enabled'])

                # Save the TLS config...
                self.set_config_ambassador(module, 'tls_config', tmp_config)

            self.logger.debug("TLS config: %s" % json.dumps(self.ambassador_module['tls_config'], indent=4))

        # After that, check for port definitions and probes, and copy them in as we find them.
        for key in [ 'service_port', 'admin_port', 'diag_port',
                     'liveness_probe', 'readiness_probe' ]:
            if key in module:
                # Yes. It overrides the default.
                self.set_config_ambassador(module, key, module[key])

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

            if ':' not in svc:
                svc = "%s:80" % svc

            self.add_intermediate_cluster(module['_source'],
                                          'cluster_ext_auth', [ "tcp://%s" % svc ],
                                          type="logical_dns", lb_type="random")

    def pretty(self, obj, out=sys.stdout):
        out.write(obj)
#        json.dump(obj, out, indent=4, separators=(',',':'), sort_keys=True)
#        out.write("\n")

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
