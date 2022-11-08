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
import collections
import logging
import os
import socket
from typing import TYPE_CHECKING, Any, ClassVar, Dict, Iterable, List, Optional, Tuple, Union
from typing import cast as typecast

from pkg_resources import Requirement, resource_filename

from multi import multi

from ..resource import Resource
from ..utils import RichStatus, dump_json, parse_bool
from .acresource import ACResource

if TYPE_CHECKING:
    from ambassador.fetch.fetcher import ResourceFetcher

#############################################################################
## config.py -- the main configuration parser for Ambassador
##
## Ambassador configures itself by creating a new Config object, which calls
## Config.__init__().

# Custom types
# StringOrList is either a string or a list of strings.
StringOrList = Union[str, List[str]]


def envoy_api_version():
    """
    Return the Envoy API version we should be using.
    """
    env_version = os.environ.get("AMBASSADOR_ENVOY_API_VERSION", "V3")

    version = env_version.upper()

    if version == "V2" or env_version == "V3":
        return version

    return "V2"


class Config:
    # CLASS VARIABLES
    # When using multiple Ambassadors in one cluster, use AMBASSADOR_ID to distinguish them.
    ambassador_id: ClassVar[str] = os.environ.get("AMBASSADOR_ID", "default")
    ambassador_namespace: ClassVar[str] = os.environ.get("AMBASSADOR_NAMESPACE", "default")
    single_namespace: ClassVar[bool] = bool(os.environ.get("AMBASSADOR_SINGLE_NAMESPACE"))
    certs_single_namespace: ClassVar[bool] = bool(
        os.environ.get(
            "AMBASSADOR_CERTS_SINGLE_NAMESPACE", os.environ.get("AMBASSADOR_SINGLE_NAMESPACE")
        )
    )
    enable_endpoints: ClassVar[bool] = not bool(os.environ.get("AMBASSADOR_DISABLE_ENDPOINTS"))
    log_resources: ClassVar[bool] = parse_bool(os.environ.get("AMBASSADOR_LOG_RESOURCES"))
    envoy_api_version: ClassVar[str] = envoy_api_version()
    envoy_bind_address: ClassVar[str] = os.environ.get("AMBASSADOR_ENVOY_BIND_ADDRESS", "0.0.0.0")

    StorageByKind: ClassVar[Dict[str, str]] = {
        "authservice": "auth_configs",
        "consulresolver": "resolvers",
        "host": "hosts",
        "listener": "listeners",
        "mapping": "mappings",
        "kubernetesendpointresolver": "resolvers",
        "kubernetesserviceresolver": "resolvers",
        "ratelimitservice": "ratelimit_configs",
        "devportal": "devportals",
        "tcpmapping": "tcpmappings",
        "tlscontext": "tls_contexts",
        "tracingservice": "tracing_configs",
        "logservice": "log_services",
    }

    SupportedVersions: ClassVar[Dict[str, str]] = {
        "v2": "is deprecated, consider upgrading",
        "v3alpha1": "ok",
    }

    # INSTANCE VARIABLES
    ambassador_nodename: str = "ambassador"  # overridden in Config.reset

    schema_dir_path: str  # where to look for JSONSchema files
    current_resource: Optional[ACResource] = None
    helm_chart: Optional[str]

    config: Dict[str, Dict[str, ACResource]]

    breakers: Dict[str, ACResource]
    outliers: Dict[str, ACResource]

    counters: Dict[str, int]

    # rkey => ACResource
    sources: Dict[str, ACResource]

    # Invalid objects (currently loaded using load_invalid())
    invalid: List[Dict]

    errors: Dict[str, List[dict]]  # errors to post to the UI
    notices: Dict[str, List[str]]  # notices to post to the UI
    fatal_errors: int
    object_errors: int

    def __init__(self, schema_dir_path: Optional[str] = None) -> None:
        self.logger = logging.getLogger("ambassador.config")

        if not schema_dir_path:
            # Note that this "resource_filename" has to do with setuptool packages, not
            # with our ACResource class.
            schema_dir_path = resource_filename(Requirement.parse("ambassador"), "schemas")

        # Once here, we know that schema_dir_path cannot be None. assert that, for mypy's
        # benefit.
        assert schema_dir_path is not None

        self.statsd: Dict[str, Any] = {
            "enabled": (os.environ.get("STATSD_ENABLED", "").lower() == "true"),
            "dogstatsd": (os.environ.get("DOGSTATSD", "").lower() == "true"),
        }

        if self.statsd["enabled"]:
            self.statsd["interval"] = os.environ.get("STATSD_FLUSH_INTERVAL", "1")

            statsd_host = os.environ.get("STATSD_HOST", "statsd-sink")
            try:
                resolved_ip = socket.gethostbyname(statsd_host)
                self.statsd["ip"] = resolved_ip
            except socket.gaierror as e:
                self.logger.error("Unable to resolve {} to IP : {}".format(statsd_host, e))
                self.logger.error("Stats will not be exported to {}".format(statsd_host))
                self.statsd["enabled"] = False

        self.schema_dir_path = schema_dir_path

        self.logger.debug("SCHEMA DIR    %s" % os.path.abspath(self.schema_dir_path))
        self.k8s_status_updates: Dict[
            str, Tuple[str, str, Optional[Dict[str, Any]]]
        ] = {}  # Tuple is (name, namespace, status_json)
        self.pod_labels: Dict[str, str] = {}
        self._reset()

    def _reset(self) -> None:
        """
        Resets this Config to the empty, default state so it can load a new config.
        """

        self.logger.debug("ACONF RESET")

        self.current_resource = None
        self.helm_chart = None

        self.config = {}

        self.breakers = {}
        self.outliers = {}

        self.counters = collections.defaultdict(lambda: 0)

        self.invalid = []
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
        self.ambassador_nodename = "%s-%s" % (
            os.environ.get("AMBASSADOR_ID", "ambassador"),
            Config.ambassador_namespace,
        )

    def __str__(self) -> str:
        s = ["<Config:"]

        for kind, configs in self.config.items():
            s.append("  %s:" % kind)

            for rkey, resource in configs.items():
                s.append("    %s" % resource)

        s.append(">")

        return "\n".join(s)

    def as_dict(self) -> Dict[str, Any]:
        od: Dict[str, Any] = {"_errors": self.errors, "_notices": self.notices, "_sources": {}}

        if self.helm_chart:
            od["_helm_chart"] = self.helm_chart

        for k, v in self.sources.items():
            sd = dict(v)  # Shallow copy

            if "_errors" in v:
                sd["_errors"] = [x.as_dict() for x in v._errors]

            od["_sources"][k] = sd

        for kind, configs in self.config.items():
            od[kind] = {}

            for rkey, config in configs.items():
                od[kind][rkey] = config.as_dict()

        return od

    def as_json(self):
        return dump_json(self.as_dict(), pretty=True)

    # Often good_ambassador_id will be passed an ACResource, but sometimes
    # just a plain old dict.
    def good_ambassador_id(self, resource: dict):
        resource_kind = resource.get("kind", "")

        # Is an ambassador_id present in this object?
        #
        # NOTE WELL: when we update the status of a Host (or a Mapping?) then reserialization
        # can cause the `ambassador_id` element to turn into an `ambassadorId` element. So
        # treat those as synonymous.
        allowed_ids: StringOrList = resource.get("ambassadorId", None)

        if allowed_ids is None:
            allowed_ids = resource.get("ambassador_id", "default")

        # If we find the array [ '_automatic_' ] then allow it, so that hardcoded resources
        # can have a useful effect. This is mostly for init-config, but could be used for
        # other things, too.

        if allowed_ids == ["_automatic_"]:
            self.logger.debug(f"ambassador_id {allowed_ids} always accepted")
            return True

        if allowed_ids:
            # Make sure it's a list. Yes, this is Draconian,
            # but the jsonschema will allow only a string or a list,
            # and guess what? Strings are Iterables.
            if type(allowed_ids) != list:
                allowed_ids = typecast(StringOrList, [allowed_ids])

            if Config.ambassador_id in allowed_ids:
                return True
            else:
                rkey = resource.get("rkey", "-anonymous-yaml-")
                name = resource.get("name", "-no-name-")

                self.logger.debug(
                    f"{rkey}: {resource_kind} {name} has IDs {allowed_ids}, no match with {Config.ambassador_id}"
                )
                return False

    def incr_count(self, key: str) -> None:
        self.counters[key] += 1

    def get_count(self, key: str) -> int:
        return self.counters.get(key, 0)

    def save_source(self, resource: ACResource) -> None:
        """
        Save a given ACResource as a source of Ambassador config information.
        """
        self.sources[resource.rkey] = resource

    def load_invalid(self, fetcher: "ResourceFetcher") -> None:
        """
        Loads the invalid resources from a ResourceFetcher. This and load_all() should be
        combined.
        """

        self.invalid = fetcher.invalid

    def load_all(self, resources: Iterable[ACResource]) -> None:
        """
        Loads all of a set of ACResources. It is the caller's responsibility to arrange for
        the set of ACResources to be sorted in some way that makes sense.
        """

        self.logger.debug(f"Loading config")

        rcount = 0

        for resource in resources:
            if Config.log_resources:
                self.logger.debug("Trying to parse resource: %s", resource)

            rcount += 1

            if not self.good_ambassador_id(resource):
                continue

            if Config.log_resources:
                self.logger.debug("LOAD_ALL: %s @ %s", resource, resource.location)
            else:
                self.logger.debug("LOAD_ALL: process %s", resource.location)

            rc = self.process(resource)

            if not rc:
                # Object error. Not good but we'll allow the system to start.
                self.post_error(rc, resource=resource)

        self.logger.debug("LOAD_ALL: processed %d resource%s", rcount, "" if (rcount == 1) else "s")

        if self.fatal_errors:
            # Kaboom.
            raise Exception("ERROR ERROR ERROR Unparseable configuration; exiting")

        if self.errors:
            self.logger.error("ERROR ERROR ERROR Starting with configuration errors")

    def post_notice(
        self, msg: str, resource: Optional[Resource] = None, log_level=logging.DEBUG
    ) -> None:
        if resource is None:
            resource = self.current_resource

        rkey = "-global-"

        if resource is not None:
            rkey = resource.rkey

        notices = self.notices.setdefault(rkey, [])
        notices.append(msg)

        self.logger.log(log_level, "%s: NOTICE: %s" % (rkey, msg))

    @multi
    def post_error(
        self,
        msg: Union[RichStatus, str],
        resource: Optional[Resource] = None,
        rkey: Optional[str] = None,
        log_level=logging.INFO,
    ) -> str:
        del resource  # silence warnings
        del rkey

        if isinstance(msg, RichStatus):
            return "RichStatus"
        elif isinstance(msg, str):
            return "string"
        else:
            return type(msg).__name__

    @post_error.when("string")
    def post_error_string(
        self,
        msg: str,
        resource: Optional[Resource] = None,
        rkey: Optional[str] = None,
        log_level=logging.INFO,
    ):
        rc = RichStatus.fromError(msg)

        self.post_error(rc, resource=resource, log_level=log_level)

    @post_error.when("RichStatus")
    def post_error_richstatus(
        self,
        rc: RichStatus,
        resource: Optional[Resource] = None,
        rkey: Optional[str] = None,
        log_level=logging.INFO,
    ):
        if resource is None:
            resource = self.current_resource

        if not rkey:
            rkey = "-global-"

            if resource is not None:
                rkey = resource.rkey

                if isinstance(resource, ACResource):
                    self.save_source(resource)

        errors = self.errors.setdefault(rkey, [])
        errors.append(rc.as_dict())

        self.logger.log(log_level, "%s: %s" % (rkey, rc))

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
        if "name" not in resource:
            return RichStatus.fromError("need name")

        # ...and also make sure it has a namespace.
        if not resource.get("namespace", None):
            resource["namespace"] = self.ambassador_namespace

        # ...it doesn't actually need a metadata_labels, so off we go. Save the source info...
        self.save_source(resource)

        # ...and figure out if this thing is OK.
        rc = self.validate_object(resource)

        if not rc:
            # Well that's no good.
            return rc

        # OK, so far so good. Should we just stash this somewhere?
        lkind = resource.kind.lower()
        store_as = Config.StorageByKind.get(lkind)

        if store_as:
            # Just stash it.
            self.safe_store(store_as, resource)
        else:
            # Can't just stash it. Is there a handler for this kind of resource?
            handler_name = f"handle_{lkind}"
            handler = getattr(self, handler_name, None)

            if not handler:
                self.logger.warning(
                    "%s: no handler for %s, just saving" % (resource, resource.kind)
                )
                handler = self.save_object
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

        if apiVersion.startswith("getambassador.io/"):
            version = apiVersion.split("/", 1)[1].lower()
            status = Config.SupportedVersions.get(version, "is not supported")
            if status != "ok":
                self.post_notice(f"apiVersion {apiVersion} {status}", resource=resource)
        elif apiVersion.startswith("networking.internal.knative.dev"):
            # This is not an Ambassador resource, we're trying to parse Knative
            # here
            pass
        else:
            return RichStatus.fromError("apiVersion %s unsupported" % apiVersion)

        ns = resource.get("namespace") or self.ambassador_namespace
        name = f"{resource.name} ns {ns}"

        # Did entrypoint.go flag errors here that we should show to the user?
        #
        # (It's still called watt_errors because our other docs talk about "watt
        # snapshots", and I'm OK with retaining that name for the format.)
        if "errors" in resource:
            # Pop the errors out of this resource...
            errors = resource.pop("errors").split("\n")

            # ...strip any empty lines in the error list with this one weird list
            # comprehension...
            watt_errors = "; ".join([error for error in errors if error])

            # ...and, assuming that we're left with any error message, post it.
            if watt_errors:
                return RichStatus.fromError(watt_errors)

        return RichStatus.OK(msg=f"good {resource.kind}")

    def safe_store(self, storage_name: str, resource: ACResource, allow_log: bool = True) -> None:
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
            if resource.namespace == storage[resource.name].get("namespace"):
                # If the name and namespace, both match, then it's definitely an error.
                # Oooops.
                self.post_error(
                    "%s defines %s %s, which is already defined by %s"
                    % (resource, resource.kind, resource.name, storage[resource.name].location),
                    resource=resource,
                )
            else:
                # Here, we deal with the case when multiple resources have the same name but they exist in different
                # namespaces. Our current data structure to store resources is a flat string. Till we move to
                # identifying resources with both, name and namespace, we change names of any subsequent resources with
                # the same name here.
                resource.name = f"{resource.name}.{resource.namespace}"

        if allow_log:
            self.logger.debug("%s: saving %s %s" % (resource, resource.kind, resource.name))

        storage[resource.name] = resource

    def save_object(self, resource: ACResource, allow_log: bool = False) -> None:
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

    def module_lookup(self, module_name: str, key: str, default: Any = None) -> Any:
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

    def handle_secret(self, resource: ACResource) -> None:
        """
        Handles a Secret resource. We need a handler for this because the key needs to be
        the rkey, not the name.
        """

        self.logger.debug(f"Handling secret resource {resource.as_dict()}")

        storage = self.config.setdefault("secrets", {})
        key = resource.rkey

        if key in storage:
            self.post_error(
                "%s defines %s %s, which is already defined by %s"
                % (resource, resource.kind, key, storage[key].location),
                resource=resource,
            )

        storage[key] = resource

    def handle_ingress(self, resource: ACResource) -> None:
        storage = self.config.setdefault("ingresses", {})
        key = resource.rkey

        if key in storage:
            self.post_error(
                "%s defines %s %s, which is already defined by %s"
                % (resource, resource.kind, key, storage[key].location),
                resource=resource,
            )

        storage[key] = resource

    def handle_service(self, resource: ACResource) -> None:
        """
        Handles a Service resource. We need a handler for this because the key needs to be
        the rkey, not the name, and because we need to check the helm_chart attribute.
        """

        storage = self.config.setdefault("service", {})
        key = resource.rkey

        if key in storage:
            self.post_error(
                "%s defines %s %s, which is already defined by %s"
                % (resource, resource.kind, key, storage[key].location),
                resource=resource,
            )

        self.logger.debug("%s: saving %s %s" % (resource, resource.kind, key))

        storage[key] = resource

        chart = resource.get("helm_chart", None)

        if chart:
            self.helm_chart = chart
