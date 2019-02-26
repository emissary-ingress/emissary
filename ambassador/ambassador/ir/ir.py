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

from typing import Any, Callable, Dict, Iterable, List, Optional, Type, Union, ValuesView
from typing import cast as typecast

import json
import logging
import os

from ..utils import RichStatus, SavedSecret
from ..config import Config

from .irresource import IRResource
from .irambassador import IRAmbassador
from .irauth import IRAuth
from .irfilter import IRFilter
from .ircluster import IRCluster
from .irbasemappinggroup import IRBaseMappingGroup
from .irbasemapping import IRBaseMapping
from .irmappingfactory import MappingFactory
from .irratelimit import IRRateLimit
from .irtls import TLSModuleFactory, IRAmbassadorTLS
from .irlistener import ListenerFactory, IRListener
from .irtracing import IRTracing
from .irtlscontext import IRTLSContext

from ..VERSION import Version, Build

#############################################################################
## ir.py -- the Ambassador Intermediate Representation (IR)
##
## After getting an ambassador.Config, you can create an ambassador.IR. The
## IR is the basis for everything else: you can use it to configure an Envoy
## or to run diagnostics.


def error_secret_reader(context: IRTLSContext, secret_name: str, namespace: str) -> SavedSecret:
    # Failsafe only.
    return SavedSecret(secret_name, namespace, None, None, {})


class IR:
    ambassador_module: IRAmbassador
    ambassador_id: str
    ambassador_namespace: str
    ambassador_nodename: str
    tls_module: Optional[IRAmbassadorTLS]
    tracing: Optional[IRTracing]
    ratelimit: Optional[IRRateLimit]
    router_config: Dict[str, Any]
    filters: List[IRFilter]
    listeners: List[IRListener]
    groups: Dict[str, IRBaseMappingGroup]
    clusters: Dict[str, IRCluster]
    grpc_services: Dict[str, IRCluster]
    saved_resources: Dict[str, IRResource]
    tls_contexts: Dict[str, IRTLSContext]
    aconf: Config
    secret_root: str
    secret_reader: Callable[[IRTLSContext, str, str], SavedSecret]
    file_checker: Callable[[str], bool]

    def __init__(self, aconf: Config, secret_reader=None, file_checker=None) -> None:
        self.ambassador_id = Config.ambassador_id
        self.ambassador_namespace = Config.ambassador_namespace
        self.ambassador_nodename = aconf.ambassador_nodename
        self.statsd = aconf.statsd

        self.logger = logging.getLogger("ambassador.ir")

        # We're using setattr since since mypy complains about assigning directly to a method.
        secret_root = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', "/ambassador")
        setattr(self, 'secret_reader', secret_reader or error_secret_reader)
        setattr(self, 'file_checker', file_checker if file_checker is not None else os.path.isfile)

        self.logger.debug("IR __init__:")
        self.logger.debug("IR: Version         %s built from %s on %s" % (Version, Build.git.commit, Build.git.branch))
        self.logger.debug("IR: AMBASSADOR_ID   %s" % self.ambassador_id)
        self.logger.debug("IR: Namespace       %s" % self.ambassador_namespace)
        self.logger.debug("IR: Nodename        %s" % self.ambassador_nodename)

        self.logger.debug("IR: file checker:   %s" % getattr(self, 'file_checker').__name__)
        self.logger.debug("IR: secret reader:  %s" % getattr(self, 'secret_reader').__name__)

        # First up: save the Config object. Its source map may be necessary later.
        self.aconf = aconf

        # Next, we'll want a way to keep track of resources we end up working
        # with. It starts out empty.
        self.saved_resources = {}

        # Next, define the initial IR state -- which is empty.
        #
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.
        self.clusters = {}
        self.grpc_services = {}
        self.filters = []
        self.tracing = None
        self.tls_contexts = {}
        self.ratelimit = None
        self.listeners = []
        self.groups = {}

        # Set up default TLS stuff.
        #
        # XXX This feels like a hack -- shouldn't it be class-wide initialization
        # in TLSModule or TLSContext? So far it's the only place we need anything like
        # this though.

        self.tls_module = None

        # OK! Start by wrangling TLS-context stuff, both from the TLS module (if any)...
        TLSModuleFactory.load_all(self, aconf)

        # ...and from any TLSContext resources.
        self.save_tls_contexts(aconf)

        # Next, handle the "Ambassador" module. This is last so that the Ambassador module has all
        # the TLS contexts available to it.
        self.ambassador_module = typecast(IRAmbassador, self.save_resource(IRAmbassador(self, aconf)))

        # Save breaker & outlier configs.
        self.breakers = aconf.get_config("CircuitBreaker") or {}
        self.outliers = aconf.get_config("OutlierDetection") or {}
        self.endpoints = aconf.get_config("endpoints") or {}
        self.service_info = aconf.get_config("service_info") or {}

        # Save tracing and ratelimit settings.
        self.tracing = typecast(IRTracing, self.save_resource(IRTracing(self, aconf)))
        self.ratelimit = typecast(IRRateLimit, self.save_resource(IRRateLimit(self, aconf)))

        # After the Ambassador and TLS modules are done, we need to set up the
        # filter chains, which requires checking in on the auth, and
        # ratelimit configuration. Note that order of the filters matter.        
        self.save_filter(IRAuth(self, aconf))

        # ...note that ratelimit is a filter too...
        if self.ratelimit:
            self.save_filter(self.ratelimit, already_saved=True)

        # ...then deal with the non-configurable cors filter...
        self.save_filter(IRFilter(ir=self, aconf=aconf,
                                  rkey="ir.cors", kind="ir.cors", name="cors",
                                  config={}))

        # ...and the marginally-configurable router filter.
        router_config = {}

        if self.tracing:
            router_config['start_child_span'] = True

        self.save_filter(IRFilter(ir=self, aconf=aconf,
                                  rkey="ir.router", kind="ir.router", name="router", type="decoder",
                                  config=router_config))

        # We would handle other modules here -- but guess what? There aren't any.
        # At this point ambassador, tls, and the deprecated auth module are all there
        # are, and they're handled above. So. At this point go sort out all the Mappings
        ListenerFactory.load_all(self, aconf)
        MappingFactory.load_all(self, aconf)

        self.walk_saved_resources(aconf, 'add_mappings')

        TLSModuleFactory.finalize(self, aconf)
        ListenerFactory.finalize(self, aconf)
        MappingFactory.finalize(self, aconf)

        # At this point we should know the full set of clusters, so we can normalize
        # any long cluster names.
        collisions: Dict[str, List[str]] = {}

        for name in sorted(self.clusters.keys()):
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
                self.clusters[name]['name'] = mangled_name

        # After we have the cluster names fixed up, go finalize filters.
        if self.tracing:
            self.tracing.finalize()

        if self.ratelimit:
            self.ratelimit.finalize()

        for filter in self.filters:
            filter.finalize()

    # XXX Brutal hackery here! Probably this is a clue that Config and IR and such should have
    # a common container that can hold errors.
    def post_error(self, rc: Union[str, RichStatus], resource: Optional[IRResource]=None):
        self.aconf.post_error(rc, resource=resource)

    def save_resource(self, resource: IRResource) -> IRResource:
        if resource.is_active():
            self.saved_resources[resource.rkey] = resource

        return resource

    # Save TLS contexts from the aconf into the IR. Note that the contexts in the aconf
    # are just ACResources; they need to be turned into IRTLSContexts.
    def save_tls_contexts(self, aconf):
        tls_contexts = aconf.get_config('tls_contexts')

        if tls_contexts is not None:
            for config in tls_contexts.values():
                ctx = IRTLSContext(self, config)

                if ctx.is_active():
                    self.save_tls_context(ctx)

    def save_tls_context(self, ctx: IRTLSContext) -> None:
        extant_ctx = self.tls_contexts.get(ctx.name, None)

        if extant_ctx:
            self.post_error("Duplicate TLSContext %s; keeping definition from %s" % (ctx.name, extant_ctx.location))
        else:
            self.tls_contexts[ctx.name] = ctx

    # def has_tls_context(self, name: str) -> bool:
    #     return bool(self.get_tls_context(name))

    def get_tls_context(self, name: str) -> Optional[IRTLSContext]:
        return self.tls_contexts.get(name, None)

    def get_tls_contexts(self) -> ValuesView[IRTLSContext]:
        return self.tls_contexts.values()

    def save_filter(self, resource: IRFilter, already_saved=False) -> None:
        if resource.is_active():
            if not already_saved:
                resource = typecast(IRFilter, self.save_resource(resource))

            self.filters.append(resource)

    def walk_saved_resources(self, aconf, method_name):
        for res in self.saved_resources.values():
            getattr(res, method_name)(self, aconf)

    def add_listener(self, listener: IRListener) -> None:
        self.listeners.append(listener)

    def add_to_listener(self, listener_name: str, **kwargs) -> bool:
        for listener in self.listeners:
            if listener.get('name') == listener_name:
                listener.update(kwargs)
                return True
        return False

    def add_to_primary_listener(self, **kwargs) -> bool:
        primary_listener = 'ir.listener'
        return self.add_to_listener(primary_listener, **kwargs)

    def add_mapping(self, aconf: Config, mapping: IRBaseMapping) -> Optional[IRBaseMappingGroup]:
        group: IRBaseMappingGroup = None

        if mapping.is_active():
            if mapping.group_id not in self.groups:
                group_name = "GROUP: %s" % mapping.name
                group_class = mapping.group_class()
                group = group_class(ir=self, aconf=aconf,
                                    location=mapping.location,
                                    name=group_name,
                                    mapping=mapping)

                self.groups[group.group_id] = group
            else:
                group = self.groups[mapping.group_id]
                group.add_mapping(aconf, mapping)

        return group

    def ordered_groups(self) -> Iterable[IRBaseMappingGroup]:
        return reversed(sorted(self.groups.values(), key=lambda x: x['group_weight']))

    def has_cluster(self, name: str) -> bool:
        return name in self.clusters

    def get_cluster(self, name: str) -> Optional[IRCluster]:
        return self.clusters.get(name, None)

    def add_cluster(self, cluster: IRCluster) -> IRCluster:
        if not self.has_cluster(cluster.name):
            self.clusters[cluster.name] = cluster

        return self.clusters[cluster.name]

    def merge_cluster(self, cluster: IRCluster) -> bool:
        extant = self.get_cluster(cluster.name)

        if extant:
            return extant.merge(cluster)
        else:
            self.add_cluster(cluster)
            return True

    def has_grpc_service(self, name: str) -> bool:
        return name in self.grpc_services

    def add_grpc_service(self, name: str, cluster: IRCluster) -> IRCluster:
        if not self.has_grpc_service(name):
            if not self.has_cluster(cluster.name):
                self.clusters[cluster.name] = cluster

            self.grpc_services[name] = cluster

        return self.grpc_services[name]

    def as_dict(self) -> Dict[str, Any]:
        od = {
            'identity': {
                'ambassador_id': self.ambassador_id,
                'ambassador_namespace': self.ambassador_namespace,
                'ambassador_nodename': self.ambassador_nodename,
            },
            'ambassador': self.ambassador_module.as_dict(),
            'clusters': { cluster_name: cluster.as_dict()
                          for cluster_name, cluster in self.clusters.items() },
            'grpc_services': { svc_name: cluster.as_dict()
                               for svc_name, cluster in self.grpc_services.items() },
            'listeners': [ listener.as_dict() for listener in self.listeners ],
            'filters': [ filt.as_dict() for filt in self.filters ],
            'groups': [ group.as_dict() for group in self.ordered_groups() ],
            'tls_contexts': [ context.as_dict() for context in self.tls_contexts.values() ]
        }

        if self.tracing:
            od['tracing'] = self.tracing.as_dict()

        if self.ratelimit:
            od['ratelimit'] = self.ratelimit.as_dict()

        return od

    def as_json(self) -> str:
        return json.dumps(self.as_dict(), sort_keys=True, indent=4)

    def features(self) -> Dict[str, Any]:
        od: Dict[str, Union[bool, int, Optional[str]]] = {}

        tls_termination_count = 0   # TLS termination contexts
        tls_origination_count = 0   # TLS origination contexts

        using_tls_module = False
        using_tls_contexts = False

        for ctx in self.get_tls_contexts():
            if ctx:
                secret_info = ctx.get('secret_info', {})

                if secret_info:
                    using_tls_contexts = True

                    if secret_info.get('certificate_chain_file', None):
                        tls_termination_count += 1

                    if secret_info.get('cacert_chain_file', None):
                        tls_origination_count += 1

                if ctx.get('_legacy', False):
                    using_tls_module = True

        od['tls_using_module'] = using_tls_module
        od['tls_using_contexts'] = using_tls_contexts
        od['tls_termination_count'] = tls_termination_count
        od['tls_origination_count'] = tls_origination_count

        for key in [ 'diagnostics', 'liveness_probe', 'readiness_probe', 'statsd' ]:
            od[key] = self.ambassador_module.get(key, {}).get('enabled', False)

        for key in [ 'use_proxy_proto', 'use_remote_address', 'x_forwarded_proto_redirect' ]:
            od[key] = self.ambassador_module.get(key, False)

        od['custom_ambassador_id'] = bool(self.ambassador_id != 'default')

        default_port = 443 if tls_termination_count else 80

        od['custom_listener_port'] = bool(self.ambassador_module.service_port != default_port)
        od['custom_diag_port'] = bool(self.ambassador_module.diag_port != 8877)

        cluster_count = 0
        cluster_grpc_count = 0      # clusters using GRPC upstream
        cluster_http_count = 0      # clusters using HTTP or HTTPS upstream
        cluster_tls_count = 0       # clusters using TLS origination

        endpoint_grpc_count = 0     # endpoints using GRPC upstream
        endpoint_http_count = 0     # endpoints using HTTP/HTTPS upstream
        endpoint_tls_count = 0      # endpoints using TLS origination

        for cluster in self.clusters.values():
            cluster_count += 1
            using_tls = False
            using_http = False
            using_grpc = False

            if cluster.get('tls_context', None):
                using_tls = True
                cluster_tls_count += 1

            if cluster.get('grpc', False):
                using_grpc = True
                cluster_grpc_count += 1
            else:
                using_http = True
                cluster_http_count += 1

            for url in cluster.urls:
                if using_tls:
                    endpoint_tls_count += 1

                if using_http:
                    endpoint_http_count += 1

                if using_grpc:
                    endpoint_grpc_count += 1

        od['cluster_count'] = cluster_count
        od['cluster_grpc_count'] = cluster_grpc_count
        od['cluster_http_count'] = cluster_http_count
        od['cluster_tls_count'] = cluster_tls_count
        od['endpoint_grpc_count'] = endpoint_grpc_count
        od['endpoint_http_count'] = endpoint_http_count
        od['endpoint_tls_count'] = endpoint_tls_count

        extauth = False
        extauth_proto: Optional[str] = None
        extauth_allow_body = False
        extauth_host_count = 0

        ratelimit = False
        ratelimit_data_plane_proto = False
        ratelimit_custom_domain = False

        tracing = False
        tracing_driver: Optional[str] = None

        for filter in self.filters:
            if filter.kind == 'IRAuth':
                extauth = True
                extauth_proto = filter.get('proto', 'http')
                extauth_allow_body = filter.get('allow_request_body', False)
                extauth_host_count = len(filter.hosts.keys())

        if self.ratelimit:
            ratelimit = True
            ratelimit_data_plane_proto = self.ratelimit.get('data_plane_proto', False)
            ratelimit_custom_domain = bool(self.ratelimit.domain != 'ambassador')

        if self.tracing:
            tracing = True
            tracing_driver = self.tracing.driver

        od['extauth'] = extauth
        od['extauth_proto'] = extauth_proto
        od['extauth_allow_body'] = extauth_allow_body
        od['extauth_host_count'] = extauth_host_count
        od['ratelimit'] = ratelimit
        od['ratelimit_data_plane_proto'] = ratelimit_data_plane_proto
        od['ratelimit_custom_domain'] = ratelimit_custom_domain
        od['tracing'] = tracing
        od['tracing_driver'] = tracing_driver

        group_count = 0
        group_precedence_count = 0      # groups using explicit precedence
        group_header_match_count = 0    # groups using header matches
        group_regex_header_count = 0    # groups using regex header matches
        group_regex_prefix_count = 0    # groups using regex prefix matches
        group_shadow_count = 0          # groups using shadows
        group_host_redirect_count = 0   # groups using host_redirect
        group_host_rewrite_count = 0    # groups using host_rewrite
        group_canary_count = 0          # groups coalescing multiple mappings
        mapping_count = 0               # total mappings

        for group in self.ordered_groups():
            group_count += 1

            if group.get('precedence', 0) != 0:
                group_precedence_count += 1

            using_headers = False
            using_regex_headers = False

            for header in group.get('headers', []):
                using_headers = True

                if header['regex']:
                    using_regex_headers = True
                    break

            if using_headers:
                group_header_match_count += 1

            if using_regex_headers:
                group_regex_header_count += 1

            if len(group.mappings) > 1:
                group_canary_count += 1

            mapping_count += len(group.mappings)

            if group.get('shadows', []):
                group_shadow_count += 1

            if group.get('host_redirect', {}):
                group_host_redirect_count += 1

            if group.get('host_rewrite', None):
                group_host_rewrite_count += 1

        od['group_count'] = group_count
        od['group_precedence_count'] = group_precedence_count
        od['group_header_match_count'] = group_header_match_count
        od['group_regex_header_count'] = group_regex_header_count
        od['group_regex_prefix_count'] = group_regex_prefix_count
        od['group_shadow_count'] = group_shadow_count
        od['group_host_redirect_count'] = group_host_redirect_count
        od['group_host_rewrite_count'] = group_host_rewrite_count
        od['group_canary_count'] = group_canary_count
        od['mapping_count'] = mapping_count

        od['listener_count'] = len(self.listeners)

        return od
