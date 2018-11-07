from typing import ClassVar, Dict, Optional, TYPE_CHECKING

import json

from ..config import Config

from .irresource import IRResource
from .irmapping import IRMapping
from .irtls import IREnvoyTLS, IRAmbassadorTLS
from .ircors import IRCORS

if TYPE_CHECKING:
    from .ir import IR


class IRAmbassador (IRResource):
    service_port: int
    diag_port: int

    # Set up the default probes and such.
    default_liveness_probe: ClassVar[Dict[str, str]] = {
        "prefix": "/ambassador/v0/check_alive",
        "rewrite": "/ambassador/v0/check_alive",
    }

    default_readiness_probe: ClassVar[Dict[str, str]] = {
        "prefix": "/ambassador/v0/check_ready",
        "rewrite": "/ambassador/v0/check_ready",
    }

    default_diagnostics: ClassVar[Dict[str, str]] = {
        "prefix": "/ambassador/v0/",
        "rewrite": "/ambassador/v0/",
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.ambassador",
                 kind: str="IRAmbassador",
                 name: str="ir.ambassador",
                 use_remote_address: bool=True,
                 **kwargs) -> None:
        # print("IRAmbassador __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            service_port=80,
            admin_port=8001,
            diag_port=8877,
            auth_enabled=None,
            liveness_probe={"enabled": True},
            readiness_probe={"enabled": True},
            diagnostics={"enabled": True},
            use_proxy_proto=False,
            use_remote_address=use_remote_address,
            x_forwarded_proto_redirect=False,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # We're interested in the 'ambassador' module from the Config, if any...
        amod = aconf.get_module("ambassador")

        # Is there a TLS module in the Ambassador module?
        if amod:
            self.sourced_by(amod)
            self.referenced_by(amod)

            amod_tls = amod.get('tls', None)

            if amod_tls:
                # ir.logger.debug("IRAmbassador saving TLS module: %s" %
                #                 json.dumps(amod_tls, sort_keys=True, indent=4))

                # XXX What a hack. IRAmbassadorTLS.from_resource() should be able to make
                # this painless.
                new_args = dict(amod_tls)
                new_rkey = new_args.pop('rkey', amod.rkey)
                new_kind = new_args.pop('kind', 'Module')
                new_name = new_args.pop('name', 'tls-from-ambassador-module')
                new_location = new_args.pop('location', amod.location)

                # Overwrite any existing TLS module.
                ir.tls_module = IRAmbassadorTLS(ir, aconf,
                                                rkey=new_rkey,
                                                kind=new_kind,
                                                name=new_name,
                                                location=new_location,
                                                **new_args)

        if ir.tls_module:
            self.logger.debug("final TLS module: %s" %
                              json.dumps(ir.tls_module.as_dict(), sort_keys=True, indent=4))

            # Create TLS contexts.
            for ctx_name, ctx in ir.tls_module.items():
                if ctx_name.startswith('_'):
                    continue

                if isinstance(ctx, dict):
                    ctxloc = ir.tls_module.get('location', self.location)
                    etls = IREnvoyTLS(ir=ir, aconf=aconf, name=ctx_name,
                                      location=ctxloc, **ctx)

                    if ir.save_envoy_tls_context(ctx_name, etls):
                        self.logger.debug("created context %s from %s" % (ctx_name, ctxloc))
                        # self.logger.debug(etls.as_json())
                    else:
                        self.logger.debug("not updating context %s from %s" % (ctx_name, ctxloc))
                        # self.logger.debug(etls.as_json())

                    if etls.get('valid_tls'):
                        self.logger.debug("TLS termination enabled!")
                        self.service_port = 443

        ctx = ir.get_envoy_tls_context('client')

        if ctx:
            # Client-side TLS is enabled.
            self.logger.debug("TLS client certs enabled!")

        # After that, check for port definitions, probes, etc., and copy them in
        # as we find them.
        for key in [ 'service_port', 'admin_port', 'diag_port',
                     'liveness_probe', 'readiness_probe', 'auth_enabled',
                     'use_proxy_proto', 'use_remote_address', 'diagnostics',
                     'x_forwarded_proto_redirect' ]:
            if amod and (key in amod):
                # Yes. It overrides the default.
                self[key] = amod[key]

        # Do they want statsd?
        if amod and ('statsd' in amod):
            self['statsd'] = amod['statsd']

        # Next up: diag port & services.
        diag_port = aconf.module_lookup('ambassador', 'diag_port', 8877)
        diag_service = "127.0.0.1:%d" % diag_port

        for name, cur, dflt in [
            ("liveness",    self.liveness_probe,  IRAmbassador.default_liveness_probe),
            ("readiness",   self.readiness_probe, IRAmbassador.default_readiness_probe),
            ("diagnostics", self.diagnostics,     IRAmbassador.default_diagnostics)
        ]:
            if cur and cur.get("enabled", False):
                if not cur.get('prefix', None):
                    cur['prefix'] = dflt['prefix']

                if not cur.get('rewrite', None):
                    cur['rewrite'] = dflt['rewrite']

                if not cur.get('service', None):
                    cur['service'] = diag_service

        # Finally, default CORS stuff.
        if amod and ('cors' in amod):
            self.cors = IRCORS(ir=ir, aconf=aconf, location=self.location, **amod.cors)

            if self.cors:
                self.cors.referenced_by(self)
            else:
                return False

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        for name, cur in [
            ( "liveness",    self.liveness_probe ),
            ( "readiness",   self.readiness_probe ),
            ( "diagnostics", self.diagnostics )
        ]:
            if cur and cur.get("enabled", False):
                name = "internal_%s_probe_mapping" % name

                mapping = IRMapping(ir, aconf, rkey=self.rkey, name=name, location=self.location, **cur)
                mapping.referenced_by(self)
                ir.add_mapping(aconf, mapping)
