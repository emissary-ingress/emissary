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

import sys

from typing import Any, ClassVar, Dict, Iterable, List, Optional, Tuple, Union
from typing import cast as typecast

#import collections
#import datetime
import json
import logging
import os
import re

import jsonschema

from pkg_resources import Requirement, resource_filename

from jinja2 import Environment, FileSystemLoader

from .utils import RichStatus, SourcedDict, read_cert_secret, save_cert, TLSPaths, kube_v1, check_cert_file
from .resource import Resource
from .mapping import Mapping

#from .VERSION import Version

#############################################################################
## config.py -- the main configuration parser for Ambassador
##
## Ambassador configures itself by creating a new Config object, which calls
## Config.__init__().
##
## __init__() sets up all the defaults for everything, then walks over all the
## YAML it can find and calls self.load_yaml() to load each YAML file. After
## everything is loaded, it calls self.process_all_objects() to build the
## config objects.
##
## load_yaml() does the heavy lifting around YAML parsing and such, including
## managing K8s annotations if so requested. Every object in every YAML file is
## parsed and saved before any object is processed.
##
## process_all_objects() walks all the saved objects and creates an internal
## representation of the Ambassador config in the data structures initialized
## by __init__(). Each object is processed with self.process_object(). This
## internal representation is called the intermediate config.
##
## process_object() handles a single parsed object from YAML. It uses
## self.validate_object() to make sure of a schema match; assuming that's
## good, most of the heavy lifting is done by a handler method. The handler
## method for a given type is named handle_kind(), with kind in lowercase,
## so e.g. the Mapping object is processed using the handle_mapping() method.
##
## After all of that, the actual Envoy config is generated from the intermediate
## config using generate_envoy_config().
##
## The diag service also uses generate_intermediate_for() to extract the
## intermediate config for a given mapping or service.

# Custom types
# ServiceInfo is a tuple of information about a service: 
# service name, service URL, originate TLS?, TLS context name
ServiceInfo = Tuple[str, str, bool, str]

# StringOrList is either a string or a list of strings.
StringOrList = Union[str, List[str]]

class Config:
    # When using multiple Ambassadors in one cluster, use AMBASSADOR_ID to distinguish them.
    ambassador_id: ClassVar[str] = os.environ.get('AMBASSADOR_ID', 'default')
    runtime: ClassVar[str] = "kubernetes" if os.environ.get('KUBERNETES_SERVICE_HOST', None) else "docker"
    namespace: ClassVar[str] = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

    def __init__(self, schema_dir_path: Optional[str]=None) -> None:
        if not schema_dir_path:
            # Note that this "resource_filename" has to do with setuptool packages, not
            # with our Resource class.
            schema_dir_path = resource_filename(Requirement.parse("ambassador"),"schemas")

        self.schema_dir_path = schema_dir_path

        self.logger = logging.getLogger("ambassador.config")

        # self.logger.debug("Scout version %s" % Config.scout_version)
        self.logger.debug("Runtime       %s" % Config.runtime)
        self.logger.debug("SCHEMA DIR    %s" % os.path.abspath(self.schema_dir_path))

        self._reset()

    def _reset(self) -> None:
        """
        Resets this Config to the empty, default state so it can load a new config.
        """

        self.logger.debug("RESET")

        self.current_resource: Optional[Resource] = None

        # XXX flat wrong
        self.schemas: Dict[str, dict] = {}
        self.config: Dict[str, Dict[str, Resource]] = {}
        self.tls_contexts: Dict[str, SourcedDict] = {}

        # XXX flat wrong
        self.envoy_config: Dict[str, SourcedDict] = {}
        self.envoy_clusters: Dict[str, SourcedDict] = {}
        self.envoy_routes: Dict[str, SourcedDict] = {}

        # res_key => Resource
        self.sources: Dict[str, Resource] = {}

        # Allow overriding the location of a resource with a Pragma
        self.location_overrides: Dict[str, Dict[str, str]] = {}

        # Save our magic internal sources.
        self.save_source(Resource.internal_resource())
        self.save_source(Resource.diagnostics_resource())

        # Set up the default probes and such.
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

        self.default_diagnostics = {
            "enabled": True,
            "prefix": "/ambassador/v0/",
            "rewrite": "/ambassador/v0/",
            # "service" gets added later
        }

        # 'server' and 'client' are special contexts. Others
        # use cert_chain_file defaulting to context.crt,
        # private_key_file (context.key), and cacert_chain_file
        # (context.pem).

        self.default_tls_config: Dict[str, Dict[str, str]] = {
            "server": {},
            "client": {},
        }
        if os.path.isfile(TLSPaths.mount_tls_crt.value):
            self.default_tls_config["server"]["cert_chain_file"] = TLSPaths.mount_tls_crt.value
        if os.path.isfile(TLSPaths.mount_tls_key.value):
            self.default_tls_config["server"]["private_key_file"] = TLSPaths.mount_tls_key.value
        if os.path.isfile(TLSPaths.client_mount_crt.value):
            self.default_tls_config["client"]["cacert_chain_file"] = TLSPaths.client_mount_crt.value

        self.tls_config = None

        self.errors: Dict[str, List[str]] = {}
        self.fatal_errors = 0
        self.object_errors = 0

    def save_source(self, resource: Resource) -> None:
        """
        Save a give Resource as a source of Ambassador config information.
        """
        self.sources[resource.res_key] = resource

    def load_all(self, resources: Iterable[Resource]) -> None:
        """
        Loads all of a set of Resources. It is the caller's responsibility to arrange for 
        the set of Resources to be sorted in some way that makes sense.
        """
        for resource in resources:
            # XXX I think this whole override thing should go away.
            #
            # Any override here?
            if resource.res_key in self.location_overrides:
                # Let Pragma objects override source information for this filename.
                override = self.location_overrides[resource.res_key]
                resource.location = override.get('source', resource.res_key)

            # Is an ambassador_id present in this object?
            allowed_ids: StringOrList = resource.attrs.get('ambassador_id', 'default')

            if allowed_ids:
                # Make sure it's a list. Yes, this is Draconian,
                # but the jsonschema will allow only a string or a list,
                # and guess what? Strings are iterables.
                if type(allowed_ids) != list:
                    allowed_ids = typecast(StringOrList, [ allowed_ids ])

                if Config.ambassador_id not in allowed_ids:
                    self.logger.debug("LOAD_ALL: skip %s; id %s not in %s" %
                                        (resource, Config.ambassador_id, allowed_ids))
                    return

            self.logger.debug("LOAD_ALL: %s @ %s" % (resource, resource.location))

            rc = self.process(resource)

            if not rc:
                # Object error. Not good but we'll allow the system to start.
                self.post_error(rc, resource=resource)

        if self.fatal_errors:
            # Kaboom.
            raise Exception("ERROR ERROR ERROR Unparseable configuration; exiting")

        if self.errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

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

    def post_error(self, rc: RichStatus, resource=None):
        if not resource:
            resource = self.current_resource

        if not resource:
            raise Exception("FATAL: trying to post an error from a totally unknown resource??")

        self.save_source(resource)
        resource.post_error(rc.toDict())

        # XXX Probably don't need this data structure, since we can walk the source
        # list and get them all.
        errors = self.errors.setdefault(resource.res_key, [])
        errors.append(rc.toDict())
        self.logger.error("%s: %s" % (resource, rc))

    def process(self, resource: Resource) -> RichStatus:
        # This should be impossible.
        if not resource or not resource.attrs:
            return RichStatus.fromError("undefined object???")

        self.current_res_key = resource.res_key

        if not resource.version:
            return RichStatus.fromError("need apiVersion")

        if not resource.kind:
            return RichStatus.fromError("need kind")

        # Is this a pragma object?
        if resource.kind == 'Pragma':
            # Yes. Handle this inline and be done.
            return self.handle_pragma(resource)

        # Not a pragma. It needs a name...
        if 'name' not in resource.attrs:
            return RichStatus.fromError("need name")

        # ...and off we go. Save the source info...
        self.save_source(resource)

        # ...and figure out if this thing is OK.
        rc = self.validate_object(resource)

        if not rc:
            # Well that's no good.
            return rc

        # OK, so far so good. Grab the handler for this object type.
        handler_name = "handle_%s" % resource.kind.lower()
        handler = getattr(self, handler_name, None)

        if not handler:
            handler = self.save_object
            self.logger.warning("%s: no handler for %s, just saving" % (resource, resource.kind))
        else:
            self.logger.debug("%s: handling %s..." % (resource, resource.kind))

        try:
            handler(resource)
        except Exception as e:
            # Bzzzt.
            return RichStatus.fromError("%s: could not process %s object: %s" % (resource, resource.kind, e))

        # OK, all's well.
        return RichStatus.OK(msg="%s object processed successfully" % resource.kind)

    def validate_object(self, resource: Resource) -> RichStatus:
        obj = resource.attrs

        # This is basically "impossible"
        if not (("apiVersion" in obj) and ("kind" in obj) and ("name" in obj)):
            return RichStatus.fromError("must have apiVersion, kind, and name")

        version = resource.version

        # Ditch the leading ambassador/ that really needs to be there.
        if version.startswith("ambassador/"):
            version = version.split('/')[1]
        else:
            return RichStatus.fromError("apiVersion %s unsupported" % version)

        # Do we already have this schema loaded?
        schema_key = "%s-%s" % (version, resource.kind)
        schema = self.schemas.get(schema_key, None)

        if not schema:
            # Not loaded. Go find it on disk.
            schema_path = os.path.join(self.schema_dir_path, version,
                                       "%s.schema" % resource.kind)

            try:
                # Load it up...
                schema = json.load(open(schema_path, "r"))

                # ...and then cache it, if it exists. Note that we'll never
                # get here if we find something that doesn't parse.
                if schema:
                    self.schemas[schema_key] = typecast(Dict[Any, Any], schema)
            except OSError:
                self.logger.debug("no schema at %s, skipping" % schema_path)
            except json.decoder.JSONDecodeError as e:
                self.logger.warning("corrupt schema at %s, skipping (%s)" %
                                    (schema_path, e))

        if schema:
            # We have a schema. Does the object validate OK?
            try:
                jsonschema.validate(obj, schema)
            except jsonschema.exceptions.ValidationError as e:
                # Nope. Bzzzzt.
                return RichStatus.fromError("not a valid %s: %s" % (resource.kind, e))

        # All good. Return an OK.
        return RichStatus.OK(msg="valid %s" % resource.kind)

    def safe_store(self, storage_name: str, resource: Resource, allow_log: bool=True) -> None:
        """
        Safely store a Resource under a given storage name. The storage_name is separate
        because we may need to e.g. store a Module under the 'ratelimit' name or the like.
        Within a storage_name bucket, the Resource will be stored under its name.

        :param storage_name: where shall we file this?
        :param resource: what shall we file?
        :param allow_log: if True, logs that we're saving this thing.
        """

        storage = self.config.setdefault(storage_name, {})

        if resource.name in storage:
            # Oooops.
            raise Exception("%s defines %s %s, which is already present" %
                            (resource, resource.kind, resource.name))

        if allow_log:
            self.logger.debug("%s: saving %s %s" %
                          (resource, resource.kind, resource.name))

        storage[resource.name] = resource

    def save_object(self, resource: Resource, allow_log: bool=False) -> None:
        """
        Saves a Resource using its kind as the storage class name. Sort of the
        defaulted version of safe_store.

        :param resource: what shall we file?
        :param allow_log: if True, logs that we're saving this thing.
        """

        self.safe_store(resource.kind, resource, allow_log=allow_log)

    def get_module(self, module_name: str) -> Optional[Resource]:
        """
        Fetch a module from the module store. Can return None if no
        such module exists.

        :param module_name: name of the module you want.
        """

        modules = self.config.get("modules", None)

        if modules:
            return modules.get(module_name, None)
        else:
            return None

    def module_lookup(self, module_name: str, key: str, default: Any=None) -> Any:
        """
        Look up a specific key in a given module. If the named module doesn't 
        exist, or if the key doesn't exist in the module, return the default.

        :param module_name: name of the module you want.
        :param key: key to look up within the module
        :param default: default value if the module is missing or has no such key
        """

        module = self.get_module(module_name)

        if module:
            return module.get(key, default)

        return default

    # XXX Misnamed. handle_pragma isn't the same signature as, say, handle_mapping.
    # XXX Is this needed any more??
    def handle_pragma(self, resource: Resource) -> RichStatus:
        """
        Handles a Pragma object. May not be needed any more...
        """

        attrs = resource.attrs
        res_key = resource.res_key

        keylist = sorted([x for x in sorted(attrs.keys()) if ((x != 'apiVersion') and (x != 'kind'))])

        self.logger.debug("PRAGMA: %s" % keylist)

        for key in keylist:
            if key == 'source':
                override = self.location_overrides.setdefault(res_key, {})
                override['source'] = attrs['source']

                self.logger.debug("PRAGMA: override %s to %s" %
                                  (res_key, self.location_overrides[res_key]['source']))

        return RichStatus.OK(msg="handled pragma object")

    def handle_module(self, resource: Resource) -> None:
        """
        Handles a Module resource.
        """

        # Make a new Resource from the 'config' element of this Resource
        # Note that we leave the original serialization intact, since it will
        # indeed show a human the YAML that defined this module.
        module_resource = Resource.from_resource(resource, attrs=resource.attrs['config'])

        self.safe_store("modules", module_resource)

    def handle_ratelimitservice(self, resource: Resource) -> None:
        """
        Handles a RateLimitService resource.
        """

        self.safe_store("ratelimit_configs", resource)

    def handle_authservice(self, resource: Resource) -> None:
        """
        Handles an AuthService resource.
        """

        self.safe_store("auth_configs", resource)

    def handle_mapping(self, resource: Resource) -> None:
        """
        Handles a Mapping resource.

        Mappings are complex things, so a lot of stuff gets buried in a Mapping 
        object.
        """

        mapping = Mapping(resource.res_key, **resource.attrs)
        self.safe_store("mappings", mapping)

    def diag_port(self):
        """
        Returns the diagnostics port for this Ambassador config.
        """

        return self.module_lookup("ambassador", "diag_port", 8877)

    def diag_service(self):
        """
        Returns the diagnostics service URL for this Ambassador config.
        """

        return "127.0.0.1:%d" % self.diag_port()

    def add_intermediate_cluster(self, _source: str, name: str, _service: str, urls: List[str],
                                 type: str="strict_dns", lb_type: str="round_robin",
                                 cb_name: Optional[str]=None, od_name: Optional[str]=None,
                                 originate_tls: Union[str, bool]=None,
                                 grpc: Optional[bool]=False, host_rewrite: Optional[str]=None):
        """
        Adds a cluster to the IR. This is wicked ugly because clusters are complex,
        and happen to live in an annoying bit of the IR.
        """

        if name not in self.envoy_clusters:
            self.logger.debug("CLUSTER %s: new from %s" % (name, _source))

            cluster = SourcedDict(
                _source=_source,
                _referenced_by=[ _source ],
                _service=_service,
                name=name,
                type=type,
                lb_type=lb_type,
                urls=urls
            )

            if cb_name and (cb_name in self.breakers):
                cluster['circuit_breakers'] = self.breakers[cb_name]
                self.breakers[cb_name]._mark_referenced_by(_source)

            if od_name and (od_name in self.outliers):
                cluster['outlier_detection'] = self.outliers[od_name]
                self.outliers[od_name]._mark_referenced_by(_source)

            if originate_tls == True:
                cluster['tls_context'] = { '_ambassador_enabled': True }
                cluster['tls_array'] = []
            elif (originate_tls and (originate_tls in self.tls_contexts)):
                cluster['tls_context'] = self.tls_contexts[typecast(str, originate_tls)]
                self.tls_contexts[typecast(str, originate_tls)]._mark_referenced_by(_source)

                tls_array: List[Dict[str, str]] = []

                for key, value in cluster['tls_context'].items():
                    if key.startswith('_'):
                        continue

                    tls_array.append({ 'key': key, 'value': value })
                    cluster['tls_array'] = sorted(tls_array, key=lambda x: x['key'])

            if host_rewrite and originate_tls:
                cluster['tls_array'].append({'key': 'sni', 'value': host_rewrite })

            if grpc:
                cluster['features'] = 'http2'

            self.envoy_clusters[name] = cluster
        else:
            self.logger.debug("CLUSTER %s: referenced by %s" % (name, _source))

            self.envoy_clusters[name]._mark_referenced_by(_source)

    # XXX This is a silly API. We should have a Cluster object that can carry what kind
    #     of cluster it is (this is a target cluster of weight 50%, this is a shadow cluster,
    #     whatever) and the API should be "add this cluster to this Mapping".
    def add_intermediate_route(self, _source: str, mapping: Mapping, svc: str, cluster_name: str, 
                               shadow: bool=False) -> None:
        """
        Adds a route to the IR. This is wicked ugly because routes are complex,
        and happen to live in an annoying bit of the IR.
        """

        route = self.envoy_routes.get(mapping.group_id, None)
        host_redirect = mapping.get('host_redirect', False)
        shadow = mapping.get('shadow', False)

        if route:
            # Is this a host_redirect? If so, that's an error.
            if host_redirect:
                self.logger.error("ignoring non-unique host_redirect mapping %s (see also %s)" %
                                  (mapping['name'], route['_source']))

            # Is this a shadow? If so, is there already a shadow marked?
            elif shadow:
                extant_shadow = route.get('shadow', None)

                if extant_shadow:
                    shadow_name = extant_shadow.get('name', None)

                    if shadow_name != cluster_name:
                        self.logger.error("mapping %s defines multiple shadows! Ignoring %s" %
                                        (mapping['name'], cluster_name))
                else:
                    # XXX CODE DUPLICATION with mapping.py!!
                    # We're going to need to support shadow weighting later, so use a dict here.
                    route['shadow'] = {
                        'name': cluster_name
                    }
                    route.setdefault('clusters', [])
            else:
                # Take the easy way out -- just add a new entry to this
                # route's set of weighted clusters.
                route["clusters"].append( { "name": cluster_name,
                                            "weight": mapping.attrs.get("weight", None) } )

            route._mark_referenced_by(_source)

            return

        # OK, if here, we don't have an extent route group for this Mapping. Make a
        # new one.
        route = mapping.new_route(svc, cluster_name)
        self.envoy_routes[mapping.group_id] = route

    def service_tls_check(self, svc: str, context: Optional[Union[str, bool]], host_rewrite: bool) -> ServiceInfo:
        """
        Uniform handling of service definitions, TLS origination, etc.

        Here's how it goes:
        - If the service starts with https://, it is forced to originate TLS.
        - Else, if it starts with http://, it is forced to _not_ originate TLS.
        - Else, if the context is the boolean value True, it will originate TLS.

        After figuring that out, if we have a context which is a string value,
        we try to use that context name to look up certs to use. If we can't 
        find any, we won't send any originating cert.

        Finally, if no port is present in the service already, we force port 443
        if we're originating TLS, 80 if not.

        :param svc: URL of the service (from the Ambassador Mapping)
        :param context: TLS context name, or True to originate TLS but use no certs
        :param host_rewrite: Is host rewriting active?
        """

        originate_tls: Union[str, bool] = False
        name_fields: List[str] = []

        if svc.lower().startswith("http://"):
            originate_tls = False
            svc = svc[len("http://"):]
        elif svc.lower().startswith("https://"):
            originate_tls = True
            name_fields = [ 'otls' ]
            svc = svc[len("https://"):]
        elif context == True:
            originate_tls = True
            name_fields = [ 'otls' ]

        # Separate if here because you need to be able to specify a context
        # even after you say "https://" for the service.

        if context and (context != True):
            # We know that context is a string here.
            if context in self.tls_contexts:
                name_fields = [ 'otls', typecast(str, context) ]
                originate_tls = typecast(str, context)
            else:
                self.logger.error("Originate-TLS context %s is not defined" % context)

        if originate_tls and host_rewrite:
            name_fields.append("hr-%s" % host_rewrite)

        port = 443 if originate_tls else 80
        context_name = typecast(str, "_".join(name_fields) if name_fields else None)

        svc_url = 'tcp://%s' % svc

        if ':' not in svc:
            svc_url = '%s:%d' % (svc_url, port)

        return (svc, svc_url, bool(originate_tls), context_name)

    def add_clusters_for_mapping(self, mapping: Mapping) -> Tuple[str, Optional[str]]:
        """
        Given a Mapping, add the clusters we need for that Mapping to the IR.
        Returns the (possibly updated) service URL and main cluster name.

        :param Mapping: Mapping for which we need clusters
        :return: Tuple of (possibly updated) service URL and main cluster name
        """

        svc: str = mapping['service']
        tls_context: Optional[Union[str, bool]] = mapping.get('tls', None)
        grpc: bool = mapping.get('grpc', False)
        host_rewrite: Optional[str] = mapping.get('host_rewrite', None)

        # We're going to build up the cluster name for the main service in an array,
        # then join everything up afterward. Start with just the service name...

        cluster_name_fields = [ svc ]

        host_redirect: bool = mapping.get('host_redirect', False)
        shadow: bool = mapping.get('shadow', False)

        if host_redirect:
            if shadow:
                # Not allowed.
                errstr = "At most one of host_redirect and shadow may be set; ignoring host_redirect"
                self.post_error(RichStatus.fromError(errstr), resource=mapping)
                host_redirect = False
            else:
                # Short-circuit. You needn't actually create a cluster for a
                # host_redirect mapping.
                return svc, None

        if shadow:
            cluster_name_fields.insert(0, "shadow")

        # ...then do whatever normalization we need for the name and the URL. This can
        # change the service name (e.g. "http://foo" will turn into "foo"), which is why
        # we initialize cluster_name_fields above, before changing it. (This isn't a
        # functional issue, just a matter of trying not to confuse people trying to find
        # things later, especially across Ambassador upgrades where we might alter internal
        # stuff.)

        (svc, url, originate_tls, otls_name) = self.service_tls_check(svc, tls_context, bool(host_rewrite))

        # Build up the common name stuff that we'll need for the service and
        # the shadow service.
        #
        # XXX I don't think we need aux_name_fields any more.
        aux_name_fields: List[str] = []

        cb_name: Optional[str] = mapping.get('circuit_breaker', None)

        if cb_name:
            if cb_name in self.breakers:
                aux_name_fields.append("cb_%s" % cb_name)
            else:
                self.logger.error("CircuitBreaker %s is not defined (mapping %s)" %
                                  (cb_name, mapping.name))

        od_name: Optional[str] = mapping.get('outlier_detection', None)

        if od_name:
            if od_name in self.outliers:
                aux_name_fields.append("od_%s" % od_name)
            else:
                self.logger.error("OutlierDetection %s is not defined (mapping %s)" %
                                  (od_name, mapping.name))

        # OK. Use the main service stuff to build up the main clustor.

        if otls_name:
            cluster_name_fields.append(otls_name)

        cluster_name_fields.extend(aux_name_fields)

        cluster_name = 'cluster_%s' % "_".join(cluster_name_fields)
        cluster_name = re.sub(r'[^0-9A-Za-z_]', '_', cluster_name)

        self.logger.debug("%s: svc %s -> cluster %s" % (mapping.name, svc, cluster_name))

        self.add_intermediate_cluster(mapping['_source'], cluster_name,
                                      svc, [ url ],
                                      cb_name=cb_name, od_name=od_name, grpc=grpc,
                                      originate_tls=originate_tls, host_rewrite=host_rewrite)

        return svc, cluster_name

    def merge_tmods(self, tls_module, generated_module, key):
        """
        Merge TLS module configuration for a particular key. In the event of conflicts, the 
        tls_module element wins, and an error is posted so that the diagnostics service can 
        show it.

        Returns a TLS module with a correctly-merged config element. This will be the
        tls_module (possibly modified) unless no tls_module is present, in which case
        the generated_module will be promoted. If any changes were made to the module, it
        will be marked as referenced by the generated_module.

        :param tls_module: the `tls` module; may be None
        :param generated_module: the `tls-from-ambassador-certs` module; may be None
        :param key: the key in the module config to merge
        :return: TLS module object; see above.
        """

        # First up, the easy cases. If either module is missing, return the other.
        # (The other might be None too, of course.)
        if generated_module is None:
            return tls_module
        elif tls_module is None:
            return generated_module
        else:
            self.logger.debug("tls_module %s" % json.dumps(tls_module, indent=4))
            self.logger.debug("generated_module %s" % json.dumps(generated_module, indent=4))

            # OK, no easy cases. We know that both modules exist: grab the config dicts.
            tls_source = tls_module['_source']
            tls_config = tls_module.get(key, {})

            gen_source = generated_module['_source']
            gen_config = generated_module.get(key, {})

            # Now walk over the tls_config and copy anything needed.
            any_changes = False

            for ckey in gen_config:
                if ckey in tls_config:
                    # ckey exists in both modules. Do they have the same value?
                    if tls_config[ckey] != gen_config[ckey]:
                        # No -- post an error, but let the version from the TLS module win.
                        errfmt = "CONFLICT in TLS config for {}.{}: using {} from TLS module in {}"
                        errstr = errfmt.format(key, ckey, tls_config[ckey], tls_source)
                        self.post_error(RichStatus.fromError(errstr))
                    else:
                        # They have the same value. Worth mentioning in debug.
                        self.logger.debug("merge_tmods: {}.{} duplicated with same value".format(key, ckey))
                else:
                    # ckey only exists in gen_config. Copy it over.
                    self.logger.debug("merge_tmods: copy {}.{} from gen_config".format(key, ckey))
                    tls_config[ckey] = gen_config[ckey]
                    any_changes = True

            # If we had changes...
            if any_changes:
                # ...then mark the tls_module as referenced by the generated_module's
                # source..
                tls_module._mark_referenced_by(gen_source)

                # ...and copy the tls_config back in (in case the key wasn't in the tls_module 
                # config at all originally).
                tls_module[key] = tls_config

            # Finally, return the tls_module.
            return tls_module

    def generate_intermediate_config(self):
        # First things first. The "Ambassador" module always exists; create it with
        # default values now.

        self.ambassador_module = SourcedDict(
            service_port = 80,
            admin_port = 8001,
            diag_port = 8877,
            auth_enabled = None,
            liveness_probe = { "enabled": True },
            readiness_probe = { "enabled": True },
            diagnostics = { "enabled": True },
            tls_config = None,
            use_proxy_proto = False,
            x_forwarded_proto_redirect = False,
        )

        # Next up: let's define initial clusters, routes, and filters.
        #
        # Our set of clusters starts out empty; we use add_intermediate_cluster()
        # to build it up while making sure that all the source-tracking stuff
        # works out.
        #
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.
        self.envoy_clusters = {}

        # Our initial set of routes is empty...
        self.envoy_routes = {}

        # Our initial list of grpc_services is empty...
        self.envoy_config['grpc_services'] = []

        # Now we look at user-defined modules from our config...
        modules = self.config.get('modules', {})

        # ...most notably the 'ambassador' and 'tls' modules, which are handled first.
        amod = modules.get('ambassador', None)
        tls_module = modules.get('tls', None)

        # Part of handling the 'tls' module is folding in the 'tls-from-ambassador-certs'
        # module, so grab that too...
        generated_module = modules.get('tls-from-ambassador-certs', None)

        # ...and merge the 'server' and 'client' config elements.
        tls_module = self.merge_tmods(tls_module, generated_module, 'server')
        tls_module = self.merge_tmods(tls_module, generated_module, 'client')

        # OK, done. Make sure we have _something_ for the TLS module going forward.
        tmod = tls_module or {}
        self.logger.debug("TLS module after merge: %s" % json.dumps(tmod, indent=4))

        if amod or tmod:
            self.module_config_ambassador("ambassador", amod, tmod)

        # !!!! WARNING WARNING WARNING !!!! Filters are actually ORDER-DEPENDENT.
        self.envoy_config['filters'] = []
        # Start with authentication filter
        auth_mod = modules.get('authentication', None)
        auth_configs = self.config.get('auth_configs', None)
        auth_filter = self.module_config_authentication("authentication", amod, auth_mod, auth_configs)
        if auth_filter:
            self.envoy_config['filters'].append(auth_filter)
        # Then append the rate-limit filter, because we might rate-limit based on auth headers
        ratelimit_configs = self.config.get('ratelimit_configs', None)
        (ratelimit_filter, ratelimit_grpc_service) = self.module_config_ratelimit(ratelimit_configs)
        if ratelimit_filter and ratelimit_grpc_service:
            self.envoy_config['filters'].append(ratelimit_filter)
            self.envoy_config['grpc_services'].append(ratelimit_grpc_service)
        # Then append non-configurable cors and decoder filters
        self.envoy_config['filters'].append(SourcedDict(name="cors", config={}))
        self.envoy_config['filters'].append(SourcedDict(type="decoder", name="router", config={}))

        # For mappings, start with empty sets for everything.
        mappings = self.config.get("mappings", {})

        self.breakers = self.config.get("CircuitBreaker", {})

        for _, breaker in self.breakers.items():
            breaker['_referenced_by'] = []

        self.outliers = self.config.get("OutlierDetection", {})

        for _, outlier in self.outliers.items():
            outlier['_referenced_by'] = []

        # OK. Given those initial sets, let's look over our global modules.
        for module_name in modules.keys():
            if ((module_name == 'ambassador') or
                (module_name == 'tls') or
                (module_name == 'authentication') or
                (module_name == 'tls-from-ambassador-certs')):
                continue

            handler_name = "module_config_%s" % module_name
            handler = getattr(self, handler_name, None)

            if not handler:
                self.logger.error("module %s: no configuration generator, skipping" % module_name)
                continue

            handler(module_name, modules[module_name])

        # Once modules are handled, we can set up our admin config...
        self.envoy_config['admin'] = SourcedDict(
            _from=self.ambassador_module,
            admin_port=self.ambassador_module["admin_port"]
        )

        # ...and our listeners.
        primary_listener = SourcedDict(
            _from=self.ambassador_module,
            service_port=self.ambassador_module["service_port"],
            require_tls=False,
            use_proxy_proto=self.ambassador_module['use_proxy_proto']
        )

        if 'use_remote_address' in self.ambassador_module:
            primary_listener['use_remote_address'] = self.ambassador_module['use_remote_address']

        # If x_forwarded_proto_redirect is set, then we enable require_tls in primary listener, which in turn sets
        # require_ssl to true in envoy config. Once set, then all requests that contain X-FORWARDED-PROTO set to
        # https, are processes normally by envoy. In all the other cases, including X-FORWARDED-PROTO set to http,
        # a 301 redirect response to https://host is sent
        if self.ambassador_module.get('x_forwarded_proto_redirect', False):
            primary_listener['require_tls'] = True
            self.logger.debug("x_forwarded_proto_redirect is set to true, enabling 'require_tls' in listener")

        redirect_cleartext_from = None
        tmod = self.ambassador_module.get('tls_config', None)

        # ...TLS config, if necessary...
        if tmod:
            # self.logger.debug("USING TLS")
            primary_listener['tls'] = tmod
            if self.tmod_certs_exist(primary_listener['tls']) > 0:
                primary_listener['tls']['ssl_context'] = True
            redirect_cleartext_from = tmod.get('redirect_cleartext_from')

        self.envoy_config['listeners'] = [ primary_listener ]

        if redirect_cleartext_from:
            # We only want to set `require_tls` on the primary listener when certs are present on the pod
            if self.tmod_certs_exist(primary_listener['tls']) > 0:
                primary_listener['require_tls'] = True

            new_listener = SourcedDict(
                _from=self.ambassador_module,
                service_port=redirect_cleartext_from,
                require_tls=True,
                # Note: no TLS context here, this is a cleartext listener.
                # We can set require_tls True because we can let the upstream
                # tell us about that.
                use_proxy_proto=self.ambassador_module['use_proxy_proto']
            )

            if 'use_remote_address' in self.ambassador_module:
                new_listener['use_remote_address'] = self.ambassador_module['use_remote_address']

            self.envoy_config['listeners'].append(new_listener)

        self.default_liveness_probe['service'] = self.diag_service()
        self.default_readiness_probe['service'] = self.diag_service()
        self.default_diagnostics['service'] = self.diag_service()

        for name, cur, dflt in [
            ("liveness",    self.ambassador_module['liveness_probe'],
                            self.default_liveness_probe),
            ("readiness",   self.ambassador_module['readiness_probe'],
                            self.default_readiness_probe),
            ("diagnostics", self.ambassador_module['diagnostics'],
                            self.default_diagnostics)
        ]:
            if cur and cur.get("enabled", False):
                prefix = cur.get("prefix", dflt['prefix'])
                rewrite = cur.get("rewrite", dflt['rewrite'])
                service = cur.get("service", dflt['service'])

                # Push a fake mapping to handle this.
                name = "internal_%s_probe_mapping" % name

                mappings[name] = Mapping(
                    _from=self.ambassador_module,
                    kind='Mapping',
                    name=name,
                    prefix=prefix,
                    rewrite=rewrite,
                    service=service
                )

                # self.logger.debug("PROBE %s: %s -> %s%s" % (name, prefix, service, rewrite))

        # OK! We have all the mappings we need. Process them (don't worry about sorting
        # yet, we'll do that on routes).

        for mapping_name in sorted(mappings.keys()):
            mapping = mappings[mapping_name]

            # OK. Set up clusters for this service...
            svc, cluster_name = self.add_clusters_for_mapping(mapping)

            # ...and route.
            self.add_intermediate_route(mapping['_source'], mapping, svc, cluster_name)

        # OK. Walk the set of clusters and normalize names...
        collisions = {}
        mangled = {}

        for name in sorted(self.envoy_clusters.keys()):
            if len(name) > 60:
                # Too long.
                short_name = name[0:40]

                collision_list = collisions.setdefault(short_name, [])
                collision_list.append(name)

        for short_name in sorted(collisions.keys()):
            name_list = collisions[short_name]

            i = 0

            for name in sorted(name_list):
                mangled_name = "%s-%d" % (short_name, i)
                i += 1

                self.logger.info("%s => %s" % (name, mangled_name))

                mangled[name] = mangled_name
                self.envoy_clusters[name]['name'] = mangled_name

        # We need to default any unspecified weights and renormalize to 100
        for _, route in self.envoy_routes.items():
            clusters = route["clusters"]

            total = 0.0
            unspecified = 0

            # If this is a websocket route, it will support only one cluster right now.
            if route.get('use_websocket', False):
                if len(clusters) > 1:
                    errmsg = "Only one cluster is supported for websockets; using %s" % clusters[0]['name']
                    self.post_error(RichStatus.fromError(errmsg))

            for c in clusters:
                # Mangle the name, if need be.
                c_name = c["name"]

                if c_name in mangled:
                    c["name"] = mangled[c_name]
                    # self.logger.info("%s: mangling cluster %s to %s" % (group_id, c_name, c["name"]))

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

        # OK. When all is said and done, sort the list of routes by route weight...
        self.envoy_config['routes'] = sorted([
            route for group_id, route in self.envoy_routes.items()
        ], reverse=True, key=Mapping.route_weight)

        # ...then map clusters back into a list...
        self.envoy_config['clusters'] = [
            self.envoy_clusters[cluster_key] for cluster_key in sorted(self.envoy_clusters.keys())
        ]

        # ...and finally repeat for breakers and outliers, but copy them in the process so we
        # can mess with the originals.
        #
        # What's going on here is that circuit-breaker and outlier-detection configs aren't
        # included as independent objects in envoy.json, but we want to be able to discuss
        # them in diag. We also don't need to keep the _source and _referenced_by elements
        # in their real Envoy appearances.

        self.envoy_config['breakers'] = self.clean_and_copy(self.breakers)
        self.envoy_config['outliers'] = self.clean_and_copy(self.outliers)

    @staticmethod
    def tmod_certs_exist(tmod):
        """
        Returns the number of certs that are defined in the supplied tmod

        :param tmod: The TLS module configuration
        :return: number of certs in tmod
        :rtype: int
        """
        cert_count = 0
        if tmod.get('cert_chain_file') is not None:
            cert_count += 1
        if tmod.get('private_key_file') is not None:
            cert_count += 1
        if tmod.get('cacert_chain_file') is not None:
            cert_count += 1
        return cert_count

    def _get_intermediate_for(self, element_list, res_keys, value):
        if not isinstance(value, dict):
            return

        good = True

        if '_source' in value:
            good = False

            value_source = value.get("_source", None)
            value_referenced_by = value.get("_referenced_by", [])

            if ((value_source in res_keys) or
                (res_keys & set(value_referenced_by))):
                good = True

        if good:
            element_list.append(value)

    def get_intermediate_for(self, res_key):
        res_keys = []

        if res_key.startswith("grp-"):
            group_id = res_key[4:]

            for route in self.envoy_config['routes']:
                if route['_group_id'] == group_id:
                    res_keys.append(route['_source'])

                    for reference_key in route['_referenced_by']:
                        res_keys.append(reference_key)

            if not res_keys:
                return {
                    "error": "No group matches %s" % group_id
                }
        else:
            if res_key in self.location_map:
                # Exact match for a file in the source map: include all the objects
                # in the file.
                res_keys = self.location_map[res_key]
            elif res_key in self.sources:
                # Exact match for an object in a file: include only that object.
                res_keys.append(res_key)
            else:
                # No match at all. Weird.
                return {
                    "error": "No source matches %s" % res_key
                }

        res_keys = set(res_keys)

        # self.logger.debug("get_intermediate_for: res_keys %s" % res_keys)
        # self.logger.debug("get_intermediate_for: errors %s" % self.errors)

        sources = []

        for key in res_keys:
            source_dict = dict(self.sources[key])
            source_dict['errors'] = [
                {
                    'summary': error['error'].split('\n', 1)[0],
                    'text': error['error']
                }
                for error in self.errors.get(key, [])
            ]
            source_dict['res_key'] = key

            sources.append(source_dict)

        result = {
            "sources": sources
        }

        # self.logger.debug("get_intermediate_for: initial result %s" % result)

        for key in self.envoy_config.keys():
            result[key] = []

            value = self.envoy_config[key]

            if isinstance(value, list):
                for v2 in value:
                    self._get_intermediate_for(result[key], res_keys, v2)
            else:
                self._get_intermediate_for(result[key], res_keys, value)

        return result

    def generate_envoy_config(self, template=None, template_dir=None, **kwargs):
        # Finally! Render the template to JSON...
        envoy_json = self.to_json(template=template, template_dir=template_dir)

        # We used to use the JSON parser as a final sanity check here. That caused
        # Forge some issues, so it's turned off for now.

        # rc = RichStatus.fromError("impossible")

        # # ...and use the JSON parser as a final sanity check.
        # try:
        #     obj = json.loads(envoy_json)
        #     rc = RichStatus.OK(msg="Envoy configuration OK", envoy_config=obj)
        # except json.decoder.JSONDecodeError as e:
        #     rc = RichStatus.fromError("Invalid Envoy configuration: %s" % str(e),
        #                               raw=envoy_json, exception=e)

        # # Go ahead and report that we generated an Envoy config, if we can.
        # scout_result = Config.scout_report(action="config", result=True, generated=True, **kwargs)

        rc = RichStatus.OK(envoy_config=envoy_json) # , scout_result=scout_result

        # self.logger.debug("Scout reports %s" % json.dumps(rc.scout_result))

        return rc

    def set_config_ambassador(self, module, key, value, merge=False):
        if not merge:
            self.ambassador_module[key] = value
        else:
            self.ambassador_module[key].update(value)

        # XXX This is actually wrong sometimes. If, for example, you have an
        # ambassador module that defines the admin_port, sure, bringing in its
        # source makes sense. On the other hand, if you have a TLS module 
        # created by a secret, that source shouldn't really take over the
        # admin document. This will take enough unraveling that I'm going to
        # leave it for now, though.
        self.ambassador_module['_source'] = module['_source']

    def update_config_ambassador(self, module, key, value):
        self.set_config_ambassador(module, key, value, merge=True)

    def tls_config_helper(self, name, amod, tmod):
        tmp_config = SourcedDict(_from=amod)
        some_enabled = False

        for context_name in tmod.keys():
            if context_name.startswith('_'):
                continue

            context = tmod[context_name]

            # self.logger.debug("context %s -- %s" % (context_name, json.dumps(context)))

            if context.get('enabled', True):
                if context_name == 'server':
                    # Server-side TLS is enabled.
                    self.logger.debug("TLS termination enabled!")
                    some_enabled = True

                    # Switch to port 443 by default...
                    self.set_config_ambassador(amod, 'service_port', 443)

                    # ...and merge in the server-side defaults.
                    tmp_config.update(self.default_tls_config['server'])
                    tmp_config.update(tmod['server'])

                    # Check if secrets are supplied for TLS termination and/or TLS auth
                    secret = context.get('secret')
                    if secret is not None:
                        self.logger.debug("config.server.secret is {}".format(secret))
                        # If /{etc,ambassador}/certs/tls.crt does not exist, then load the secrets
                        if check_cert_file(TLSPaths.mount_tls_crt.value):
                            self.logger.debug("Secret already exists, taking no action for secret {}".format(secret))
                        elif check_cert_file(TLSPaths.tls_crt.value):
                            tmp_config['cert_chain_file'] = TLSPaths.tls_crt.value
                            tmp_config['private_key_file'] = TLSPaths.tls_key.value
                        else:
                            (server_cert, server_key, server_data) = read_cert_secret(kube_v1(), secret, self.namespace)
                            if server_cert and server_key:
                                self.logger.debug("saving contents of secret {} to {}".format(
                                    secret, TLSPaths.cert_dir.value))
                                save_cert(server_cert, server_key, TLSPaths.cert_dir.value)
                                tmp_config['cert_chain_file'] = TLSPaths.tls_crt.value
                                tmp_config['private_key_file'] = TLSPaths.tls_key.value

                elif context_name == 'client':
                    # Client-side TLS is enabled.
                    self.logger.debug("TLS client certs enabled!")
                    some_enabled = True

                    # Merge in the client-side defaults.
                    tmp_config.update(self.default_tls_config['client'])
                    tmp_config.update(tmod['client'])

                    secret = context.get('secret')
                    if secret is not None:
                        self.logger.debug("config.client.secret is {}".format(secret))
                        if check_cert_file(TLSPaths.client_mount_crt.value):
                            self.logger.debug("Secret already exists, taking no action for secret {}".format(secret))
                        elif check_cert_file(TLSPaths.client_tls_crt.value):
                            tmp_config['cacert_chain_file'] = TLSPaths.client_tls_crt.value
                        else:
                            (client_cert, _, _) = read_cert_secret(kube_v1(), secret, self.namespace)
                            if client_cert:
                                self.logger.debug("saving contents of secret {} to  {}".format(
                                    secret, TLSPaths.client_cert_dir.value))
                                save_cert(client_cert, None, TLSPaths.client_cert_dir.value)
                                tmp_config['cacert_chain_file'] = TLSPaths.client_tls_crt.value

                else:
                    # This is a wholly new thing.
                    self.tls_contexts[context_name] = SourcedDict(
                        _from=tmod,
                        **context
                    )

        if some_enabled:
            if 'enabled' in tmp_config:
                del(tmp_config['enabled'])

            # Save the TLS config...
            self.set_config_ambassador(amod, 'tls_config', tmp_config)

        self.logger.debug("TLS config: %s" % json.dumps(self.ambassador_module['tls_config'], indent=4))
        self.logger.debug("TLS contexts: %s" % json.dumps(self.tls_contexts, indent=4))

        return some_enabled

    def module_config_ambassador(self, name, amod, tmod):
        # Toplevel Ambassador configuration. First up, check out TLS.

        have_amod_tls = False

        if amod and ('tls' in amod):
            have_amod_tls = self.tls_config_helper(name, amod, amod['tls'])

        if not have_amod_tls and tmod:
            self.tls_config_helper(name, tmod, tmod)

        # After that, check for port definitions, probes, etc., and copy them in
        # as we find them.
        for key in [ 'service_port', 'admin_port', 'diag_port',
                     'liveness_probe', 'readiness_probe', 'auth_enabled',
                     'use_proxy_proto', 'use_remote_address', 'diagnostics', 'x_forwarded_proto_redirect' ]:
            if amod and (key in amod):
                # Yes. It overrides the default.
                self.set_config_ambassador(amod, key, amod[key])

    def module_config_ratelimit(self, ratelimit_config):
        cluster_hosts = None
        sources = []

        if ratelimit_config:
            for config in ratelimit_config.values():
                sources.append(config['_source'])
                cluster_hosts = config.get("service", None)

        if not cluster_hosts or not sources:
            return (None, None)

        host_rewrite = config.get("host_rewrite", None)

        cluster_name = "cluster_ext_ratelimit"
        filter_config = {
            "domain": "ambassador",
            "request_type": "both",
            "timeout_ms": 20
        }
        grpc_service = SourcedDict(
            name="rate_limit_service",
            cluster_name=cluster_name
        )

        first_source = sources.pop(0)

        filter = SourcedDict(
            _source=first_source,
            type="decoder",
            name="rate_limit",
            config=filter_config
        )

        if cluster_name not in self.envoy_clusters:
            # (svc, url, originate_tls, otls_name) = self.service_tls_check(cluster_hosts, None, host_rewrite)
            (_, url, _, _) = self.service_tls_check(cluster_hosts, None, host_rewrite)
            self.add_intermediate_cluster(first_source, cluster_name,
                                          'extratelimit', [url],
                                          type="strict_dns", lb_type="round_robin",
                                          grpc=True, host_rewrite=host_rewrite)

        for source in sources:
            filter._mark_referenced_by(source)
            self.envoy_clusters[cluster_name]._mark_referenced_by(source)

        return (filter, grpc_service)

    def auth_helper(self, sources, config, cluster_hosts, module):
        sources.append(module['_source'])

        for key in [ 'path_prefix', 'timeout_ms', 'cluster' ]:
            value = module.get(key, None)

            if value != None:
                previous = config.get(key, None)

                if previous and (previous != value):
                    errstr = (
                        "AuthService cannot support multiple %s values; using %s" %
                        (key, previous)
                    )

                    self.post_error(RichStatus.fromError(errstr), key=module['_source'])
                else:
                    config[key] = value

        headers = module.get('allowed_headers', None)

        if headers:
            allowed_headers = config.get('allowed_headers', [])

            for hdr in headers:
                if hdr not in allowed_headers:
                    allowed_headers.append(hdr)

            config['allowed_headers'] = allowed_headers

        auth_service = module.get("auth_service", None)
        # weight = module.get("weight", 100)
        weight = 100    # Can't support arbitrary weights right now.

        if auth_service:
            cluster_hosts[auth_service] = ( weight, module.get('tls', None) )

    def module_config_authentication(self, name, amod, auth_mod, auth_configs):
        filter_config = {
            "cluster": "cluster_ext_auth",
            "timeout_ms": 5000
        }

        cluster_hosts = {}
        sources = []

        if auth_mod:
            self.auth_helper(sources, filter_config, cluster_hosts, auth_mod)

        if auth_configs:
            # self.logger.debug("auth_configs: %s" % auth_configs)
            for config in auth_configs.values():
                self.auth_helper(sources, filter_config, cluster_hosts, config)

        if not sources:
            return None

        first_source = sources.pop(0)

        filter = SourcedDict(
            _source=first_source,
            _services=sorted(cluster_hosts.keys()),
            type="decoder",
            name="extauth",
            config=filter_config
        )

        cluster_name = filter_config['cluster']
        host_rewrite = filter_config.get('host_rewrite', None)

        if cluster_name not in self.envoy_clusters:
            if not cluster_hosts:
                cluster_hosts = { '127.0.0.1:5000': ( 100, None ) }

            urls = []
            protocols = {}

            for svc in sorted(cluster_hosts.keys()):
                _, tls_context = cluster_hosts[svc]

                (svc, url, originate_tls, otls_name) = self.service_tls_check(svc, tls_context, host_rewrite)

                if originate_tls:
                    protocols['https'] = True
                else:
                    protocols['http'] = True

                if otls_name:
                    filter_config['cluster'] = cluster_name + "_" + otls_name
                    cluster_name = filter_config['cluster']

                urls.append(url)

            if len(protocols.keys()) != 1:
                raise Exception("auth config cannot try to use both HTTP and HTTPS")

            self.add_intermediate_cluster(first_source, cluster_name,
                                          'extauth', urls,
                                          type="strict_dns", lb_type="round_robin",
                                          originate_tls=originate_tls, host_rewrite=host_rewrite)

        for source in sources:
            filter._mark_referenced_by(source)
            self.envoy_clusters[cluster_name]._mark_referenced_by(source)

        return filter

    ### DIAGNOSTICS
    def diagnostic_overview(self):
        # Build a set of source _files_ rather than source _objects_.
        source_files = {}

        for filename, res_keys in self.location_map.items():
            # self.logger.debug("overview -- filename %s, res_keys %d" %
            #                   (filename, len(res_keys)))

            # # Skip '--internal--' etc.
            # if filename.startswith('--'):
            #     continue

            source_dict = source_files.setdefault(
                filename,
                {
                    'filename': filename,
                    'objects': {},
                    'count': 0,
                    'plural': "objects",
                    'error_count': 0,
                    'error_plural': "errors"
                }
            )

            for res_key in res_keys:
                # self.logger.debug("overview --- res_key %s" % res_key)

                source = self.sources[res_key]

                if ('source' in source) and not ('source' in source_dict):
                    source_dict['source'] = source['source']

                raw_errors = self.errors.get(res_key, [])

                errors = []

                for error in raw_errors:
                    source_dict['error_count'] += 1

                    errors.append({
                        'summary': error['error'].split('\n', 1)[0],
                        'text': error['error']
                    })

                source_dict['error_plural'] = "error" if (source_dict['error_count'] == 1) else "errors"

                source_dict['count'] += 1
                source_dict['plural'] = "object" if (source_dict['count'] == 1) else "objects"

                object_dict = source_dict['objects']
                object_dict[res_key] = {
                    'key': res_key,
                    'kind': source['kind'],
                    'errors': errors
                }

        routes = []

        for route in self.envoy_config['routes']:
            if route['_source'] != "--diagnostics--":
                route['_group_id'] = Mapping.group_id(route.get('method', 'GET'),
                                                      route['prefix'] if 'prefix' in route else route['regex'],
                                                      route.get('headers', []))

                routes.append(route)

        configuration = { key: self.envoy_config[key] for key in self.envoy_config.keys()
                          if key != "routes" }

        # Is extauth active?
        extauth = None
        filters = configuration.get('filters', [])

        for filter in filters:
            if filter['name'] == 'extauth':
                extauth = filter

                extauth['_service_weight'] = 100.0 / len(extauth['_services'])

        overview = dict(sources=sorted(source_files.values(), key=lambda x: x['filename']),
                        routes=routes,
                        **configuration)

        if extauth:
            overview['extauth'] = extauth

        # self.logger.debug("overview result %s" % json.dumps(overview, indent=4, sort_keys=True))

        return overview

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

if __name__ == '__main__':
    aconf = Config(sys.argv[1])
    print(json.dumps(aconf.diagnostic_overview(), indent=4, sort_keys=True))
