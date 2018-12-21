from typing import ClassVar, List, Optional, TYPE_CHECKING

from ..utils import RichStatus
from ..config import ACResource
from .irresource import IRResource

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

        if not self.validate(config):
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

        resolved = ir.tls_secret_resolver(secret_name=self.name, context=self,
                                          namespace=ir.ambassador_namespace)

        rc = False

        if resolved:
            self.secret_info.update(resolved)
            rc = True

        ir.logger.debug("IRTLSContext setup done (returning %s): %s" % (rc, self.as_json()))

        return rc

    def validate(self, config) -> bool:
        if 'name' not in config:
            self.post_error(RichStatus.fromError("`name` field is required in a TLSContext resource", module=config))
            return False

        if 'hosts' in config:
            termination_count = 0
            errors = 0

            if 'secret' in config:
                termination_count += 1

            if 'cert_chain_file' in config:
                termination_count += 1

                if 'private_key_file' not in config:
                    self.post_error(
                        RichStatus.fromError("`cert_chain_file` requires `private_key` in a TLSContext resource",
                                             module=config))
                    errors += 1

            if termination_count != 1:
                self.post_error(RichStatus.fromError(
                    "Either `secret` or `cert_chain_file` must be present in a TLSContext resource", module=config))
                errors += 1

            if errors:
                return False
        else:
            # Not a termination context. We don't support this yet.
            self.post_error(RichStatus.fromError("`hosts` is currently required in a TLSContext resource", module=config))
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
    def fromConfig(cls, ir: 'IR', rkey: str, location: str, *,
                   kind="synthesized-TLS-context", name: str, **kwargs) -> 'IRTLSContext':
        ctx_config = ACResource(rkey, location, kind=kind, name=name, **kwargs)

        return cls(ir, ctx_config)
