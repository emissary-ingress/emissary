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

from typing import Any, Dict, List, Optional
from typing import cast as typecast

import sys

import json
import logging
import os

from ..utils import TLSPaths
from ..config import Config

from .irresource import IRResource
from .irambassador import IRAmbassador
from .irauth import IRAuth
from .irfilter import IRFilter
from .ircluster import IRCluster
from .irmapping import MappingFactory, IRMapping, IRMappingGroup
from .irratelimit import IRRateLimit
from .irtls import TLSModuleFactory, IRAmbassadorTLS, IREnvoyTLS
from .irlistener import ListenerFactory, IRListener
from .irtracing import IRTracing
from .irtlscontext import IRTLSContext

#from .VERSION import Version

#############################################################################
## ir.py -- the Ambassador Intermediate Representation (IR)
##
## After getting an ambassador.Config, you can create an ambassador.IR. The
## IR is the basis for everything else: you can use it to configure an Envoy
## or to run diagnostics.


class IR:
    ambassador_module: IRAmbassador
    ambassador_id: str
    ambassador_namespace: str
    ambassador_nodename: str
    tls_module: Optional[IRAmbassadorTLS]
    tracing: Optional[IRTracing]
    ratelimit: Optional[IRRateLimit]
    router_config: Dict[str, Any]
    filters: List[IRResource]
    listeners: List[IRListener]
    groups: Dict[str, IRMappingGroup]
    clusters: Dict[str, IRCluster]
    grpc_services: Dict[str, IRCluster]
    saved_resources: Dict[str, IRResource]
    envoy_tls: Dict[str, IREnvoyTLS]
    tls_contexts: List[IRTLSContext]
    tls_defaults: Dict[str, Dict[str, str]]
    aconf: Config

    def __init__(self, aconf: Config, tls_secret_resolver=None, file_checker=None) -> None:
        self.ambassador_id = Config.ambassador_id
        self.ambassador_namespace = Config.ambassador_namespace
        self.ambassador_nodename = aconf.ambassador_nodename

        self.logger = logging.getLogger("ambassador.ir")
        self.tls_secret_resolver = tls_secret_resolver
        self.file_checker = file_checker if file_checker is not None else os.path.isfile
        self.logger.debug("File checker: {}".format(self.file_checker.__name__))

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
        self.tls_contexts = []
        self.ratelimit = None
        self.listeners = []
        self.groups = {}

        # Set up default TLS stuff.
        #
        # XXX This feels like a hack -- shouldn't it be class-wide initialization
        # in TLSModule or TLSContext? So far it's the only place we need anything like
        # this though.

        self.tls_module = None
        self.envoy_tls = {}
        self.tls_defaults = {
            "server": {},
            "client": {},
        }

        if os.path.isfile(TLSPaths.mount_tls_crt.value):
            self.tls_defaults["server"]["cert_chain_file"] = TLSPaths.mount_tls_crt.value

        if os.path.isfile(TLSPaths.mount_tls_key.value):
            self.tls_defaults["server"]["private_key_file"] = TLSPaths.mount_tls_key.value

        if os.path.isfile(TLSPaths.client_mount_crt.value):
            self.tls_defaults["client"]["cacert_chain_file"] = TLSPaths.client_mount_crt.value

        # OK! Start by wrangling TLS-context stuff.
        TLSModuleFactory.load_all(self, aconf)

        # Next, handle the "Ambassador" module.
        self.ambassador_module = typecast(IRAmbassador, self.save_resource(IRAmbassador(self, aconf)))

        # Save breaker & outlier configs.
        self.breakers = aconf.get_config("CircuitBreaker") or {}
        self.outliers = aconf.get_config("OutlierDetection") or {}

        # Save tracing and ratelimit settings.
        self.tracing = self.save_resource(IRTracing(self, aconf))
        self.ratelimit = self.save_resource(IRRateLimit(self, aconf))

        # Save TLSContext resource settings.
        self.save_tls_contexts(aconf)

        # After the Ambassador and TLS modules are done, we need to set up the
        # filter chains, which requires checking in on the auth, and
        # ratelimit configuration stuff.
        #
        # ORDER MATTERS HERE.
        # After the Ambassador and TLS modules are done, check in on auth...
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

    def save_resource(self, resource: IRResource) -> IRResource:
        if resource.is_active():
            self.saved_resources[resource.rkey] = resource

        return resource

    def save_tls_contexts(self, aconf):
        tls_contexts = aconf.get_config('tls_contexts')
        if tls_contexts is not None:
            for config in tls_contexts.values():
                resource = IRTLSContext(self, config)
                if resource.is_active():
                    self.tls_contexts.append(resource)

    def get_tls_contexts(self):
        return self.tls_contexts

    def save_filter(self, resource: IRResource, already_saved=False) -> None:
        if resource.is_active():
            if not already_saved:
                resource = self.save_resource(resource)

            self.filters.append(resource)

    def walk_saved_resources(self, aconf, method_name):
        for res in self.saved_resources.values():
            getattr(res, method_name)(self, aconf)

    def save_envoy_tls_context(self, ctx_name: str, ctx: IREnvoyTLS) -> bool:
        if ctx_name in self.envoy_tls:
            return False

        self.envoy_tls[ctx_name] = ctx
        return True

    def get_envoy_tls_context(self, ctx_name: str) -> Optional[IREnvoyTLS]:
        return self.envoy_tls.get(ctx_name, None)

    def get_tls_defaults(self, ctx_name: str) -> Optional[Dict]:
        return self.tls_defaults.get(ctx_name, None)

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

    def add_mapping(self, aconf: Config, mapping: IRMapping) -> Optional[IRMappingGroup]:
        group: Optional[IRMappingGroup] = None

        if mapping.is_active():
            if mapping.group_id not in self.groups:
                group_name = "GROUP: %s" % mapping.name
                group = IRMappingGroup(ir=self, aconf=aconf,
                                       location=mapping.location,
                                       name=group_name,
                                       mapping=mapping)

                self.groups[group.group_id] = group
            else:
                group = self.groups[mapping.group_id]
                group.add_mapping(aconf, mapping)

        return group

    def ordered_groups(self) -> List[IRMappingGroup]:
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
            'envoy_tls_contexts': {ctx_name: ctx.as_dict()
                             for ctx_name, ctx in self.envoy_tls.items()},
            'listeners': [ listener.as_dict() for listener in self.listeners ],
            'filters': [ filter.as_dict() for filter in self.filters ],
            'groups': [ group.as_dict() for group in self.ordered_groups() ],
            'tls_contexts': [context.as_dict() for context in self.tls_contexts]
        }

        if self.tracing:
            od['tracing'] = self.tracing.as_dict()

        if self.ratelimit:
            od['ratelimit'] = self.ratelimit.as_dict()

        return od

    def as_json(self):
        return json.dumps(self.as_dict(), sort_keys=True, indent=4)
