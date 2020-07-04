import copy
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
        'tlsContext',
        'tls',
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
        ir.logger.debug(f"Host {self.name} setting up")

        tls_ss: Optional[SavedSecret] = None
        pkey_ss: Optional[SavedSecret] = None

        if self.get('tlsSecret', None):
            tls_secret = self.tlsSecret
            tls_name = tls_secret.get('name', None)

            if tls_name:
                ir.logger.debug(f"Host {self.name}: TLS secret name is {tls_name}")

                tls_ss = self.resolve(ir, tls_name)

                if tls_ss:
                    # OK, we have a TLS secret! Fire up a TLS context for it, if one doesn't
                    # already exist.

                    ctx_name = f"{self.name}-context"

                    host_tls_context_name = self.get('tlsContext', None)
                    host_tls_config = self.get('tls', None)

                    self.logger.info(f"Found TLSContext: {host_tls_context_name}")
                    self.logger.info(f"Found TLS config: {host_tls_config}")

                    if host_tls_context_name and host_tls_config:
                        ir.logger.info(f"Host {self.name}: both TLSContext name and TLS config specified, ignoring...")
                        ir.logger.error(f"Host {self.name}: both TLSContext name and TLS config specified, ignoring...")
                        return False

                    if ir.has_tls_context(ctx_name):
                        ir.logger.debug(f"Host {self.name}: TLSContext {ctx_name} already exists")
                    elif host_tls_context_name:
                        if not ir.has_tls_context(host_tls_context_name):
                            ir.logger.info(f"Host {self.name}: invalid TLSContext specified {host_tls_context_name}")
                            ir.logger.error(f"Host {self.name}: invalid TLSContext specified {host_tls_context_name}")
                            return False

                        ir.logger.info(f"Found TLSContext {host_tls_context_name} as specified")

                        host_tls_context = ir.get_tls_context(host_tls_context_name)

                        new_ctx = copy.deepcopy(host_tls_context)

                        # We don't need a duplicate TLSContext
                        host_tls_context.set_active(False)
                        new_ctx.set_active(True)

                        new_ctx.rkey = self.rkey
                        new_ctx.name = ctx_name
                        new_ctx.namespace = self.namespace
                        new_ctx.location = self.location
                        new_ctx.hosts = [self.hostname or self.name]
                        new_ctx.secret = tls_name

                        match_labels = self.get('matchLabels')
                        if not match_labels:
                            match_labels = self.get('match_labels')
                        if match_labels:
                            new_ctx['metadata_labels'] = match_labels

                        self.context = new_ctx
                        new_ctx.referenced_by(self)
                        new_ctx.sourced_by(self)

                        ir.logger.info(f"Created new TLSContext: {new_ctx}")
                        ir.resolve_secret(self, tls_name, self.namespace)
                        ir.save_tls_context(new_ctx)

                    elif host_tls_config:
                        ir.logger.info(f"Host {self.name}: creating TLSContext {ctx_name}")

                        new_ctx = dict(
                            rkey=self.rkey,
                            name=ctx_name,
                            namespace=self.namespace,
                            location=self.location,
                            hosts=[self.hostname or self.name],
                            secret=tls_name,
                            alpn_protocols=host_tls_config.get('alpn_protocols') or host_tls_config.get('AlpnProtocols'),
                            cipher_suites=host_tls_config.get('cipher_suites') or host_tls_config.get('CipherSuites'),
                            ecdh_curves=host_tls_config.get('ecdh_curves') or host_tls_config.get('EcdhCurves'),
                            redirect_cleartext_from=host_tls_config.get('redirect_cleartext_from') or host_tls_config.get('RedirectCleartextFrom'),
                            sni=host_tls_config.get('sni') or host_tls_config.get('Sni'),
                            cert_required=host_tls_config.get('cert_required') or host_tls_config.get('CertRequired')
                        )

                        host_min_tls_version = host_tls_config.get('min_tls_version') or host_tls_config.get('MinTlsVersion')
                        if host_min_tls_version:
                            if host_min_tls_version in IRTLSContext.AllowedTLSVersions:
                                new_ctx['min_tls_version'] = host_min_tls_version
                            else:
                                self.post_error(f"Invalid min_tls_version set in Host.tls: {host_min_tls_version}")
                                return False

                        host_max_tls_version = host_tls_config.get('max_tls_version') or host_tls_config.get('MaxTlsVersion')
                        if host_max_tls_version:
                            if host_max_tls_version in IRTLSContext.AllowedTLSVersions:
                                new_ctx['max_tls_version'] = host_max_tls_version
                            else:
                                self.post_error(f"Invalid max_tls_version set in Host.tls: {host_max_tls_version}")
                                return False


                        host_cert_chain_file = host_tls_config.get('cert_chain_file') or host_tls_config.get('CertChainFile')
                        if host_cert_chain_file:
                            new_ctx['cert_chain_file'] = host_cert_chain_file

                        host_private_key_file = host_tls_config.get('private_key_file') or host_tls_config.get('PrivateKeyFile')
                        if host_private_key_file:
                            new_ctx['private_key_file'] = host_private_key_file

                        host_cacert_chain_file = host_tls_config.get('cacert_chain_file') or host_tls_config.get('CacertChainFile')
                        if host_cacert_chain_file:
                            new_ctx['cacert_chain_file'] = host_cacert_chain_file

                        host_ca_secret = host_tls_config.get('ca_secret') or host_tls_config.get('CaSecret')
                        if host_ca_secret:
                            new_ctx['ca_secret'] = host_ca_secret

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

                            ir.logger.info(f"w00t: Create internal TLSContext: {ctx.as_json()}")
                            ir.save_tls_context(ctx)
                        else:
                            ir.logger.info(f"Host {self.name}: new TLSContext {ctx_name} is not valid")
                            ir.logger.error(f"Host {self.name}: new TLSContext {ctx_name} is not valid")
                    else:
                        ir.logger.debug(f"Host {self.name}: creating TLSContext {ctx_name}")

                        new_ctx = dict(
                            rkey=self.rkey,
                            name=ctx_name,
                            namespace=self.namespace,
                            location=self.location,
                            hosts=[ self.hostname or self.name ],
                            secret=tls_name
                        )

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

            # The ACME client is disabled if we're running as an intercept agent.
            if ir.edge_stack_allowed and not ir.agent_active:
                authority = acme.get('authority', None)

                if authority and (authority.lower() != 'none'):
                    # ACME is active. Are they trying to not set insecure.additionalPort?
                    request_policy = self.get('requestPolicy', {})
                    insecure_policy = request_policy.get('insecure', {})

                    # Default the additionalPort to 8080. This can be overridden by the user
                    # explicitly setting it to -1.
                    insecure_addl_port = insecure_policy.get('additionalPort', 8080)

                    if insecure_addl_port < 0:
                        # Bzzzt.
                        self.post_error("ACME requires insecure.additionalPort to function; forcing to 8080")
                        insecure_policy['additionalPort'] = 8080

                        if 'action' not in insecure_policy:
                            # No action when we're overriding the additionalPort already means that we
                            # default the action to Reject (the hole-puncher will do the right thing).
                            insecure_policy['action'] = 'Reject'

                        request_policy['insecure'] = insecure_policy
                        self['requestPolicy'] = request_policy

            pkey_secret = acme.get('privateKeySecret', None)

            if pkey_secret:
                pkey_name = pkey_secret.get('name', None)

                if pkey_name:
                    ir.logger.debug(f"Host {self.name}: ACME private key name is {pkey_name}")

                    pkey_ss = self.resolve(ir, pkey_name)

                    if not pkey_ss:
                        ir.logger.error(f"Host {self.name}: continuing with invalid private key secret {pkey_name}")

        ir.logger.debug(f"Host setup OK: {self.pretty()}")
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

                    ir.logger.debug(f"HostFactory: saving host {host.pretty()}")
                    ir.save_host(host)
                else:
                    ir.logger.debug(f"HostFactory: not saving inactive host {host.pretty()}")

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        if ir.edge_stack_allowed:
            # We're running Edge Stack. Figure out how many hosts we have, and whether
            # we have any termination contexts.
            #
            # If we're running as an intercept agent, there should be a Host in all cases.
            host_count = len(ir.get_hosts() or [])
            contexts = ir.get_tls_contexts() or []

            found_termination_context = False
            for ctx in contexts:
                if ctx.get('hosts'):  # not None and not the empty list
                    found_termination_context = True

            ir.logger.debug(f"HostFactory: FTC {found_termination_context}, host_count {host_count}")

            if (host_count == 0) and not found_termination_context:
                # We have no Hosts and no termination contexts, so we know that this is an unconfigured
                # installation. Set up the fallback TLSContext so we can redirect people to the UI.
                ir.logger.debug("Creating fallback context")
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
