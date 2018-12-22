from typing import ClassVar, List, Optional, TYPE_CHECKING

from ..utils import RichStatus
from ..config import ACResource
from .irresource import IRResource
from .irtls import IRAmbassadorTLS

if TYPE_CHECKING:
    from .ir import IR


class IRTLSContext(IRResource):
    CertKeys: ClassVar = [
        'secret',
        'cert_chain_file',
        'private_key_file',

        'ca_secret',
        'cacert_chain_file'
    ]

    def __init__(self, ir: 'IR', config,
                 rkey: str="ir.tlscontext",
                 kind: str="IRTLSContext",
                 name: str="tlscontext",
                 **kwargs) -> None:

        super().__init__(
            ir=ir, aconf=config, rkey=rkey, kind=kind, name=name
        )

    def setup(self, ir: 'IR', config) -> bool:
        ir.logger.debug("IRTLSContext incoming config: %s" % config.as_json())

        if config.get('_ambassador_enabled', False):
            # Null context.
            self['_ambassador_enabled'] = True
        elif not self.validate(config):
            return False

        self.sourced_by(config)
        self.referenced_by(config)

        self.name: str = config.get('name')
        self.hosts: List[str] = config.get('hosts')
        self.alpn_protocols: Optional[str] = config.get('alpn_protocols')
        self.cert_required: Optional[str] = config.get('cert_required')
        self.redirect_cleartext_from: Optional[str] = config.get('redirect_cleartext_from')

        self.secret_info = {}

        for key in IRTLSContext.CertKeys:
            if key in config:
                self.secret_info[key] = config[key]

        ir.logger.debug("IRTLSContext at setup: %s" % self.as_json())

        rc = False

        if self.get('_ambassador_enabled', False):
            ir.logger.debug("IRTLSContext skipping resolution of null context")
            rc = True
        else:
            resolved = ir.tls_secret_resolver(secret_name=self.name, context=self,
                                              namespace=ir.ambassador_namespace)

            if resolved:
                self.secret_info.update(resolved)
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

        if spec_count != 1:
            err_msg = "TLSContext %s: exactly one of 'secret' and 'cert_chain_file' must be present" % config.name

            self.post_error(RichStatus.fromError(err_msg, module=config))
            errors += 1

        if errors:
            return False

        return True

    # def handle_secret(self, ir: 'IR', secret):
    #     if ir.tls_secret_resolver is not None:
    #         resolved = ir.tls_secret_resolver(secret_name=secret, context=self,
    #                                           namespace=ir.ambassador_namespace,
    #                                           cert_dir='/ambassador/{}/'.format(secret))
    #         if resolved is None:
    #             self.post_error(RichStatus.fromError("Secret {} could not be resolved".format(secret)))
    #             return None
    #         return resolved

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
                    cert: IRAmbassadorTLS, termination: bool,
                    validation_ca: Optional[IRAmbassadorTLS]) -> 'IRTLSContext':
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
                        new_args[key] = value

                if (('ca_secret' not in new_args) and
                        ('cacert_chain_file' not in new_args)):
                    # Assume they want the 'ambassador-cacert' secret.
                    new_args['secret'] = 'ambassador-cacert'

        return IRTLSContext.from_config(ir, rkey, location,
                                        kind="synthesized-TLS-context",
                                        name=name, **new_args)
