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

import json
import os

from typing import Optional, TYPE_CHECKING
from typing import cast as typecast

from ..config import Config
from ..utils import RichStatus, TLSPaths
from ..config import ACResource
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

        # print("IREnvoyTLS __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            enabled=enabled,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config):
        if not self.enabled:
            return False

        # Backfill with the correct defaults.
        defaults = ir.get_tls_defaults(self.name) or {}

        for key in defaults:
            if key not in self:
                self[key] = defaults[key]

        # # Check if secrets are supplied for TLS termination and/or TLS auth
        # secret = context.get('secret')
        # if secret is not None:
        #     self.logger.debug("config.server.secret is {}".format(secret))
        #     # If /{etc,ambassador}/certs/tls.crt does not exist, then load the secrets
        #     if check_cert_file(TLSPaths.mount_tls_crt.value):
        #         self.logger.debug("Secret already exists, taking no action for secret {}".format(secret))
        #     elif check_cert_file(TLSPaths.tls_crt.value):
        #         self.cert_chain_file = TLSPaths.tls_crt.value
        #         self.private_key_file = TLSPaths.tls_key.value
        #     else:
        #         (server_cert, server_key, server_data) = read_cert_secret(kube_v1(), secret, self.namespace)
        #         if server_cert and server_key:
        #             self.logger.debug("saving contents of secret {} to {}".format(
        #                 secret, TLSPaths.cert_dir.value))
        #             save_cert(server_cert, server_key, TLSPaths.cert_dir.value)
        #             self.cert_chain_file = TLSPaths.tls_crt.value
        #             self.private_key_file = TLSPaths.tls_key.value

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

        # Check for the tls and tls-from-ambassador-certs Modules...
        tls_module = aconf.get_module('tls')
        generated_module = aconf.get_module('tls-from-ambassador-certs')

        # ...and merge them into a single set of configs.
        tls_module = cls.merge_tmods(ir, tls_module, generated_module, 'server')
        tls_module = cls.merge_tmods(ir, tls_module, generated_module, 'client')

        # OK, done. Merge the result back in.
        if tls_module:
            # ir.logger.debug("TLSModuleFactory saving TLS module: %s" % tls_module.as_json())

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

            # ir.logger.debug("TLSModuleFactory saved TLS module: %s" % ir.tls_module.as_json())

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass

    @staticmethod
    def merge_tmods(ir: 'IR',
                    tls_module: Optional[ACResource],
                    generated_module: Optional[ACResource],
                    key: str) -> Optional[ACResource]:
        """
        Merge TLS module configuration for a particular key. In the event of conflicts, the
        tls_module element wins, and an error is posted so that the diagnostics service can
        show it.

        Returns a TLS module with a correctly-merged config element. This will be the
        tls_module (possibly modified) unless no tls_module is present, in which case
        the generated_module will be promoted. If any changes were made to the module, it
        will be marked as referenced by the generated_module.

        :param tls_module: the `tls` module; may be None
        :param generated_module: the `tls-from-ambassador-certs` module; may be None
        :param key: the key in the module config to merge
        :return: TLS module object; see above.
        """

        # First up, the easy cases. If either module is missing, return the other.
        # (The other might be None too, of course.)
        if generated_module is None:
            return tls_module
        elif tls_module is None:
            return generated_module
        else:
            if TYPE_CHECKING:
                tls_module = typecast(ACResource, tls_module)
                generated_module = typecast(ACResource, generated_module)

            ir.logger.debug("tls_module %s" % json.dumps(tls_module, indent=4))
            ir.logger.debug("generated_module %s" % json.dumps(generated_module, indent=4))

            # OK, no easy cases. We know that both modules exist: grab the config dicts.
            tls_config = tls_module.get(key, {})
            gen_config = generated_module.get(key, {})

            # Now walk over the tls_config and copy anything needed.
            any_changes = False

            for ckey in gen_config:
                if ckey in tls_config:
                    # ckey exists in both modules. Do they have the same value?
                    if tls_config[ckey] != gen_config[ckey]:
                        # No -- post an error, but let the version from the TLS module win.
                        errfmt = "CONFLICT in TLS config for {}.{}: using {} from TLS module in {}"
                        errstr = errfmt.format(key, ckey, tls_config[ckey], tls_module.location)
                        ir.post_error(RichStatus.fromError(errstr))
                    else:
                        # They have the same value. Worth mentioning in debug.
                        ir.logger.debug("merge_tmods: {}.{} duplicated with same value".format(key, ckey))
                else:
                    # ckey only exists in gen_aconf. Copy it over.
                    ir.logger.debug("merge_tmods: copy {}.{} from gen_config".format(key, ckey))
                    tls_config[ckey] = gen_config[ckey]
                    any_changes = True

            # If we had changes...
            if any_changes:
                # ...then mark the tls_module as referenced by the generated_module's
                # source..
                tls_module.referenced_by(generated_module)

                # ...and copy the tls_config back in (in case the key wasn't in the tls_module
                # config at all originally).
                tls_module[key] = tls_config

            # Finally, return the tls_module.
            return tls_module

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
