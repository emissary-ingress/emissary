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

from typing import Any, Dict, List, Optional, Tuple, Union

import sys

import json
import logging
import os

from ..utils import TLSPaths
from ..config import Config

from .irresource import IRResource
from .irambassador import IRAmbassador
from .irauth import IRAuth
from .ircluster import IRCluster
from .irmapping import MappingFactory, IRMapping, IRMappingGroup
from .irratelimit import IRRateLimit
from .irtls import IREnvoyTLS, IRAmbassadorTLS
from .irlistener import ListenerFactory, IRListener

#from .VERSION import Version

#############################################################################
## ir.py -- the Ambassador Intermediate Representation (IR)
##
## After getting an ambassador.Config, you can create an ambassador.IR. The
## IR is the basis for everything else: you can use it to configure an Envoy
## or to run diagnostics.

# Custom types
# ServiceInfo is a tuple of information about a service:
# service name, service URL, originate TLS?, TLS context name
ServiceInfo = Tuple[str, str, bool, str]

# StringOrList is either a string or a list of strings.
StringOrList = Union[str, List[str]]


class IR:
    ambassador_module: IRAmbassador
    # clusters: Dict[str, Cluster]
    # routes: Dict[str, Route]

    router_config: Dict[str, Any]
    filters: List[IRResource]
    listeners: List[IRListener]
    groups: Dict[str, IRMappingGroup]
    clusters: Dict[str, IRCluster]
    tls_contexts: Dict[str, IREnvoyTLS]
    tls_defaults: Dict[str, Dict[str, str]]

    def __init__(self, aconf: Config) -> None:
        self.logger = logging.getLogger("ambassador.ir")

        # First up: let's define initial clusters, routes, and filters.
        #
        # Our set of clusters starts out empty; we use add_intermediate_cluster()
        # to build it up while making sure that all the source-tracking stuff
        # works out.
        #
        # Note that we use a map for clusters, not a list -- the reason is that
        # multiple mappings can use the same service, and we don't want multiple
        # clusters.
        self.clusters = {}

        # Our initial configuration stuff is all empty...
        self.router_config = {}
        self.filters = []
        self.tracing_config = None
        self.listeners = []
        self.groups = {}

        # self.routes = {}
        # self.grpc_services = []

        # Set up default TLS stuff.
        #
        # XXX This feels like a hack -- shouldn't it be class-wide initialization
        # in TLSModule or TLSContext? So far it's the only place we need anything like
        # this though.

        self.tls_contexts = {}
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
        self.tls_module = IRAmbassadorTLS(self, aconf)

        # Next, handle the "Ambassador" module.
        self.ambassador_module = IRAmbassador(self, aconf)

        # Save breaker & outlier configs.
        self.breakers = aconf.get_config("CircuitBreaker") or {}
        self.outliers = aconf.get_config("OutlierDetection") or {}

        # After the Ambassador and TLS modules are done, we need to set up the
        # filter chains, which requires checking in on the tracing, auth, and
        # ratelimit configuration stuff.
        #
        # ORDER MATTERS HERE.

        for cls in [ IRAuth, IRRateLimit ]:
            r = cls(self, aconf)

            # print("CHECKING FILTER %s (%s) %s" % (r, r.is_active(), repr(r)))

            if r.is_active():
                # print("SAVING FILTER %s" % r)
                self.filters.append(r)

        # Then append non-configurable cors and decoder filters
        self.filters.append(IRResource(ir=self, aconf=aconf, rkey="ir.cors", kind="ir.cors", name="IRCORS",
                                       config={}))
        self.filters.append(IRResource(ir=self, aconf=aconf, rkey="ir.router", kind="ir.router", name="IRRouter",
                                       type="decoder", config=self.router_config))

        # We would handle other modules here -- but guess what? There aren't any.
        # At this point ambassador, tls, and the deprecated auth module are all there
        # are, and they're handled above. So. At this point we can set up our listeners.
        ListenerFactory.load_all(self, aconf)

        # After listeners, handle mappings, clusters, etc.
        MappingFactory.load_all(self, aconf)

        # At this point we should know the full set of clusters, so we can normalize
        # any long cluster names.
        collisions: Dict[str, List[str]] = {}
        # mangled: Dict[str, str] = {}

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

                # mangled[name] = mangled_name
                self.clusters[name]['name'] = mangled_name

    def save_tls_context(self, ctx_name: str, ctx: IREnvoyTLS) -> bool:
        if ctx_name in self.tls_contexts:
            return False

        self.tls_contexts[ctx_name] = ctx
        return True

    def get_tls_context(self, ctx_name: str) -> Optional[IREnvoyTLS ]:
        return self.tls_contexts.get(ctx_name, None)

    def get_tls_defaults(self, ctx_name: str) -> Optional[Dict]:
        return self.tls_defaults.get(ctx_name, None)

    def add_listener(self, listener: IRListener) -> None:
        self.listeners.append(listener)

    def add_mapping(self, aconf: Config, mapping: IRMapping) -> None:
        if mapping.is_active():
            if mapping.group_id not in self.groups:
                group_name = "GROUP: %s" % mapping.name
                group = IRMappingGroup(ir=self, aconf=aconf,
                                       location=mapping.location,
                                       name=group_name,
                                       mapping=mapping)

                self.groups[group.group_id] = group
            else:
                self.groups[mapping.group_id].add_mapping(aconf, mapping)

    def has_cluster(self, rkey: str) -> bool:
        return rkey in self.clusters

    def get_cluster(self, rkey: str) -> Optional[IRCluster]:
        return self.clusters.get(rkey, None)

    def add_cluster_for_mapping(self, cluster: IRCluster, mapping: IRMapping) -> IRCluster:
        if not self.has_cluster(cluster.name):
            self.clusters[cluster.name] = cluster

        return self.clusters[cluster.name]

    def dump(self, output=sys.stdout):
        output.write("IR:\n")

        output.write("-- ambassador:\n")
        output.write("%s\n" % self.ambassador_module.as_json())

        output.write("-- tls_contexts:\n")

        for ctx_name in sorted(self.tls_contexts.keys()):
            output.write("%s: %s\n" % (ctx_name, self.tls_contexts[ctx_name].as_json()))

        output.write("-- listeners:\n")

        for listener in self.listeners:
            output.write("%s\n" % listener.as_json())

        output.write("-- filters:\n")

        for filter in self.filters:
            output.write("%s\n" % filter.as_json())

        output.write("-- groups:\n")

        for group in reversed(sorted(self.groups.values(), key=lambda x: x['group_weight'])):
            # output.write("==== %s\n" % group.group_id)
            # for k in sorted(group.keys()):
            #     output.write("     %s: %s\n" % (k, repr(group[k])))
            # output.flush()
            output.write("%s\n" % group.as_json())
            output.flush()
