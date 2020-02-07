from typing import Optional, TYPE_CHECKING

import os

from ..utils import SavedSecret
from ..config import Config
from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR


class IRHost(IRResource):
    AllowedKeys = {
        'acmeProvider',
        'hostname',
        'matchLabels',
        'requestPolicy',
        'selector',
        'tlsSecret',
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED
                 namespace: Optional[str]=None,
                 kind: str="IRHost",
                 apiVersion: str="getambassador.io/v2",   # Not a typo! See below.
                 **kwargs) -> None:

        new_args = {
            x: kwargs[x] for x in kwargs.keys()
            if x in IRHost.AllowedKeys
        }

        self.context: Optional[IRTLSContext] = None

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, apiVersion=apiVersion,
            **new_args
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        ir.logger.info(f"Host {self.name} setting up")

        tls_ss: Optional[SavedSecret] = None
        pkey_ss: Optional[SavedSecret] = None

        if self.get('tlsSecret', None):
            tls_secret = self.tlsSecret
            tls_name = tls_secret.get('name', None)

            if tls_name:
                ir.logger.info(f"Host {self.name}: TLS secret name is {tls_name}")

                tls_ss = self.resolve(ir, tls_name)

                if tls_ss:
                    # OK, we have a TLS secret! Fire up a TLS context for it, if one doesn't
                    # already exist.

                    ctx_name = f"{self.name}-context"

                    if ir.has_tls_context(ctx_name):
                        ir.logger.info(f"Host {self.name}: TLSContext {ctx_name} already exists")
                    else:
                        ir.logger.info(f"Host {self.name}: creating TLSContext {ctx_name}")

                        new_ctx = dict(
                            rkey=self.rkey,
                            name=ctx_name,
                            namespace=self.namespace,
                            location=self.location,
                            hosts=[ self.hostname or self.name ],
                            secret=tls_name
                        )

                        # if not os.environ.get('AMBASSADOR_NO_TLS_REDIRECT', None):
                        #     new_ctx['redirect_cleartext_from'] = 8080

                        ctx = IRTLSContext(ir, aconf, **new_ctx)

                        match_labels = self.get('matchLabels')

                        if not match_labels:
                            match_labels = self.get('match_labels')

                        if match_labels:
                            ctx['metadata_labels'] = match_labels

                        if ctx.is_active():
                            self.context = ctx
                            ctx.referenced_by(self)
                            ctx.sourced_by(self)

                            ir.save_tls_context(ctx)
                        else:
                            ir.logger.error(f"Host {self.name}: new TLSContext {ctx_name} is not valid")
                else:
                    ir.logger.error(f"Host {self.name}: continuing with invalid TLS secret {tls_name}")
                    return False

        if self.get('acmeProvider', None):
            acme = self.acmeProvider
            pkey_secret = acme.get('privateKeySecret', None)

            if pkey_secret:
                pkey_name = pkey_secret.get('name', None)

                if pkey_name:
                    ir.logger.info(f"Host {self.name}: ACME private key name is {pkey_name}")

                    pkey_ss = self.resolve(ir, pkey_name)

                    if not pkey_ss:
                        ir.logger.error(f"Host {self.name}: continuing with invalid private key secret {pkey_name}")

        ir.logger.info(f"Host setup OK: {self.pretty()}")
        return True

    def pretty(self) -> str:
        request_policy = self.get('requestPolicy', {})
        insecure_policy = request_policy.get('insecure', {})
        insecure_action = insecure_policy.get('action', 'Redirect')
        insecure_addl_port = insecure_policy.get('additionalPort', None)

        ctx_name = self.context.name if self.context else "-none-"
        return "<Host %s for %s ctx %s ia %s iap %s>" % (self.name, self.hostname or '*', ctx_name,
                                                         insecure_action, insecure_addl_port)

    def resolve(self, ir: 'IR', secret_name: str) -> SavedSecret:
        # Try to use our namespace for secret resolution. If we somehow have no
        # namespace, fall back to the Ambassador's namespace.
        namespace = self.namespace or ir.ambassador_namespace

        return ir.resolve_secret(self, secret_name, namespace)


class HostFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        assert ir

        hosts = aconf.get_config('hosts')

        if hosts:
            for config in hosts.values():
                ir.logger.debug("HostFactory: creating host for %s" % repr(config.as_dict()))

                host = IRHost(ir, aconf, **config)

                if host.is_active():
                    host.referenced_by(config)
                    host.sourced_by(config)

                    ir.logger.info(f"HostFactory: saving host {host.pretty()}")
                    ir.save_host(host)
                else:
                    ir.logger.info(f"HostFactory: not saving inactive host {host.pretty()}")

        if ir.edge_stack_allowed:
            # We're running Edge Stack. Figure out how many hosts we have, and whether
            # we have any termination contexts.
            host_count = len(ir.get_hosts() or [])
            contexts = ir.get_tls_contexts() or []

            found_termination_context = False
            for ctx in contexts:
                if ctx.get('hosts'):  # not None and not the empty list
                    found_termination_context = True

            ir.logger.info(f"HostFactory: FTC {found_termination_context}, host_count {host_count}")

            if (host_count == 0) and not found_termination_context:
                # We have no Hosts and no termination contexts, so we know that this is an unconfigured
                # installation. Set up the fallback TLSContext so we can redirect people to the UI.
                ir.logger.info("Creating fallback context")
                ctx_name = "fallback-self-signed-context"
                tls_name = "fallback-self-signed-cert"

                new_ctx = dict(
                    rkey=f"{ctx_name}.99999",
                    name=ctx_name,
                    location="-internal-",
                    hosts=["*"],
                    secret=tls_name,
                    is_fallback=True
                )

                if not os.environ.get('AMBASSADOR_NO_TLS_REDIRECT', None):
                    new_ctx['redirect_cleartext_from'] = 8080

                ctx = IRTLSContext(ir, aconf, **new_ctx)

                assert ctx.is_active()
                if ctx.resolve_secret(tls_name):
                    ir.save_tls_context(ctx)
