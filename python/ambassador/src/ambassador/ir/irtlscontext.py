import base64
import logging
from typing import TYPE_CHECKING, ClassVar, Dict, List, Optional

from ..config import Config
from ..utils import SavedSecret
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover
    from .irtls import IRAmbassadorTLS  # pragma: no cover


class IRTLSContext(IRResource):
    CertKeys: ClassVar = {
        "secret",
        "cert_chain_file",
        "private_key_file",
        "ca_secret",
        "cacert_chain_file",
        "crl_secret",
        "crl_file",
    }

    AllowedKeys: ClassVar = {
        "_ambassador_enabled",
        "_legacy",
        "alpn_protocols",
        "cert_required",
        "cipher_suites",
        "ecdh_curves",
        "hosts",
        "max_tls_version",
        "min_tls_version",
        "redirect_cleartext_from",
        "secret_namespacing",
        "sni",
    }

    AllowedTLSVersions = ["v1.0", "v1.1", "v1.2", "v1.3"]

    name: str
    hosts: Optional[List[str]]
    alpn_protocols: Optional[str]
    cert_required: Optional[bool]
    min_tls_version: Optional[str]
    max_tls_version: Optional[str]
    cipher_suites: Optional[str]
    ecdh_curves: Optional[str]
    redirect_cleartext_from: Optional[int]
    secret_namespacing: Optional[bool]
    secret_info: dict
    sni: Optional[str]

    is_fallback: bool

    _ambassador_enabled: bool
    _legacy: bool

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str,  # REQUIRED
        name: str,  # REQUIRED
        location: str,  # REQUIRED
        namespace: Optional[str] = None,
        metadata_labels: Optional[Dict[str, str]] = None,
        kind: str = "IRTLSContext",
        apiVersion: str = "getambassador.io/v3alpha1",
        is_fallback: Optional[bool] = False,
        **kwargs,
    ) -> None:
        new_args = {
            x: kwargs[x]
            for x in kwargs.keys()
            if x in IRTLSContext.AllowedKeys.union(IRTLSContext.CertKeys)
        }

        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            location=location,
            kind=kind,
            name=name,
            namespace=namespace,
            metadata_labels=metadata_labels,
            is_fallback=is_fallback,
            apiVersion=apiVersion,
            **new_args,
        )

    def pretty(self) -> str:
        secret_name = self.secret_info.get("secret", "-no secret-")
        hoststr = getattr(self, "hosts", "-any-")
        fbstr = " (fallback)" if self.is_fallback else ""

        rcf = self.get("redirect_cleartext_from", None)
        rcfstr = f" rcf {rcf}" if (rcf is not None) else ""

        return f"<IRTLSContext {self.name}.{self.namespace}{rcfstr}{fbstr}: hosts {hoststr} secret {secret_name}>"

    def setup(self, ir: "IR", aconf: Config) -> bool:
        if not self.get("_ambassador_enabled", False):
            spec_count = 0
            errors = 0

            if self.get("secret", None):
                spec_count += 1

            if self.get("cert_chain_file", None):
                spec_count += 1

                if not self.get("private_key_file", None):
                    err_msg = f"TLSContext {self.name}: 'cert_chain_file' requires 'private_key_file' as well"

                    self.post_error(err_msg)
                    errors += 1

            if spec_count == 2:
                err_msg = f"TLSContext {self.name}: exactly one of 'secret' and 'cert_chain_file' must be present"

                self.post_error(err_msg)
                errors += 1

            if errors:
                return False

        # self.sourced_by(config)
        # self.referenced_by(config)

        # Assume that we have no redirect_cleartext_from...
        rcf = self.get("redirect_cleartext_from", None)

        if rcf is not None:
            try:
                self.redirect_cleartext_from = int(rcf)
            except ValueError:
                err_msg = f"TLSContext {self.name}: redirect_cleartext_from must be a port number rather than '{rcf}'"
                self.post_error(err_msg)
                self.redirect_cleartext_from = None

        # Finally, move cert keys into secret_info.
        self.secret_info = {}

        for key in IRTLSContext.CertKeys:
            if key in self:
                self.secret_info[key] = self.pop(key)

        ir.logger.debug(f"IRTLSContext setup good: {self.pretty()}")

        return True

    def resolve_secret(self, secret_name: str) -> SavedSecret:
        # Assume that we need to look in whichever namespace the TLSContext itself is in...
        namespace = self.namespace

        # You can't just always allow '.' in a secret name to span namespaces, or you end up with
        # https://github.com/datawire/ambassador/issues/1255, which is particularly problematic
        # because (https://github.com/datawire/ambassador/issues/1475) Istio likes to use '.' in
        # mTLS secret names. So we default to allowing the '.' as a namespace separator, but
        # you can set secret_namespacing to False in a TLSContext or tls_secret_namespacing False
        # in the Ambassador module's defaults to prevent that.

        secret_namespacing = self.lookup(
            "secret_namespacing", True, default_key="tls_secret_namespacing"
        )

        self.ir.logger.debug(
            f"TLSContext.resolve_secret {secret_name}, namespace {namespace}: namespacing is {secret_namespacing}"
        )

        if "." in secret_name and secret_namespacing:
            secret_name, namespace = secret_name.rsplit(".", 1)

        return self.ir.resolve_secret(self, secret_name, namespace)

    def resolve(self) -> bool:
        if self.get("_ambassador_enabled", False):
            self.ir.logger.debug("IRTLSContext skipping resolution of null context")
            return True

        # is_valid determines if the TLS context is valid
        is_valid = False

        # If redirect_cleartext_from or alpn_protocols is specified, the TLS Context is
        # valid anyway, even if secret config is invalid
        if self.get("redirect_cleartext_from", False) or self.get("alpn_protocols", False):
            is_valid = True

        # If we don't have secret info, it's worth posting an error.
        if not self.secret_info:
            self.post_error(
                "TLSContext %s has no certificate information at all?" % self.name,
                log_level=logging.DEBUG,
            )

        self.ir.logger.debug("resolve_secrets working on: %s" % self.as_json())

        # OK. Do we have a secret name?
        secret_name = self.secret_info.get("secret")
        secret_valid = True

        if secret_name:
            # Yes. Try loading it. This always returns a SavedSecret, so that our caller
            # has access to the name and namespace. The SavedSecret will evaluate non-True
            # if we found no cert though.
            ss = self.resolve_secret(secret_name)

            self.ir.logger.debug("resolve_secrets: IR returned secret %s as %s" % (secret_name, ss))

            if not ss:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # post an error.
                self.post_error(
                    "TLSContext %s found no certificate in %s, ignoring..." % (self.name, ss.name)
                )
                self.secret_info.pop("secret")
                secret_valid = False
            else:
                # If they only gave a public key, that's an error
                if not ss.key_path:
                    self.post_error(
                        "TLSContext %s found no private key in %s" % (self.name, ss.name)
                    )
                    return False

                # So far, so good.
                self.ir.logger.debug("TLSContext %s saved secret %s" % (self.name, ss.name))

                # Update paths for this cert.
                self.secret_info["cert_chain_file"] = ss.cert_path
                self.secret_info["private_key_file"] = ss.key_path

                if ss.root_cert_path:
                    self.secret_info["cacert_chain_file"] = ss.root_cert_path

        self.ir.logger.debug(
            "TLSContext - successfully processed the cert_chain_file, private_key_file, and cacert_chain_file: %s"
            % self.secret_info
        )

        # OK. Repeat for the crl_secret.
        crl_secret = self.secret_info.get("crl_secret")
        if crl_secret:
            # They gave a secret name for the certificate revocation list. Try loading it.
            crls = self.resolve_secret(crl_secret)

            self.ir.logger.debug(
                "resolve_secrets: IR returned secret %s as %s" % (crl_secret, crls)
            )

            if not crls:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # give up.
                self.post_error(
                    "TLSContext %s found no certificate revocation list in %s"
                    % (self.name, crls.name)
                )
                secret_valid = False
            else:
                self.ir.logger.debug(
                    "TLSContext %s saved certificate revocation list secret %s"
                    % (self.name, crls.name)
                )
                self.secret_info["crl_file"] = crls.user_path

        # OK. Repeat for the ca_secret_name.
        ca_secret_name = self.secret_info.get("ca_secret")

        if ca_secret_name:
            if not self.secret_info.get("cert_chain_file"):
                # DUPLICATED BELOW: This is an error: validation without termination isn't meaningful.
                # (This is duplicated for the case where they gave a validation path.)
                self.post_error(
                    "TLSContext %s cannot validate client certs without TLS termination" % self.name
                )
                return False

            # They gave a secret name for the validation cert. Try loading it.
            ss = self.resolve_secret(ca_secret_name)

            self.ir.logger.debug(
                "resolve_secrets: IR returned secret %s as %s" % (ca_secret_name, ss)
            )

            if not ss:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # give up.
                self.post_error(
                    "TLSContext %s found no validation certificate in %s" % (self.name, ss.name)
                )
                secret_valid = False
            else:
                # Validation certs don't need the private key, but it's not an error if they gave
                # one. We're good to go here.
                self.ir.logger.debug("TLSContext %s saved CA secret %s" % (self.name, ss.name))
                self.secret_info["cacert_chain_file"] = ss.cert_path

                # While we're here, did they set cert_required _in the secret_?
                if ss.cert_data:
                    cert_required = ss.cert_data.get("cert_required")

                    if cert_required is not None:
                        decoded = base64.b64decode(cert_required).decode("utf-8").lower() == "true"

                        # cert_required is at toplevel, _not_ in secret_info!
                        self["cert_required"] = decoded
        else:
            # No secret is named; did they provide a file location instead?
            if self.secret_info.get("cacert_chain_file") and not self.secret_info.get(
                "cert_chain_file"
            ):
                # DUPLICATED ABOVE: This is an error: validation without termination isn't meaningful.
                # (This is duplicated for the case where they gave a validation secret.)
                self.post_error(
                    "TLSContext %s cannot validate client certs without TLS termination" % self.name
                )
                return False

        # If the secret has been invalidated above, then we do not need to check for paths down under.
        # We can return whether the TLS Context is valid or not.
        if not secret_valid:
            return is_valid

        # OK. Check paths.
        errors = 0

        # self.ir.logger.debug("resolve_secrets before path checks: %s" % self.as_json())
        for key in [
            "cert_chain_file",
            "private_key_file",
            "cacert_chain_file",
            "crl_file",
        ]:
            path = self.secret_info.get(key, None)

            if path:
                fc = getattr(self.ir, "file_checker")
                if not fc(path):
                    self.post_error("TLSContext %s found no %s '%s'" % (self.name, key, path))
                    errors += 1
            elif (not (key == "cacert_chain_file" or key == "crl_file")) and self.get(
                "hosts", None
            ):
                self.post_error("TLSContext %s is missing %s" % (self.name, key))
                errors += 1

        if errors > 0:
            return False

        return True

    def has_secret(self) -> bool:
        # Safely verify that self.secret_info['secret'] exists -- in other words, verify
        # that this IRTLSContext is based on a Secret we load from elsewhere, rather than
        # on files in the filesystem.
        si = self.get("secret_info", {})

        return "secret" in si

    def secret_name(self) -> Optional[str]:
        # Return the name of the Secret we're based on, or None if we're based on files
        # in the filesystem.
        #
        # XXX Currently this implies a _Kubernetes_ Secret, and we might have to change
        # this later.

        if self.has_secret():
            return self.secret_info["secret"]
        else:
            return None

    def set_secret_name(self, secret_name: str) -> None:
        # Set the name of the Secret we're based on.
        self.secret_info["secret"] = secret_name

    @classmethod
    def null_context(cls, ir: "IR") -> "IRTLSContext":
        ctx = ir.get_tls_context("no-cert-upstream")

        if not ctx:
            ctx = IRTLSContext(
                ir,
                ir.aconf,
                rkey="ir.no-cert-upstream",
                name="no-cert-upstream",
                location="ir.no-cert-upstream",
                kind="null-TLS-context",
                _ambassador_enabled=True,
            )

            ir.save_tls_context(ctx)

        return ctx

    @classmethod
    def from_legacy(
        cls,
        ir: "IR",
        name: str,
        rkey: str,
        location: str,
        cert: "IRAmbassadorTLS",
        termination: bool,
        validation_ca: Optional["IRAmbassadorTLS"],
    ) -> "IRTLSContext":
        """
        Create an IRTLSContext from a legacy TLS-module style definition.

        'cert' is the TLS certificate that we'll offer to our peer -- for a termination
        context, this is our server cert, and for an origination context, it's our client
        cert.

        For termination contexts, 'validation_ca' may also be provided. It's the TLS
        certificate that we'll use to validate the certificates our clients offer. Note
        that no private key is needed or supported.

        :param ir: IR in play
        :param name: name for the newly-created context
        :param rkey: rkey for the newly-created context
        :param location: location for the newly-created context
        :param cert: information about the cert to present to the peer
        :param termination: is this a termination context?
        :param validation_ca: information about how we'll validate the peer's cert
        :return: newly-created IRTLSContext
        """
        new_args = {}

        for key in [
            "secret",
            "cert_chain_file",
            "private_key_file",
            "alpn_protocols",
            "redirect_cleartext_from",
        ]:
            value = cert.get(key, None)

            if value:
                new_args[key] = value

        if (
            ("secret" not in new_args)
            and ("cert_chain_file" not in new_args)
            and ("private_key_file" not in new_args)
        ):
            # Assume they want the 'ambassador-certs' secret.
            new_args["secret"] = "ambassador-certs"

        if termination:
            new_args["hosts"] = ["*"]

            if validation_ca and validation_ca.get("enabled", True):
                for key in ["secret", "cacert_chain_file", "cert_required"]:
                    value = validation_ca.get(key, None)

                    if value:
                        if key == "secret":
                            new_args["ca_secret"] = value
                        else:
                            new_args[key] = value

                if ("ca_secret" not in new_args) and ("cacert_chain_file" not in new_args):
                    # Assume they want the 'ambassador-cacert' secret.
                    new_args["secret"] = "ambassador-cacert"

        ctx = IRTLSContext(
            ir,
            ir.aconf,
            rkey=rkey,
            name=name,
            location=location,
            kind="synthesized-TLS-context",
            _legacy=True,
            **new_args,
        )

        return ctx


class TLSContextFactory:
    @classmethod
    def load_all(cls, ir: "IR", aconf: Config) -> None:
        assert ir

        # Save TLS contexts from the aconf into the IR. Note that the contexts in the aconf
        # are just ACResources; they need to be turned into IRTLSContexts.
        tls_contexts = aconf.get_config("tls_contexts")

        if tls_contexts is not None:
            for config in tls_contexts.values():
                ctx = IRTLSContext(ir, aconf, **config)

                if ctx.is_active():
                    ctx.referenced_by(config)
                    ctx.sourced_by(config)

                    ir.save_tls_context(ctx)
