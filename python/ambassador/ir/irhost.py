import copy
import os
from typing import TYPE_CHECKING, Dict, List, Optional

from ..config import Config
from ..utils import SavedSecret, dump_json
from .irresource import IRResource
from .irtlscontext import IRTLSContext
from .irutils import hostglob_matches, selector_matches

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover
    from .irhttpmappinggroup import IRHTTPMappingGroup


class IRHost(IRResource):
    AllowedKeys = {
        "acmeProvider",
        "hostname",
        "mappingSelector",
        "metadata_labels",
        "requestPolicy",
        "selector",
        "tlsSecret",
        "tlsContext",
        "tls",
    }

    hostname: str
    secure_action: str
    insecure_action: str
    insecure_addl_port: Optional[int]

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str,  # REQUIRED
        name: str,  # REQUIRED
        location: str,  # REQUIRED
        namespace: Optional[str] = None,
        kind: str = "IRHost",
        apiVersion: str = "getambassador.io/v3alpha1",  # Not a typo! See below.
        **kwargs,
    ) -> None:

        new_args = {x: kwargs[x] for x in kwargs.keys() if x in IRHost.AllowedKeys}

        self.context: Optional[IRTLSContext] = None

        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            location=location,
            kind=kind,
            name=name,
            namespace=namespace,
            apiVersion=apiVersion,
            **new_args,
        )

    def setup(self, ir: "IR", aconf: Config) -> bool:
        ir.logger.debug(f"Host {self.name} setting up")

        if not self.get("hostname", None):
            self.hostname = "*"

        tls_ss: Optional[SavedSecret] = None
        pkey_ss: Optional[SavedSecret] = None

        # Go ahead and cache some things to make life easier later.
        request_policy = self.get("requestPolicy", {})

        # XXX This will change later!!
        self.secure_action = "Route"

        insecure_policy = request_policy.get("insecure", {})
        self.insecure_action = insecure_policy.get("action", "Redirect")
        self.insecure_addl_port: Optional[int] = insecure_policy.get("additionalPort", None)

        # If we have no mappingSelector, check for selector.
        mapsel = self.get("mappingSelector", None)

        if not mapsel:
            mapsel = self.get("selector", None)

            if mapsel:
                self.mappingSelector = mapsel
                del self["selector"]

        if self.get("tlsSecret", None):
            tls_secret = self.tlsSecret
            tls_name = tls_secret.get("name", None)

            if tls_name:
                ir.logger.debug(f"Host {self.name}: resolving spec.tlsSecret.name: {tls_name}")

                tls_ss = self.resolve(ir, tls_name)

                if tls_ss:
                    # OK, we have a TLS secret! Fire up a TLS context for it, if one doesn't
                    # already exist.

                    ctx_name = f"{self.name}-context"

                    implicit_tls_exists = ir.has_tls_context(ctx_name)
                    self.logger.debug(
                        f"Host {self.name}: implicit TLSContext {ctx_name} {'exists' if implicit_tls_exists else 'missing'}"
                    )

                    host_tls_context_obj = self.get("tlsContext", {})
                    host_tls_context_name = host_tls_context_obj.get("name", None)
                    self.logger.debug(f"Host {self.name}: spec.tlsContext: {host_tls_context_name}")

                    host_tls_config = self.get("tls", None)
                    self.logger.debug(f"Host {self.name}: spec.tls: {host_tls_config}")

                    # Choose explicit TLS configuration over implicit TLSContext name
                    if implicit_tls_exists and (host_tls_context_name or host_tls_config):
                        self.logger.info(
                            f"Host {self.name}: even though TLSContext {ctx_name} exists in the cluster,"
                            f"it will be ignored in favor of 'tls'/'tlsConfig' specified in the Host."
                        )

                    # Even though this is unlikely because we have a oneOf is proto definitions, but just in case the
                    # objects have a different source :shrug:
                    if host_tls_context_name and host_tls_config:
                        self.post_error(
                            f"Host {self.name}: both TLSContext name and TLS config specified, ignoring "
                            f"Host..."
                        )
                        return False

                    if host_tls_context_name:
                        # They named a TLSContext, so try to use that. self.save_context will check the
                        # context to make sure it works for us, and save it if so.
                        ir.logger.debug(
                            f"Host {self.name}: resolving spec.tlsContext: {host_tls_context_name}"
                        )

                        if not self.save_context(ir, host_tls_context_name, tls_ss, tls_name):
                            return False

                    elif host_tls_config:
                        # They defined a TLSContext inline, so go set that up if we can.
                        ir.logger.debug(f"Host {self.name}: examining spec.tls {host_tls_config}")

                        camel_snake_map = {
                            "alpnProtocols": "alpn_protocols",
                            "cipherSuites": "cipher_suites",
                            "ecdhCurves": "ecdh_curves",
                            "redirectCleartextFrom": "redirect_cleartext_from",
                            "certRequired": "cert_required",
                            "minTlsVersion": "min_tls_version",
                            "maxTlsVersion": "max_tls_version",
                            "certChainFile": "cert_chain_file",
                            "privateKeyFile": "private_key_file",
                            "cacertChainFile": "cacert_chain_file",
                            "crlSecret": "crl_secret",
                            "crlFile": "crl_file",
                            "caSecret": "ca_secret",
                            # 'sni': 'sni' (this field is not required in snake-camel but adding for completeness)
                        }

                        # We don't need any camel case in our generated TLSContext
                        for camel, snake in camel_snake_map.items():
                            if camel in host_tls_config:
                                # We use .pop() to actually replace the camelCase name with snake case
                                host_tls_config[snake] = host_tls_config.pop(camel)

                        if "min_tls_version" in host_tls_config:
                            if (
                                host_tls_config["min_tls_version"]
                                not in IRTLSContext.AllowedTLSVersions
                            ):
                                self.post_error(
                                    f"Host {self.name}: Invalid min_tls_version set in Host.tls: "
                                    f"{host_tls_config['min_tls_version']}"
                                )
                                return False

                        if "max_tls_version" in host_tls_config:
                            if (
                                host_tls_config["max_tls_version"]
                                not in IRTLSContext.AllowedTLSVersions
                            ):
                                self.post_error(
                                    f"Host {self.name}: Invalid max_tls_version set in Host.tls: "
                                    f"{host_tls_config['max_tls_version']}"
                                )
                                return False

                        tls_context_init = dict(
                            rkey=self.rkey,
                            name=ctx_name,
                            namespace=self.namespace,
                            location=self.location,
                            hosts=[self.hostname or self.name],
                            secret=tls_name,
                        )

                        tls_config_context = IRTLSContext(
                            ir, aconf, **tls_context_init, **host_tls_config
                        )

                        # This code was here because, while 'selector' was controlling things to be watched
                        # for, we figured we should update the labels on this generated TLSContext so that
                        # it would actually match the 'selector'. Nothing was actually using that, though, so
                        # we're not doing that any more.
                        #
                        # -----------------------------------------------------------------------------------
                        # # XXX This seems kind of pointless -- nothing looks at the context's labels?
                        # match_labels = self.get('selector', {}).get('matchLabels')

                        # if match_labels:
                        #     tls_config_context['metadata_labels'] = match_labels

                        if tls_config_context.is_active():
                            self.context = tls_config_context
                            tls_config_context.referenced_by(self)
                            tls_config_context.sourced_by(self)

                            ir.save_tls_context(tls_config_context)
                        else:
                            self.post_error(
                                f"Host {self.name}: generated TLSContext {tls_config_context.name} from "
                                f"Host.tls is not valid"
                            )
                            return False

                    elif implicit_tls_exists:
                        # They didn't say anything explicitly, but it happens that a context with the
                        # correct name for this Host already exists. Save that, if it works out for us.
                        ir.logger.debug(f"Host {self.name}: TLSContext {ctx_name} already exists")

                        if not self.save_context(ir, ctx_name, tls_ss, tls_name):
                            return False
                    else:
                        ir.logger.debug(f"Host {self.name}: creating TLSContext {ctx_name}")

                        new_ctx = dict(
                            rkey=self.rkey,
                            name=ctx_name,
                            namespace=self.namespace,
                            location=self.location,
                            hosts=[self.hostname or self.name],
                            secret=tls_name,
                        )

                        ctx = IRTLSContext(ir, aconf, **new_ctx)

                        match_labels = self.get("matchLabels")

                        if not match_labels:
                            match_labels = self.get("match_labels")

                        if match_labels:
                            ctx["metadata_labels"] = match_labels

                        if ctx.is_active():
                            self.context = ctx
                            ctx.referenced_by(self)
                            ctx.sourced_by(self)

                            ir.save_tls_context(ctx)
                        else:
                            ir.logger.error(
                                f"Host {self.name}: new TLSContext {ctx_name} is not valid"
                            )
                else:
                    ir.logger.error(
                        f"Host {self.name}: invalid TLS secret {tls_name}, marking inactive"
                    )
                    return False

        if self.get("acmeProvider", None):
            acme = self.acmeProvider

            # The ACME client is disabled if we're running as an intercept agent.
            if ir.edge_stack_allowed and not ir.agent_active:
                authority = acme.get("authority", None)

                if authority and (authority.lower() != "none"):
                    # ACME is active, which means that we must have an insecure_addl_port.
                    # Make sure we do -- if no port is set at all, just silently default it,
                    # but if for some reason they tried to force it disabled, be noisy.

                    override_insecure = False

                    if self.insecure_addl_port is None:
                        # Override silently, since it's not set at all.
                        override_insecure = True
                    elif self.insecure_addl_port < 0:
                        # Override noisily, since they tried to explicitly disable it.
                        self.post_error(
                            "ACME requires insecure.additionalPort to function; forcing to 8080"
                        )
                        override_insecure = True

                    if override_insecure:
                        # Force self.insecure_addl_port...
                        self.insecure_addl_port = 8080

                        # ...but also update the actual policy dict, too.
                        insecure_policy["additionalPort"] = 8080

                        if "action" not in insecure_policy:
                            # No action when we're overriding the additionalPort already means that we
                            # default the action to Reject (the hole-puncher will do the right thing).
                            insecure_policy["action"] = "Reject"

                        request_policy["insecure"] = insecure_policy
                        self["requestPolicy"] = request_policy

            pkey_secret = acme.get("privateKeySecret", None)

            if pkey_secret:
                pkey_name = pkey_secret.get("name", None)

                if pkey_name:
                    ir.logger.debug(f"Host {self.name}: ACME private key name is {pkey_name}")

                    pkey_ss = self.resolve(ir, pkey_name)

                    if not pkey_ss:
                        ir.logger.error(
                            f"Host {self.name}: continuing with invalid private key secret {pkey_name}; ACME will not be able to renew this certificate"
                        )
                        self.post_error(
                            f"continuing with invalid ACME private key secret {pkey_name}; ACME will not be able to renew this certificate"
                        )

        ir.logger.debug(f"Host setup OK: {self}")
        return True

    # Check a TLSContext name, and save the linked TLSContext if it'll work for us.
    def save_context(self, ir: "IR", ctx_name: str, tls_ss: SavedSecret, tls_name: str):
        # First obvious thing: does a TLSContext with the right name even exist?
        if not ir.has_tls_context(ctx_name):
            self.post_error(
                "Host %s: Specified TLSContext does not exist: %s" % (self.name, ctx_name)
            )
            return False

        ctx = ir.get_tls_context(ctx_name)
        assert ctx  # For mypy -- we checked above to be sure it exists.

        # Make sure that the TLSContext is "compatible" i.e. it at least has the same cert related
        # configuration as the one in this Host AND hosts are same as well.

        if ctx.has_secret():
            secret_name = ctx.secret_name()
            assert secret_name  # For mypy -- if has_secret() is true, secret_name() will be there.

            # This is a little weird. Basically we're going to resolve the secret (which should just
            # be a cache lookup here) so that we can use SavedSecret.__str__() as a serializer to
            # compare the configurations.
            context_ss = self.resolve(ir, secret_name)

            self.logger.debug(
                f"Host {self.name}, ctx {ctx.name}, secret {secret_name}, resolved {context_ss}"
            )

            if str(context_ss) != str(tls_ss):
                self.post_error(
                    "Secret info mismatch between Host %s (secret: %s) and TLSContext %s: (secret: %s)"
                    % (self.name, tls_name, ctx_name, secret_name)
                )
                return False
        else:
            # This will often be a no-op.
            ctx.set_secret_name(tls_name)

        # TLS config is good, let's make sure the hosts line up too.
        context_hosts = ctx.get("hosts")

        # XXX WTF? self.name is not OK as a hostname! Leaving this for the moment, but it's
        # almost certainly getting shredded before 2.0 GAs.
        host_hosts = [self.name]

        if self.hostname:
            host_hosts.append(self.hostname)

        if context_hosts:
            is_valid_hosts = False

            # XXX Should we be doing a glob check here?
            for host_tc in context_hosts:
                if host_tc in host_hosts:
                    is_valid_hosts = True

            if not is_valid_hosts:
                self.post_error(
                    "Hosts mismatch between Host %s (accepted hosts: %s) and TLSContext %s (hosts: %s)"
                    % (self.name, host_hosts, ctx_name, context_hosts)
                )
                # XXX Shouldn't we return false here?
        else:
            # XXX WTF? self.name is not OK as a hostname!
            ctx["hosts"] = [self.hostname or self.name]

        self.logger.debug(f"Host {self.name}, final ctx {ctx.name}: {ctx.as_json()}")

        # All seems good, this context belongs to self now!
        self.context = ctx

        return True

    def matches_httpgroup(self, group: "IRHTTPMappingGroup") -> bool:
        """
        Make sure a given IRHTTPMappingGroup is a match for this Host, meaning
        that at least one of the following is true:

        - The Host specifies mappingSelector.matchLabels, and the group has matching labels
        - The group specifies a host glob, and the Host has a matching domain.

        A Mapping that specifies no host can never match a Host that specifies no
        mappingSelector.
        """

        host_match = False
        sel_match = False

        group_regex = group.get("host_regex") or False

        if group_regex:
            # It matches.
            host_match = True
            self.logger.debug("-- hostname %s group regex => %s", self.hostname, host_match)
        else:
            # It is NOT A TYPO that we use group.get("host") here -- whether the Mapping supplies
            # "hostname" or "host", the Mapping code normalizes to "host" internally.

            # It's possible for group.host_redirect to be None instead of missing, and it's also
            # conceivably possible for group.host_redirect.host to be "", which we'd rather be
            # None. Hence we do this two-line dance to massage the various cases.
            host_redirect = (group.get("host_redirect") or {}).get("host")
            group_glob = group.get("host") or host_redirect  # NOT A TYPO: see above.

            if group_glob:
                host_match = hostglob_matches(self.hostname, group_glob)
                self.logger.debug(
                    "-- hostname %s group glob %s => %s", self.hostname, group_glob, host_match
                )

        mapsel = self.get("mappingSelector")

        if mapsel:
            sel_match = selector_matches(self.logger, mapsel, group.get("metadata_labels", {}))
            self.logger.debug(
                "-- host sel %s group labels %s => %s",
                dump_json(mapsel),
                dump_json(group.get("metadata_labels")),
                sel_match,
            )

        return host_match or sel_match

    def __str__(self) -> str:
        request_policy = self.get("requestPolicy", {})
        insecure_policy = request_policy.get("insecure", {})
        insecure_action = insecure_policy.get("action", "Redirect")
        insecure_addl_port = insecure_policy.get("additionalPort", None)

        ctx_name = self.context.name if self.context else "-none-"
        return "<Host %s for %s ns %s ctx %s ia %s iap %s>" % (
            self.name,
            self.hostname or "*",
            self.namespace,
            ctx_name,
            insecure_action,
            insecure_addl_port,
        )

    def resolve(self, ir: "IR", secret_name: str) -> SavedSecret:
        # Try to use our namespace for secret resolution. If we somehow have no
        # namespace, fall back to the Ambassador's namespace.
        namespace = self.namespace or ir.ambassador_namespace

        return ir.resolve_secret(self, secret_name, namespace)


class HostFactory:
    @classmethod
    def load_all(cls, ir: "IR", aconf: Config) -> None:
        assert ir

        hosts = aconf.get_config("hosts")

        if hosts:
            for config in hosts.values():
                ir.logger.debug("HostFactory: creating host for %s" % repr(config.as_dict()))

                host = IRHost(ir, aconf, **config)

                if host.is_active():
                    host.referenced_by(config)
                    host.sourced_by(config)

                    ir.logger.debug(f"HostFactory: saving host {host}")
                    ir.save_host(host)
                else:
                    ir.logger.debug(f"HostFactory: not saving inactive host {host}")

    @classmethod
    def finalize(cls, ir: "IR", aconf: Config) -> None:
        # First up: how many Hosts do we have?
        host_count = len(ir.get_hosts() or [])

        # Do we have any termination contexts for TLS? (If not, we'll need to bring the
        # fallback cert into play.)
        #
        # (This empty_contexts stuff mypy silliness to deal with the fact that ir.get_tls_contexts()
        # returns a ValuesView[IRTLSContext] rather than a List[IRTLSContext].
        empty_contexts: List[IRTLSContext] = []
        contexts: List[IRTLSContext] = list(ir.get_tls_contexts()) or empty_contexts

        found_termination_context = False
        for ctx in contexts:
            if ctx.get("hosts"):  # not None and not the empty list
                found_termination_context = True
                break

        ir.logger.debug(
            f"HostFactory: Host count %d, %s TLS termination contexts"
            % (host_count, "with" if found_termination_context else "no")
        )

        # OK, do we have any Hosts?
        if host_count == 0:
            # Nope. First up, scream if we _do_ have termination contexts...
            if found_termination_context:
                ir.post_error(
                    "No Hosts defined, but TLSContexts exist that terminate TLS. The TLSContexts are being ignored."
                )

            # If we don't have a fallback secret, don't try to use it.
            #
            # We use the Ambassador's namespace here because we'll be creating the
            # fallback Host in the Ambassador's namespace.
            fallback_ss = ir.resolve_secret(
                ir.ambassador_module, "fallback-self-signed-cert", ir.ambassador_namespace
            )

            host: IRHost

            if not fallback_ss:
                ir.aconf.post_notice(
                    "No TLS termination and no fallback cert -- defaulting to cleartext-only."
                )
                ir.logger.debug("HostFactory: creating cleartext-only default host")

                host = IRHost(
                    ir,
                    aconf,
                    rkey="-internal",
                    name="default-host",
                    location="-internal-",
                    hostname="*",
                    requestPolicy={"insecure": {"action": "Route"}},
                )
            else:
                ir.logger.debug(f"HostFactory: creating TLS-enabled default Host")

                host = IRHost(
                    ir,
                    aconf,
                    rkey="-internal",
                    name="default-host",
                    location="-internal-",
                    hostname="*",
                    tlsSecret={"name": "fallback-self-signed-cert"},
                )

            if not host.is_active():
                ir.post_error(
                    "Synthesized default host is inactive? %s" % dump_json(host.as_dict())
                )
            else:
                host.referenced_by(ir.ambassador_module)
                host.sourced_by(ir.ambassador_module)

                ir.logger.debug(f"HostFactory: saving host {host}")
                ir.save_host(host)
