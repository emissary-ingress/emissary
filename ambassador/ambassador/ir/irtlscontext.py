from typing import ClassVar, Dict, List, Optional, Tuple, TYPE_CHECKING

import base64
import os

from ..utils import RichStatus, SavedSecret
from ..config import ACResource
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR
    from .irtls import IRAmbassadorTLS


class IRTLSContext(IRResource):
    CertKeys: ClassVar = [
        'secret',
        'cert_chain_file',
        'private_key_file',

        'ca_secret',
        'cacert_chain_file'
    ]

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

    def __init__(self, ir: 'IR', config,
                 rkey: str="ir.tlscontext",
                 kind: str="IRTLSContext",
                 name: str="tlscontext",
                 **kwargs) -> None:

        super().__init__(
            ir=ir, aconf=config, rkey=rkey, kind=kind, name=name
        )

    def setup(self, ir: 'IR', config) -> bool:
        # ir.logger.debug("IRTLSContext incoming config: %s" % config.as_json())

        if config.get('_ambassador_enabled', False):
            # Null context.
            self['_ambassador_enabled'] = True
        elif not self.validate(config):
            return False

        self.sourced_by(config)
        self.referenced_by(config)

        self.name = config.get('name')
        self.hosts = config.get('hosts')
        self.alpn_protocols = config.get('alpn_protocols')
        self.cert_required = config.get('cert_required')
        self.min_tls_version = config.get('min_tls_version')
        self.max_tls_version = config.get('max_tls_version')
        self.cipher_suites = config.get('cipher_suites')
        self.ecdh_curves = config.get('ecdh_curves')
        self.secret_namespacing = config.get('secret_namespacing', None)

        rcf = config.get('redirect_cleartext_from')

        if rcf is not None:
            try:
                self.redirect_cleartext_from = int(rcf)
            except ValueError:
                self.post_error("redirect_cleartext_from must give a port number rather than '%s'" % rcf)
                self.redirect_cleartext_from = None

        # Finally, set up our secret_info.
        self.secret_info = {}

        for key in IRTLSContext.CertKeys:
            if key in config:
                self.secret_info[key] = config[key]

        # ir.logger.debug("IRTLSContext at setup: %s" % self.as_json())
        #
        # rc = False
        #
        # if self.get('_ambassador_enabled', False):
        #     ir.logger.debug("IRTLSContext skipping resolution of null context")
        #     rc = True
        # else:
        #     if self.resolve():
        #         rc = True

        rc = True

        ir.logger.debug("IRTLSContext setup done (returning %s): %s" % (rc, self.as_json()))

        return rc

    def validate(self, config) -> bool:
        if 'name' not in config:
            self.post_error(RichStatus.fromError("`name` field is required in a TLSContext resource", module=config))
            return False

        spec_count = 0
        errors = 0

        if config.get('secret', None):
            spec_count += 1

        if config.get('cert_chain_file', None):
            spec_count += 1

            if not config.get('private_key_file', None):
                err_msg = "TLSContext %s: 'cert_chain_file' requires 'private_key_file' as well" % config.name

                self.post_error(RichStatus.fromError(err_msg, module=config))
                errors += 1

        if spec_count == 2:
            err_msg = "TLSContext %s: exactly one of 'secret' and 'cert_chain_file' must be present" % config.name

            self.post_error(RichStatus.fromError(err_msg, module=config))
            errors += 1

        if errors:
            return False

        return True

    def resolve(self) -> bool:
        if self.get('_ambassador_enabled', False):
            self.ir.logger.debug("IRTLSContext skipping resolution of null context")
            return True

        # is_valid determines if the TLS context is valid
        is_valid = False

        # If redirect_cleartext_from or alpn_protocols is specified, the TLS Context is
        # valid anyway, even if secret config is invalid
        if self.get('redirect_cleartext_from', False) or self.get('alpn_protocols', False):
            is_valid = True

        # If we don't have secret info, it's worth logging.
        if not self.secret_info:
            self.logger.info("TLSContext %s has no certificate information at all?" % self.name)

        self.ir.logger.debug("resolve_secrets working on: %s" % self.as_json())

        # OK. Do we have a secret name?
        secret_name = self.secret_info.get('secret')
        secret_valid = True

        if secret_name:
            # Yes. Try loading it. This always returns a SavedSecret, so that our caller
            # has access to the name and namespace. The SavedSecret will evaluate non-True
            # if we found no cert though.
            ss = self.ir.resolve_secret(self, secret_name)

            self.ir.logger.debug("resolve_secrets: IR returned secret %s as %s" % (secret_name, ss))

            if not ss:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # post an error.
                self.post_error("TLSContext %s found no certificate in %s, ignoring..." % (self.name, ss.name))
                self.secret_info.pop('secret')
                secret_valid = False
            else:
                # If they only gave a public key, that's an error
                if not ss.key_path:
                    self.post_error("TLSContext %s found no private key in %s" % (self.name, ss.name))
                    return False

                # So far, so good.
                self.ir.logger.debug("TLSContext %s saved secret %s" % (self.name, ss.name))

                # Update paths for this cert.
                self.secret_info['cert_chain_file'] = ss.cert_path
                self.secret_info['private_key_file'] = ss.key_path

        # OK. Repeat for the ca_secret_name.
        ca_secret_name = self.secret_info.get('ca_secret')

        if ca_secret_name:
            if not self.secret_info.get('cert_chain_file'):
                # DUPLICATED BELOW: This is an error: validation without termination isn't meaningful.
                # (This is duplicated for the case where they gave a validation path.)
                self.post_error("TLSContext %s cannot validate client certs without TLS termination" %
                                self.name)
                return False

            # They gave a secret name for the validation cert. Try loading it.
            ss = self.ir.resolve_secret(self, ca_secret_name)

            self.ir.logger.debug("resolve_secrets: IR returned secret %s as %s" % (ca_secret_name, ss))

            if not ss:
                # This is definitively an error: they mentioned a secret, it can't be loaded,
                # give up.
                self.post_error("TLSContext %s found no validation certificate in %s" % (self.name, ss.name))
                secret_valid = False
            else:
                # Validation certs don't need the private key, but it's not an error if they gave
                # one. We're good to go here.
                self.ir.logger.debug("TLSContext %s saved CA secret %s" % (self.name, ss.name))
                self.secret_info['cacert_chain_file'] = ss.cert_path

                # While we're here, did they set cert_required _in the secret_?
                if ss.cert_data:
                    cert_required = ss.cert_data.get('cert_required')

                    if cert_required is not None:
                        decoded = base64.b64decode(cert_required).decode('utf-8').lower() == 'true'

                        # cert_required is at toplevel, _not_ in secret_info!
                        self['cert_required'] = decoded
        else:
            # No secret is named; did they provide a file location instead?
            if self.secret_info.get('cacert_chain_file') and not self.secret_info.get('cert_chain_file'):
                # DUPLICATED ABOVE: This is an error: validation without termination isn't meaningful.
                # (This is duplicated for the case where they gave a validation secret.)
                self.post_error("TLSContext %s cannot validate client certs without TLS termination" %
                                self.name)
                return False

        # If the secret has been invalidated above, then we do not need to check for paths down under.
        # We can return whether the TLS Context is valid or not.
        if not secret_valid:
            return is_valid

        # OK. Check paths.
        errors = 0

        # self.ir.logger.debug("resolve_secrets before path checks: %s" % self.as_json())
        for key in [ 'cert_chain_file', 'private_key_file', 'cacert_chain_file' ]:
            path = self.secret_info.get(key, None)

            if path:
                fc = getattr(self.ir, 'file_checker')
                if not fc(path):
                    self.post_error("TLSContext %s found no %s '%s'" % (self.name, key, path))
                    errors += 1
            elif key != 'cacert_chain_file' and self.hosts:
                self.post_error("TLSContext %s is missing %s" % (self.name, key))
                errors += 1

        if errors > 0:
            return False

        return True

    @classmethod
    def from_config(cls, ir: 'IR', rkey: str, location: str, *,
                    kind="synthesized-TLS-context", name: str, **kwargs) -> 'IRTLSContext':
        ctx_config = ACResource(rkey, location, kind=kind, name=name, **kwargs)

        return cls(ir, ctx_config)

    @classmethod
    def null_context(cls, ir: 'IR') -> 'IRTLSContext':
        ctx = ir.get_tls_context("no-cert-upstream")

        if not ctx:
            ctx = IRTLSContext.from_config(ir, "ir.no-cert-upstream", "ir.no-cert-upstream",
                                           kind="null-TLS-context", name="no-cert-upstream",
                                           _ambassador_enabled=True)

            ir.save_tls_context(ctx)

        return ctx

    @classmethod
    def from_legacy(cls, ir: 'IR', name: str, rkey: str, location: str,
                    cert: 'IRAmbassadorTLS', termination: bool,
                    validation_ca: Optional['IRAmbassadorTLS']) -> 'IRTLSContext':
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

        for key in [ 'secret', 'cert_chain_file', 'private_key_file',
                     'alpn_protocols', 'redirect_cleartext_from' ]:
            value = cert.get(key, None)

            if value:
                new_args[key] = value

        if (('secret' not in new_args) and
            ('cert_chain_file' not in new_args) and
            ('private_key_file' not in new_args)):
            # Assume they want the 'ambassador-certs' secret.
            new_args['secret'] = 'ambassador-certs'

        if termination:
            new_args['hosts'] = [ '*' ]

            if validation_ca and validation_ca.get('enabled', True):
                for key in [ 'secret', 'cacert_chain_file', 'cert_required' ]:
                    value = validation_ca.get(key, None)

                    if value:
                        if key == 'secret':
                            new_args['ca_secret'] = value
                        else:
                            new_args[key] = value

                if (('ca_secret' not in new_args) and
                    ('cacert_chain_file' not in new_args)):
                    # Assume they want the 'ambassador-cacert' secret.
                    new_args['secret'] = 'ambassador-cacert'

        # Remember that this is a legacy context.
        new_args['_legacy'] = True

        return IRTLSContext.from_config(ir, rkey, location,
                                        kind="synthesized-TLS-context",
                                        name=name, **new_args)
