from typing import ClassVar, Dict, List, Optional, Tuple, TYPE_CHECKING

import base64
import os

from ..utils import RichStatus, SavedSecret
from ..config import Config, ACResource
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR
    from .irtls import IRAmbassadorTLS


class IRHost(IRResource):
    AllowedKeys = {
        'acmeProvider',
        'hostname',
        'selector',
        'tlsSecret',
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED
                 namespace: Optional[str]=None,
                 kind: str="IRHost",
                 apiVersion: str="ambassador/v2",   # Not a typo! See below.
                 **kwargs) -> None:

        new_args = {
            x: kwargs[x] for x in kwargs.keys()
            if x in IRHost.AllowedKeys
        }

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, apiVersion=apiVersion,
            **new_args
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        ir.logger.info(f"Host {self.name} setting up")

        if self.get('tlsSecret', None):
            tls_secret = self.tlsSecret
            tls_name = tls_secret.get('name', None)

            # ir.logger.info(f"Host {self.name} has TLS secret {tls_secret}")

            if tls_name:
                ir.logger.info(f"Host {self.name} has TLS secret name {tls_name}")

            ss = self.resolve(ir, tls_name)

            if not ss:
                # Bzzt.
                ir.logger.error(f"Host {self.name} has invalid TLS secret {tls_name}")
                return False

        if self.get('acmeProvider', None):
            acme = self.acmeProvider
            pkey_secret = acme.get('privateKeySecret', None)

            # ir.logger.info(f"Host {self.name} has ACME provider {acme}")

            if pkey_secret:
                # ir.logger.info(f"Host {self.name} has ACME privateKeySecret {pkey_secret}")

                pkey_name = pkey_secret.get('name', None)

                if pkey_name:
                    ir.logger.info(f"Host {self.name} has ACME private key name {pkey_name}")

                    ss = self.resolve(ir, pkey_name)

                    if not ss:
                        # Bzzt.
                        ir.logger.error(f"Host {self.name} has invalid private key secret {pkey_name}")
                        return False

        return True

    def resolve(self, ir: 'IR', secret_name: str) -> SavedSecret:
        # Try to use our namespace for secret resolution. If we somehow have no
        # namespace, fall back to the Ambassador's namespace.
        namespace = self.namespace or ir.ambassador_namespace

        return ir.resolve_secret(self, secret_name, namespace)


class HostFactory():
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        assert ir

        hosts = aconf.get_config('hosts')

        if hosts:
            for config in hosts.values():
                ir.logger.debug("creating host for %s" % repr(config.as_dict()))

                host = IRHost(ir, aconf, **config)
                ir.save_resource(host)
