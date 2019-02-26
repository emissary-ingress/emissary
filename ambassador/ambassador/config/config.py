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
import socket

from typing import Any, ClassVar, Dict, Iterable, List, Optional, Tuple, Union
from typing import cast as typecast

import json
import logging
import os

import jsonschema

from multi import multi
from pkg_resources import Requirement, resource_filename

from ..utils import RichStatus

from ..resource import Resource
from .acresource import ACResource
from .acmapping import ACMapping


#############################################################################
## config.py -- the main configuration parser for Ambassador
##
## Ambassador configures itself by creating a new Config object, which calls
## Config.__init__().

# Custom types
# StringOrList is either a string or a list of strings.
StringOrList = Union[str, List[str]]


class Config:
    # CLASS VARIABLES
    # When using multiple Ambassadors in one cluster, use AMBASSADOR_ID to distinguish them.
    ambassador_id: ClassVar[str] = os.environ.get('AMBASSADOR_ID', 'default')
    ambassador_namespace: ClassVar[str] = os.environ.get('AMBASSADOR_NAMESPACE', 'default')
    single_namespace: ClassVar[bool] = bool(os.environ.get('AMBASSADOR_SINGLE_NAMESPACE'))

    # INSTANCE VARIABLES
    ambassador_nodename: str = "ambassador"     # overridden in Config.reset

    current_resource: Optional[ACResource] = None

    # XXX flat wrong
    schemas: Dict[str, dict]
    config: Dict[str, Dict[str, ACResource]]

    breakers: Dict[str, ACResource]
    outliers: Dict[str, ACResource]

    # rkey => ACResource
    sources: Dict[str, ACResource]

    errors: Dict[str, List[dict]]
    notices: Dict[str, List[str]]
    fatal_errors: int
    object_errors: int

    def __init__(self, schema_dir_path: Optional[str]=None) -> None:

        self.logger = logging.getLogger("ambassador.config")

        if not schema_dir_path:
            # Note that this "resource_filename" has to do with setuptool packages, not
            # with our ACResource class.
            schema_dir_path = resource_filename(Requirement.parse("ambassador"), "schemas")

        self.statsd: Dict[str, Any] = {
            'enabled': (os.environ.get('STATSD_ENABLED', '').lower() == 'true')
        }

        if self.statsd['enabled']:
            self.statsd['interval'] = os.environ.get('STATSD_FLUSH_INTERVAL', '1')

            statsd_host = os.environ.get('STATSD_HOST', 'statsd-sink')
            try:
                resolved_ip = socket.gethostbyname(statsd_host)
                self.statsd['ip'] = resolved_ip
            except socket.gaierror as e:
                self.logger.error("Unable to resolve {} to IP : {}".format(statsd_host, e))
                self.logger.error("Stats will not be exported to {}".format(statsd_host))
                self.statsd['enabled'] = False

        self.schema_dir_path = schema_dir_path

        self.logger.debug("SCHEMA DIR    %s" % os.path.abspath(self.schema_dir_path))

        self._reset()

    def _reset(self) -> None:
        """
        Resets this Config to the empty, default state so it can load a new config.
        """

        self.logger.debug("ACONF RESET")

        self.current_resource = None

        self.schemas = {}
        self.config = {}

        self.breakers = {}
        self.outliers = {}

        self.sources = {}

        # Save our magic internal sources.
        self.save_source(ACResource.internal_resource())
        self.save_source(ACResource.diagnostics_resource())

        self.errors = {}
        self.notices = {}
        self.fatal_errors = 0
        self.object_errors = 0

        # Build up the Ambassador node name.
        #
        # XXX This should be overrideable by the Ambassador module.
        self.ambassador_nodename = "%s-%s" % (os.environ.get('AMBASSADOR_ID', 'ambassador'),
                                              Config.ambassador_namespace)

    def __str__(self) -> str:
        s = [ "<Config:" ]

        for kind, configs in self.config.items():
            s.append("  %s:" % kind)

            for rkey, resource in configs.items():
                s.append("    %s" % resource)

        s.append(">")

        return "\n".join(s)

    def as_dict(self) -> Dict[str, Any]:
        od: Dict[str, Any] = {
            '_errors': self.errors,
            '_notices': self.notices,
            '_sources': {}
        }

        for k, v in self.sources.items():
            sd = dict(v)    # Shallow copy

            if '_errors' in v:
                sd['_errors'] = [ x.as_dict() for x in v._errors ]

            od['_sources'][k] = sd

        for kind, configs in self.config.items():
            od[kind] = {}

            for rkey, config in configs.items():
                od[kind][rkey] = config.as_dict()

        return od

    def as_json(self):
        return json.dumps(self.as_dict(), sort_keys=True, indent=4)

    # Often good_ambassador_id will be passed an ACResource, but sometimes
    # just a plain old dict.
    def good_ambassador_id(self, resource: dict):
        # Is an ambassador_id present in this object?
        allowed_ids: StringOrList = resource.get('ambassador_id', 'default')

        if allowed_ids:
            # Make sure it's a list. Yes, this is Draconian,
            # but the jsonschema will allow only a string or a list,
            # and guess what? Strings are Iterables.
            if type(allowed_ids) != list:
                allowed_ids = typecast(StringOrList, [ allowed_ids ])

            return Config.ambassador_id in allowed_ids

    def save_source(self, resource: ACResource) -> None:
        """
        Save a given ACResource as a source of Ambassador config information.
        """
        self.sources[resource.rkey] = resource

    def load_all(self, resources: Iterable[ACResource]) -> None:
        """
        Loads all of a set of ACResources. It is the caller's responsibility to arrange for
        the set of ACResources to be sorted in some way that makes sense.
        """

        rcount = 0

        for resource in resources:
            rcount += 1

            if not self.good_ambassador_id(resource):
                continue

            self.logger.debug("LOAD_ALL: %s @ %s" % (resource, resource.location))

            rc = self.process(resource)

            if not rc:
                # Object error. Not good but we'll allow the system to start.
                self.post_error(rc, resource=resource)

        self.logger.debug("LOAD_ALL: processed %d resource%s" % (rcount, "" if (rcount == 1) else "s"))

        if self.fatal_errors:
            # Kaboom.
            raise Exception("ERROR ERROR ERROR Unparseable configuration; exiting")

        if self.errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

    # # Utility methods built around ResourceFetcher. Ambassador doesn't really use these running
    # # "for real" but they make life easier for the CLI.
    # def fetch_resources(self, config_dir_path: str, k8s=False, recurse=False):
    #     fetcher = ResourceFetcher(self, config_dir_path, k8s=k8s, recurse=recurse)
    #     return fetcher.__iter__()
    #
    # def load_from_directory(self, config_dir_path: str, k8s=False, recurse=False, key=lambda x: x.rkey) -> None:
    #     """
    #     Load all the resources contained in YAML files in a given directory. To be considered,
    #     the files must have names ending in '.yaml' (case insensitive).
    #
    #     By default, resources are sorted according to their rkey before loading. Pass a different
    #     sort function as key if you want to change the sort order.
    #
    #     This is a really just a convenience method that uses a ResourceFetcher to find the resources,
    #     sorts them, and then calls self.load_all().
    #
    #     :param config_dir_path: the directory to search for YAML files
    #     :param k8s: should we expect that the files we find are annotated K8s resources?
    #     :param key: sort function; defaults to lambda x: x.rkey
    #     """
    #
    #     raw = list(self.fetch_resources(config_dir_path, k8s=k8s, recurse=recurse))
    #     resources = sorted(raw, key=key)
    #
    #     self.load_all(resources)

    def post_notice(self, msg: str, resource: Optional[Resource]=None) -> None:
        if resource is None:
            resource = self.current_resource

        rkey = '-global-'

        if resource is not None:
            rkey = resource.rkey

        notices = self.notices.setdefault(rkey, [])
        notices.append(msg)
        self.logger.info("%s: NOTICE: %s" % (rkey, msg))

    @multi
    def post_error(self, msg: Union[RichStatus, str], resource: Optional[Resource]=None) -> str:
        if isinstance(msg, RichStatus):
            return 'RichStatus'
        elif isinstance(msg, str):
            return 'string'
        else:
            return type(msg).__name__

    @post_error.when('string')
    def post_error_string(self, msg: str, resource: Optional[Resource]=None):
        rc = RichStatus.fromError(msg)

        self.post_error(rc, resource=resource)

    @post_error.when('RichStatus')
    def post_error_richstatus(self, rc: RichStatus, resource: Optional[Resource]=None):
        if resource is None:
            resource = self.current_resource

        rkey = '-global-'

        if resource is not None:
            rkey = resource.rkey
            # resource.post_error(rc)

            if isinstance(resource, ACResource):
                self.save_source(resource)
        # elif not unparsed_resource:
        #     raise Exception("FATAL: trying to post an error from a totally unknown resource??")

        # XXX Probably don't need this data structure, since we can walk the source
        # list and get them all.
        errors = self.errors.setdefault(rkey, [])
        errors.append(rc.as_dict())
        self.logger.error("%s: %s" % (rkey, rc))

    def process(self, resource: ACResource) -> RichStatus:
        # This should be impossible.
        if not resource:
            return RichStatus.fromError("undefined object???")

        self.current_resource = resource

        if not resource.apiVersion:
            return RichStatus.fromError("need apiVersion")

        if not resource.kind:
            return RichStatus.fromError("need kind")

        # Make sure this resource has a name...
        if 'name' not in resource:
            return RichStatus.fromError("need name")

        # ...and off we go. Save the source info...
        self.save_source(resource)

        # ...and figure out if this thing is OK.
        rc = self.validate_object(resource)

        if not rc:
            # Well that's no good.
            return rc

        # Is this a v0 resource?
        version = resource.apiVersion.lower()

        if version != 'ambassador/v1':
            desc = "is deprecated, consider upgrading"

            if version != 'ambassador/v0':
                desc = "is not supported"

            self.post_notice("apiVersion ambassador/v0 %s" % desc, resource=resource)

        # OK, so far so good. Grab the handler for this object type.
        handler_name = "handle_%s" % resource.kind.lower()
        handler = getattr(self, handler_name, None)

        if not handler:
            handler = self.save_object
            self.logger.warning("%s: no handler for %s, just saving" % (resource, resource.kind))
        # else:
        #     self.logger.debug("%s: handling %s..." % (resource, resource.kind))

        try:
            handler(resource)
        except Exception as e:
            # Bzzzt.
            raise
            # return RichStatus.fromError("%s: could not process %s object: %s" % (resource, resource.kind, e))

        # OK, all's well.
        self.current_resource = None

        return RichStatus.OK(msg="%s object processed successfully" % resource.kind)

    def validate_object(self, resource: ACResource) -> RichStatus:
        # This is basically "impossible"
        if not (("apiVersion" in resource) and ("kind" in resource) and ("name" in resource)):
            return RichStatus.fromError("must have apiVersion, kind, and name")

        apiVersion = resource.apiVersion

        # Ditch the leading ambassador/ that really needs to be there.
        if apiVersion.startswith("ambassador/"):
            apiVersion = apiVersion.split('/')[1]
        else:
            return RichStatus.fromError("apiVersion %s unsupported" % apiVersion)

        # Do we already have this schema loaded?
        schema_key = "%s-%s" % (apiVersion, resource.kind)
        schema = self.schemas.get(schema_key, None)

        if not schema:
            # Not loaded. Go find it on disk.
            schema_path = os.path.join(self.schema_dir_path, apiVersion,
                                       "%s.schema" % resource.kind)

            try:
                # Load it up...
                schema = json.load(open(schema_path, "r"))

                # ...and then cache it, if it exists. Note that we'll never
                # get here if we find something that doesn't parse.
                if schema:
                    self.schemas[schema_key] = typecast(Dict[Any, Any], schema)
            except OSError:
                self.logger.debug("no schema at %s, not validating" % schema_path)
            except json.decoder.JSONDecodeError as e:
                self.logger.warning("corrupt schema at %s, skipping (%s)" %
                                    (schema_path, e))

        if schema:
            # We have a schema. Does the object validate OK?
            try:
                jsonschema.validate(resource.as_dict(), schema)
            except jsonschema.exceptions.ValidationError as e:
                # Nope. Bzzzzt.
                return RichStatus.fromError("not a valid %s: %s" % (resource.kind, e))

        # All good. Return an OK.
        return RichStatus.OK(msg="valid %s" % resource.kind)

    def safe_store(self, storage_name: str, resource: ACResource, allow_log: bool=True) -> None:
        """
        Safely store a ACResource under a given storage name. The storage_name is separate
        because we may need to e.g. store a Module under the 'ratelimit' name or the like.
        Within a storage_name bucket, the ACResource will be stored under its name.

        :param storage_name: where shall we file this?
        :param resource: what shall we file?
        :param allow_log: if True, logs that we're saving this thing.
        """

        storage = self.config.setdefault(storage_name, {})

        if resource.name in storage:
            # Oooops.
            self.post_error("%s defines %s %s, which is already defined by %s" %
                            (resource, resource.kind, resource.name, storage[resource.name].location),
                            resource=resource)

        if allow_log:
            self.logger.debug("%s: saving %s %s" %
                              (resource, resource.kind, resource.name))

        storage[resource.name] = resource

    def save_object(self, resource: ACResource, allow_log: bool=False) -> None:
        """
        Saves a ACResource using its kind as the storage class name. Sort of the
        defaulted version of safe_store.

        :param resource: what shall we file?
        :param allow_log: if True, logs that we're saving this thing.
        """

        self.safe_store(resource.kind, resource, allow_log=allow_log)

    def get_config(self, key: str) -> Optional[Dict[str, ACResource]]:
        return self.config.get(key, None)

    def get_module(self, module_name: str) -> Optional[ACResource]:
        """
        Fetch a module from the module store. Can return None if no
        such module exists.

        :param module_name: name of the module you want.
        """

        modules = self.get_config("modules")

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

    def handle_module(self, resource: ACResource) -> None:
        """
        Handles a Module resource.
        """

        # Make a new ACResource from the 'config' element of this ACResource
        # Note that we leave the original serialization intact, since it will
        # indeed show a human the YAML that defined this module.
        #
        # XXX This should be Module.from_resource()...
        module_resource = ACResource.from_resource(resource, kind="Module", **resource.config)

        self.safe_store("modules", module_resource)

    def handle_ratelimitservice(self, resource: ACResource) -> None:
        """
        Handles a RateLimitService resource.
        """

        self.safe_store("ratelimit_configs", resource)

    def handle_tracingservice(self, resource: ACResource) -> None:
        """
        Handles a TracingService resource.
        """

        self.safe_store("tracing_configs", resource)

    def handle_authservice(self, resource: ACResource) -> None:
        """
        Handles an AuthService resource.
        """

        self.safe_store("auth_configs", resource)

    def handle_tlscontext(self, resource: ACResource) -> None:
        """
        Handles a TLSContext resource.
        """

        self.safe_store("tls_contexts", resource)

    def handle_endpoints(self, resource: ACResource) -> None:
        """
        Handles an Endpoints resource.
        """

        self.safe_store("endpoints", resource)

    def handle_serviceinfo(self, resource: ACResource) -> None:
        """
        Handles an ServiceInfo resource.
        """

        self.safe_store("service_info", resource)

    def handle_mapping(self, resource: ACMapping) -> None:
        """
        Handles a ACMapping resource.

        Mappings are complex things, so a lot of stuff gets buried in a ACMapping 
        object.
        """

        self.safe_store("mappings", resource)

    def handle_tcpmapping(self, resource: ACResource) -> None:
        """
        Handles a TCPMapping resource.
        """

        self.safe_store("tcpmappings", resource)
