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

from typing import TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus, TLSPaths
from ..resource import Resource
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

        return ir.save_tls_context(self.name, self)

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

        # print("IRAmbassadorTLS __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            enabled=enabled,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config):
        assert(ir)

        # Check for the tls and tls-from-ambassador-certs Modules...
        tls_module = aconf.get_module('tls')
        generated_module = aconf.get_module('tls-from-ambassador-certs')

        # ...and merge them into a single set of configs.
        tls_module = self.merge_tmods(tls_module, generated_module, 'server')
        tls_module = self.merge_tmods(tls_module, generated_module, 'client')

        # OK, done. Merge the result back in.
        if tls_module:
            self.logger.debug("tmod after merge: %s" % json.dumps(tls_module, indent=4))
            self.update(tls_module)
            self.logger.debug("TLS module after merge: %s" % json.dumps(self, indent=4))

            # Create TLS contexts.
            for ctx_name in tls_module.keys():
                IREnvoyTLS(ir=ir, aconf=aconf, name=ctx_name, **tls_module[ctx_name ])

            return True
        else:
            return False

    def merge_tmods(self, tls_module: Resource, generated_module: Resource, key: str) -> Resource:
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
            self.logger.debug("tls_module %s" % json.dumps(tls_module, indent=4))
            self.logger.debug("generated_module %s" % json.dumps(generated_module, indent=4))

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
                        self.post_error(RichStatus.fromError(errstr))
                    else:
                        # They have the same value. Worth mentioning in debug.
                        self.logger.debug("merge_tmods: {}.{} duplicated with same value".format(key, ckey))
                else:
                    # ckey only exists in gen_aconf. Copy it over.
                    self.logger.debug("merge_tmods: copy {}.{} from gen_config".format(key, ckey))
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


#     def service_tls_check(self, svc: str, context: Optional[Union[str, bool]], host_rewrite: bool) -> ServiceInfo:
#         """
#         Uniform handling of service definitions, TLS origination, etc.
#
#         Here's how it goes:
#         - If the service starts with https://, it is forced to originate TLS.
#         - Else, if it starts with http://, it is forced to _not_ originate TLS.
#         - Else, if the context is the boolean value True, it will originate TLS.
#
#         After figuring that out, if we have a context which is a string value,
#         we try to use that context name to look up certs to use. If we can't
#         find any, we won't send any originating cert.
#
#         Finally, if no port is present in the service already, we force port 443
#         if we're originating TLS, 80 if not.
#
#         :param svc: URL of the service (from the Ambassador Mapping)
#         :param context: TLS context name, or True to originate TLS but use no certs
#         :param host_rewrite: Is host rewriting active?
#         """
#
#         originate_tls: Union[str, bool] = False
#         name_fields: List[str] = []
#
#         if svc.lower().startswith("http://"):
#             originate_tls = False
#             svc = svc[len("http://"):]
#         elif svc.lower().startswith("https://"):
#             originate_tls = True
#             name_fields = [ 'otls' ]
#             svc = svc[len("https://"):]
#         elif context is True:
#             originate_tls = True
#             name_fields = [ 'otls' ]
#
#         # Separate if here because you need to be able to specify a context
#         # even after you say "https://" for the service.
#
#         if context and (context is not True):
#             # We know that context is a string here.
#             if context in self.tls_contexts:
#                 name_fields = [ 'otls', typecast(str, context) ]
#                 originate_tls = typecast(str, context)
#             else:
#                 self.logger.error("Originate-TLS context %s is not defined" % context)
#
#         if originate_tls and host_rewrite:
#             name_fields.append("hr-%s" % host_rewrite)
#
#         port = 443 if originate_tls else 80
#         context_name = typecast(str, "_".join(name_fields) if name_fields else None)
#
#         svc_url = 'tcp://%s' % svc
#
#         if ':' not in svc:
#             svc_url = '%s:%d' % (svc_url, port)
#
#         return (svc, svc_url, bool(originate_tls), context_name)
