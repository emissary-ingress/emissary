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

                    implicit_tls_exists = ir.has_tls_context(ctx_name)
                    self.logger.debug(f"TLSContext with name {ctx_name} exists in the cluster?: {implicit_tls_exists}")

                    host_tls_context_obj = self.get('tlsContext', {})
                    host_tls_context_name = host_tls_context_obj.get('name', None)
                    self.logger.debug(f"Found TLSContext: {host_tls_context_name}")

                    host_tls_config = self.get('tls', None)
                    self.logger.debug(f"Found TLS config: {host_tls_config}")

                    # Choose explicit TLS configuration over implicit TLSContext name
                    if implicit_tls_exists and (host_tls_context_name or host_tls_config):
                        self.logger.info(f"Host {self.name}: even though TLSContext {ctx_name} exists in the cluster,"
                                         f"it will be ignored in favor of 'tls'/'tlsConfig' specified in the Host.")

                    # Even though this is unlikely because we have a oneOf is proto definitions, but just in case the
                    # objects have a different source :shrug:
                    if host_tls_context_name and host_tls_config:
                        self.post_error(f"Host {self.name}: both TLSContext name and TLS config specified, ignoring "
                                        f"Host...")
                        return False

                    if host_tls_context_name:
                        ir.logger.debug(f"Host {self.name}: found TLSContext name in config: {host_tls_context_name}")

                        if not ir.has_tls_context(host_tls_context_name):
                            self.post_error(f"Host {self.name}: Specified TLSContext does not exist: "
                                            f"{host_tls_context_name}")
                            return False

                        host_tls_context = ir.get_tls_context(host_tls_context_name)

                        # First make sure that the TLSContext is "compatible" i.e. it at least has the same cert related
                        # configuration as the one in this Host AND hosts are same as well.
                        if 'secret' in host_tls_context:
                            context_ss = self.resolve(ir, host_tls_context.get('secret'))
                            if str(context_ss) != str(tls_ss):
                                self.post_error(f"Secret info mismatch between Host: {self.name} (secret: {tls_name})"
                                                f"and TLSContext: {host_tls_context_name}"
                                                f"(secret: {host_tls_context.get('secret')})")
                                return False
                        else:
                            host_tls_context['secret'] = tls_name

                        if 'hosts' in host_tls_context:
                            is_valid_hosts = False
                            for host_tc in host_tls_context.get('hosts'):
                                if host_tc in [self.hostname, self.name]:
                                    is_valid_hosts = True
                            if not is_valid_hosts:
                                self.post_error(f"Hosts mismatch between Host: {self.name} "
                                                f"(accepted hosts: {[self.hostname, self.name]}) and "
                                                f"TLSContext {host_tls_context_name} "
                                                f"(hosts: {host_tls_context.get('hosts')})")
                        else:
                            host_tls_context['hosts'] = [self.hostname or self.name]

                        # All seems good, this context belongs to self now!
                        self.context = host_tls_context

                    elif host_tls_config:
                        ir.logger.debug(f"Host {self.name}: found tlsConfig {host_tls_config}")

                        camel_snake_map = {
                            'alpnProtocols': 'alpn_protocols',
                            'cipherSuites': 'cipher_suites',
                            'ecdhCurves': 'ecdh_curves',
                            'redirectCleartextFrom': 'redirect_cleartext_from',
                            'certRequired': 'cert_required',
                            'minTlsVersion': 'min_tls_version',
                            'maxTlsVersion': 'max_tls_version',
                            'certChainFile': 'cert_chain_file',
                            'privateKeyFile': 'private_key_file',
                            'cacertChainFile': 'cacert_chain_file',
                            'caSecret': 'ca_secret',
                            # 'sni': 'sni' (this field is not required in snake-camel but adding for completeness)
                        }

                        # We don't need any camel case in our generated TLSContext
                        for camel, snake in camel_snake_map.items():
                            if camel in host_tls_config:
                                # We use .pop() to actually replace the camelCase name with snake case
                                host_tls_config[snake] = host_tls_config.pop(camel)

                        if 'min_tls_version' in host_tls_config:
                            if host_tls_config['min_tls_version'] not in IRTLSContext.AllowedTLSVersions:
                                self.post_error(f"Host {self.name}: Invalid min_tls_version set in Host.tls: "
                                                f"{host_tls_config['min_tls_version']}")
                                return False

                        if 'max_tls_version' in host_tls_config:
                            if host_tls_config['max_tls_version'] not in IRTLSContext.AllowedTLSVersions:
                                self.post_error(f"Host {self.name}: Invalid max_tls_version set in Host.tls: "
                                                f"{host_tls_config['max_tls_version']}")
                                return False

                        tls_context_init = dict(
                            rkey=self.rkey,
                            name=ctx_name,
                            namespace=self.namespace,
                            location=self.location,
                            hosts=[self.hostname or self.name],
                            secret=tls_name,
                        )

                        tls_config_context = IRTLSContext(ir, aconf, **tls_context_init, **host_tls_config)

                        match_labels = self.get('matchLabels')
                        if not match_labels:
                            match_labels = self.get('match_labels')
                        if match_labels:
                            tls_config_context['metadata_labels'] = match_labels

                        if tls_config_context.is_active():
                            self.context = tls_config_context
                            tls_config_context.referenced_by(self)
                            tls_config_context.sourced_by(self)

                            ir.save_tls_context(tls_config_context)
                        else:
                            self.post_error(f"Host {self.name}: generated TLSContext {tls_config_context.name} from "
                                            f"Host.tls is not valid")
                            return False

                    elif implicit_tls_exists:
                        ir.logger.debug(f"Host {self.name}: TLSContext {ctx_name} already exists")

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
        # We override namespace below if there is a '.' in the secret_name
        # and secret_namespacing is true.
        namespace = self.namespace or ir.ambassador_namespace

        secret_namespacing = self.lookup('secret_namespacing', True,
                                         default_key='tls_secret_namespacing')

        if "." in secret_name and secret_namespacing:
            secret_name, namespace = secret_name.split('.', 1)

        self.ir.logger.debug(f"TLSContext.resolve_secret {secret_name}, namespace {namespace}: namespacing is {secret_namespacing}")

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
