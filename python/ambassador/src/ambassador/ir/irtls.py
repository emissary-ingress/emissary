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

from typing import TYPE_CHECKING

from ..config import Config
from .irresource import IRResource as IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


#############################################################################
## tls.py -- the tls_context configuration object for Ambassador
##
## IRAmbassadorTLS represents an Ambassador TLS configuration: it's the way
## we unify the TLS module and the 'tls' block in the Ambassador module. This
## class is pretty much all about managing priority between the two -- any
## important information here gets turned into IRTLSContext objects before
## TLS configuration actually happens.
##
## There's a fair amount of logic around making priority decisions between
## the 'tls' block and the TLS module at present. Probably that logic should
## migrate here, or this class should go away.


class IRAmbassadorTLS(IRResource):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.tlsmodule",
        kind: str = "IRTLSModule",
        name: str = "ir.tlsmodule",
        enabled: bool = True,
        **kwargs,
    ) -> None:
        """
        Initialize an IRAmbassadorTLS from the raw fields of its Resource.
        """

        ir.logger.debug("IRAmbassadorTLS __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, enabled=enabled, **kwargs
        )


class TLSModuleFactory:
    @classmethod
    def load_all(cls, ir: "IR", aconf: Config) -> None:
        assert ir

        tls_module = aconf.get_module("tls")

        if tls_module:
            # ir.logger.debug("TLSModuleFactory saving TLS module: %s" % tls_module.as_json())

            # XXX What a hack. IRAmbassadorTLS.from_resource() should be able to make
            # this painless.
            new_args = dict(tls_module.as_dict())
            new_rkey = new_args.pop("rkey", tls_module.rkey)
            new_kind = new_args.pop("kind", tls_module.kind)
            new_name = new_args.pop("name", tls_module.name)
            new_location = new_args.pop("location", tls_module.location)

            ir.tls_module = IRAmbassadorTLS(
                ir,
                aconf,
                rkey=new_rkey,
                kind=new_kind,
                name=new_name,
                location=new_location,
                **new_args,
            )

            ir.logger.debug("TLSModuleFactory saved TLS module: %s" % ir.tls_module.as_json())

        # Next, a TLS module in the Ambassador module overrides any other TLS Module.
        amod = aconf.get_module("ambassador")

        if amod:
            ir.ambassador_module.sourced_by(amod)
            ir.ambassador_module.referenced_by(amod)

            amod_tls = amod.get("tls", None)

            # Check for an Ambassador module tls field so that we can warn the user that this field is deprecated!
            if amod_tls:
                ir.post_error(
                    "The 'tls' field on the Ambassador module is deprecated! Please use a TLSContext instead https://www.getambassador.io/docs/edge-stack/latest/topics/running/tls/#tlscontext"
                )

        # Finally, if we have a TLS Module, turn it into a TLSContext.
        if ir.tls_module:
            ir.logger.debug("TLSModuleFactory translating TLS module to TLSContext")

            # Stash a sane rkey and location for contexts we create.
            ctx_rkey = ir.tls_module.get("rkey", ir.ambassador_module.rkey)
            ctx_location = ir.tls_module.get("location", ir.ambassador_module.location)

            # The TLS module 'server' and 'client' blocks are actually a _single_ TLSContext
            # to Ambassador.

            server = ir.tls_module.pop("server", None)
            client = ir.tls_module.pop("client", None)

            if server and server.get("enabled", True):
                # We have a server half. Excellent.

                ctx = IRTLSContext.from_legacy(
                    ir,
                    "server",
                    ctx_rkey,
                    ctx_location,
                    cert=server,
                    termination=True,
                    validation_ca=client,
                )

                if ctx.is_active():
                    ir.save_tls_context(ctx)

            # Other blocks in the TLS module weren't ever really documented, so I seriously doubt
            # that they're a factor... but, weirdly, we have a test for them...

            for legacy_name, legacy_ctx in ir.tls_module.as_dict().items():
                if (
                    legacy_name.startswith("_")
                    or (legacy_name == "name")
                    or (legacy_name == "namespace")
                    or (legacy_name == "metadata_labels")
                    or (legacy_name == "location")
                    or (legacy_name == "kind")
                    or (legacy_name == "enabled")
                ):
                    continue

                ctx = IRTLSContext.from_legacy(
                    ir,
                    legacy_name,
                    ctx_rkey,
                    ctx_location,
                    cert=legacy_ctx,
                    termination=False,
                    validation_ca=None,
                )

                if ctx.is_active():
                    ir.save_tls_context(ctx)

    @classmethod
    def finalize(cls, ir: "IR", aconf: Config) -> None:
        pass
