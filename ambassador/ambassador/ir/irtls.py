# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import os

from typing import Optional, TYPE_CHECKING

from ..config import Config
from .irresource import IRResource as IRResource

if TYPE_CHECKING:
    from .ir import IR


#############################################################################
## tls.py -- the tls_context configuration object for Ambassador
##
## IREnvoyTLS is an Envoy TLS context. These are created from IRAmbassadorTLS
## objects.

class IREnvoyTLS (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.tlscontext",
                 kind: str="IRTLSContext",
                 name: str="ir.tlscontext",
                 enabled: bool=True,

                 **kwargs) -> None:
        """
        Initialize an IREnvoyTLS from the raw fields of its Resource.
        """

        ir.logger.debug("IREnvoyTLS __init__ (%s %s %s)" % (kind, name, kwargs))

        self.namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            enabled=enabled,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config):
        if not self.enabled:
            return False

        secret = self.get('secret')

        # Check if secret and certs, both are specified
        cert_specified = False
        if self.get('cert_chain_file') is not None or self.get('private_key_file') is not None:
            cert_specified = True

        if secret is not None and cert_specified:
            self.pop('secret', None)
            self.pop('cert_chain_file', None)
            self.pop('private_key_file', None)
            self.logger.error("Both, secret and certs are specified, stopping ...")
            return False

        if (secret is not None) and (ir.tls_secret_resolver is not None):
            resolved = ir.tls_secret_resolver(secret, self.get('name'))

            if resolved is not None:
                self.update(resolved)

        # Backfill with the correct defaults.
        defaults = ir.get_tls_defaults(self.name) or {}

        for key in defaults:
            if key not in self:
                self[key] = defaults[key]

        self.logger.debug("IREnvoyTLS setup %s" % self.as_json())

        return True

#############################################################################
## IRAmbassadorTLS represents an Ambassador TLS configuration, from which we
## can create Envoy TLS configurations.


class IRAmbassadorTLS (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.tlsmodule",
                 kind: str="IRTLSModule",
                 name: str="ir.tlsmodule",
                 enabled: bool=True,

                 **kwargs) -> None:
        """
        Initialize an IRAmbassadorTLS from the raw fields of its Resource.
        """

        ir.logger.debug("IRAmbassadorTLS __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            enabled=enabled,
            **kwargs
        )

class TLSModuleFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        assert(ir)

        tls_module = aconf.get_module('tls')

        if tls_module:
            ir.logger.debug("TLSModuleFactory saving TLS module: %s" % tls_module.as_json())

            # XXX What a hack. IRAmbassadorTLS.from_resource() should be able to make
            # this painless.
            new_args = dict(tls_module.as_dict())
            new_rkey = new_args.pop('rkey', tls_module.rkey)
            new_kind = new_args.pop('kind', tls_module.kind)
            new_name = new_args.pop('name', tls_module.name)
            new_location = new_args.pop('location', tls_module.location)

            ir.tls_module = IRAmbassadorTLS(ir, aconf,
                                            rkey=new_rkey,
                                            kind=new_kind,
                                            name=new_name,
                                            location=new_location,
                                            **new_args)

            ir.logger.debug("TLSModuleFactory saved TLS module: %s" % ir.tls_module.as_json())

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass

#     @staticmethod
#     def tmod_certs_exist(tmod):
#         """
#         Returns the number of certs that are defined in the supplied tmod
#
#         :param tmod: The TLS module configuration
#         :return: number of certs in tmod
#         :rtype: int
#         """
#         cert_count = 0
#         if tmod.get('cert_chain_file') is not None:
#             cert_count += 1
#         if tmod.get('private_key_file') is not None:
#             cert_count += 1
#         if tmod.get('cacert_chain_file') is not None:
#             cert_count += 1
#         return cert_count
