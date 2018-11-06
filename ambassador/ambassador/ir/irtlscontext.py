from typing import Optional, TYPE_CHECKING

from ..utils import RichStatus
from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRTLSContext(IRResource):
    def __init__(self, ir: 'IR', config,
                 rkey: str="ir.tlscontext",
                 kind: str="ir.tlscontext",
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

        self.name = config.get('name')
        self.hosts = config.get('hosts')

        self.secret_info = {
            'secret': config.get('secret'),
        }
        resolved = self.handle_secret(ir, config.get('secret'))
        if resolved is None:
            return False
        self.secret_info.update(resolved)

        return True

    def validate(self, config) -> bool:
        if 'name' not in config:
            self.post_error(RichStatus.fromError("`name` field is required in a TLSContext resource", module=config))
            return False
        if 'hosts' not in config:
            self.post_error(RichStatus.fromError("`hosts` field is required in a TLSContext resource", module=config))
            return False
        if 'secret' not in config:
            self.post_error(RichStatus.fromError("`secret` field is required in a TLSContext resource", module=config))
            return False
        return True

    def handle_secret(self, ir: 'IR', secret):
        if ir.tls_secret_resolver is not None:
            resolved = ir.tls_secret_resolver(secret_name=secret, context="", cert_dir='/ambassador/{}/'.format(secret))
            if resolved is None:
                self.post_error(RichStatus.fromError("Secret {} could not be resolved".format(secret)))
                return None
            return resolved
