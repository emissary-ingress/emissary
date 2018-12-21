from typing import ClassVar, Dict, List, Optional, TYPE_CHECKING

import json

from ..config import Config

from .irresource import IRResource
from .irmapping import IRMapping
from .irtls import IREnvoyTLS, IRAmbassadorTLS
from .irtlscontext import IRTLSContext
from .ircors import IRCORS
from .irbuffer import IRBuffer

if TYPE_CHECKING:
    from .ir import IR


class IRAmbassador (IRResource):
    AModTransparentKeys: ClassVar = [
        'admin_port',
        'auth_enabled',
        'default_label_domain',
        'default_labels',
        'diag_port',
        'diagnostics',
        'liveness_probe',
        'readiness_probe',
        'service_port',
        'statsd',
        'use_proxy_proto',
        'use_remote_address',
        'x_forwarded_proto_redirect'
    ]

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

                # ir.logger.debug("IRAmbassador saving TLS module: %s" % ir.tls_module.as_json())

        if ir.tls_module:
            self.logger.debug("final TLS module: %s" % ir.tls_module.as_json())

            # The TLS module 'server' and 'client' blocks are actually a _single_ TLSContext
            # to Ambassador.

            server = ir.tls_module.pop('server', None)
            client = ir.tls_module.pop('client', None)

            if server or client:
                ctx_name = 'legacy-server' if server else 'legacy-client'
                ctx_rkey = ir.tls_module.get('rkey', self.rkey)
                ctx_location = ir.tls_module.get('location', self.location)

                new_args = {
                    'hosts': ['*']
                }

                if server and server.get('enabled', True):
                    if 'secret' in server:
                        new_args['secret'] = server['secret']

                    if 'cert_chain_file' in server:
                        new_args['cert_chain_file'] = server['cert_chain_file']

                    if 'private_key_file' in server:
                        new_args['private_key_file'] = server['private_key_file']

                    if 'alpn_protocols' in server:
                        new_args['alpn_protocols'] = server['alpn_protocols']

                    if 'redirect_cleartext_from' in server:
                        new_args['redirect_cleartext_from'] = server['redirect_cleartext_from']

                    if (('secret' not in new_args) and
                        ('cert_chain_file' not in new_args) and
                        ('private_key_file' not in new_args)):
                        # Assume they want the 'ambassador-certs' secret.
                        new_args['secret'] = 'ambassador-certs'

                if client and client.get('enabled', True):
                    if 'secret' in client:
                        new_args['ca_secret'] = client['secret']

                    if 'cacert_chain_file' in client:
                        new_args['cacert_chain_file'] = client['cacert_chain_file']

                    if 'cert_required' in client:
                        new_args['cert_required'] = client['cert_required']

                    if (('ca_secret' not in new_args) and
                        ('cacert_chain_file' not in new_args)):
                        # Assume they want the 'ambassador-cacert' secret.
                        new_args['secret'] = 'ambassador-cacert'

                ctx = IRTLSContext.fromConfig(ir, ctx_rkey, ctx_location,
                                              kind="synthesized-TLS-context",
                                              name=ctx_name, **new_args)

                if ctx.is_active():
                    ir.tls_contexts.append(ctx)

            # We're going to call other blocks in the TLS module errors. They weren't ever really
            # documented, so I seriously doubt that they're a factor.

            any_errors = False

            for ctx_name, ctx in ir.tls_module.as_dict().items():
                if (ctx_name.startswith('_') or
                    (ctx_name == 'name') or
                    (ctx_name == 'location') or
                    (ctx_name == 'kind') or
                    (ctx_name == 'enabled')):
                    continue

                if not any_errors:
                    ir.post_error("The TLS Module (see %s) no longer supports arbitrary contexts." % ir.tls_module.location,
                                  ir.tls_module)

                ir.post_error("Use a TLSContext for the %s block in your TLS module" % ctx_name, ir.tls_module)
                any_errors = True

                # if isinstance(ctx, dict):
                #     ctxkey = ir.tls_module.get('rkey', self.rkey)
                #     ctxloc = ir.tls_module.get('location', self.location)
                #
                #     etls = IREnvoyTLS(ir=ir, rkey=ctxkey, aconf=aconf, name=ctx_name,
                #                       location=ctxloc, **ctx)
                #
                #     if ir.save_envoy_tls_context(ctx_name, etls):
                #         self.logger.debug("context %s: created from %s" % (ctx_name, ctxloc))
                #         # self.logger.debug(etls.as_json())
                #     else:
                #         self.logger.debug("context %s: not updating from %s" % (ctx_name, ctxloc))
                #         # self.logger.debug(etls.as_json())
                #
                #     if etls.get('valid_tls') and ctx_name == 'server':
                #         self.logger.debug("TLS termination enabled!")
                #         self.service_port = 443

        # Finally, check TLSContext resources to see if we should enable TLS termination.
        for ctx in ir.tls_contexts:
            if ctx.get('hosts', None):
                # This is a termination context
                self.logger.debug("TLSContext %s is a termination context, enabling TLS termination" % ctx.name)
                self.service_port = 443

                if ctx.get('ca_cert', None):
                    # Client-side TLS is enabled.
                    self.logger.debug("TLSContext %s enables client certs!" % ctx.name)

        # After that, check for port definitions, probes, etc., and copy them in
        # as we find them.
        for key in IRAmbassador.AModTransparentKeys:
            if amod and (key in amod):
                # Yes. It overrides the default.
                self[key] = amod[key]

        # If we don't have a default label domain, force it to 'ambassador'.
        if not self.get('default_label_domain'):
            self.default_label_domain = 'ambassador'

        # Likewise, if we have no default labels, force an empty dict (it makes life easier
        # on other modules).
        if not self.get('default_labels'):
            self.default_labels = {}

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

         # Buffer.
        if amod and ('buffer' in amod):            
            self.buffer = IRBuffer(ir=ir, aconf=aconf, location=self.location, **amod.buffer)

            if self.buffer:
                ir.save_filter(self.buffer)
            else:
                return False

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

    def get_default_label_domain(self) -> str:
        return self.default_label_domain

    def get_default_labels(self, domain: Optional[str]=None) -> Optional[List]:
        if not domain:
            domain = self.get_default_label_domain()

        domain_info = self.default_labels.get(domain, {})

        self.logger.debug("default_labels info for %s: %s" % (domain, domain_info))

        return domain_info.get('defaults')

    def get_default_label_prefix(self, domain: Optional[str]=None) -> Optional[List]:
        if not domain:
            domain = self.get_default_label_domain()

        domain_info = self.default_labels.get(domain, {})
        return domain_info.get('label_prefix')

