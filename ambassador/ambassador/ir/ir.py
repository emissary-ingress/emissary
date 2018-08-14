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

import logging
import os

from ..utils import TLSPaths
from ..mapping import Mapping
from ..config import Config
from ..resource import Resource

from .irresource import IRResource
from .irambassador import IRAmbassador
from .irauth import IRAuth
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
    listeners: List[IRListener ]

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
        self.routes = {}
        self.grpc_services = []
        self.router_config = {}
        self.filters = []
        self.tracing_config = None
        self.listeners = []

        # Set up default TLS stuff.
        #
        # XXX This feels like a hack -- shouldn't it be class-wide initialization
        # in TLSModule or TLSContext? So far it's the only place we need anything like
        # this though.

        self.tls_contexts = {}
        self.tls_defaults: Dict[str, Dict[str, str]] = {
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

            print("CHECKING FILTER %s (%s) %s" % (r, r.is_active(), repr(r)))

            if r.is_active():
                print("SAVING FILTER %s" % r)
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

        #  it's on to Mappings.
        mappings = aconf.get_config("mappings") or {}

    def get_module(self, module_name: str) -> Optional[Resource]:
        """
        Fetch a module from the module store. Can return None if no
        such module exists.

        :param module_name: name of the module you want.
        """

        return self.modules.get(module_name, None)

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
