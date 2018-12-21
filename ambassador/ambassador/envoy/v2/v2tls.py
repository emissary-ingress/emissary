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

from typing import Dict, Optional, Union

from ...ir.irtls import IREnvoyTLS
from ...ir.irtlscontext import IRTLSContext


class V2TLSContext(Dict):
    def __init__(self, ctx: Optional[Union[IREnvoyTLS, IRTLSContext]]=None, host_rewrite: Optional[str]=None) -> None:
        super().__init__()

        if ctx:
            self.add_context(ctx)

    def get_common(self) -> Dict[str, str]:
        return self.setdefault('common_tls_context', {})

    def get_certs(self) -> Dict[str, str]:
        common = self.get_common()
        return common.setdefault('tls_certificates', [])

    def update_cert_zero(self, key, value) -> None:
        certs = self.get_certs()

        if not certs:
            certs.append({})

        certs[0][key] = { 'filename': value }

    def update_common(self, key, value) -> None:
        common = self.get_common()
        common[key] = value

    def update_alpn(self, key, value) -> None:
        common = self.get_common()
        common[key] = [ value ]

    def update_validation(self, key, value) -> None:
        validation: Dict[str, str] = self.get_common().setdefault('validation_context', {})
        validation[key] = { 'filename': value }

    def add_context(self, ctx: Union[IREnvoyTLS, IRTLSContext]) -> None:
        # This is a weird method, because the definition of a V2 TLS context in
        # Envoy is weird, and because we need to manage two different inputs (which
        # is silly).

        if ctx.kind == 'IREnvoyTLS':
            for ctxkey, handler, hkey in [
                ( 'certificate_chain_file', self.update_cert_zero, 'certificate_chain' ),
                ( 'private_key_file', self.update_cert_zero, 'private_key' ),
                ( 'cacert_chain_file', self.update_validation, 'trusted_ca' ),
            ]:
                value = ctx.get(ctxkey, None)

                if value is not None:
                    handler(hkey, value)
        elif ctx.kind == 'IRTLSContext':
            for secretinfokey, handler, hkey in [
                ( 'cert_chain_file', self.update_cert_zero, 'certificate_chain' ),
                ( 'private_key_file', self.update_cert_zero, 'private_key' ),
                ( 'cacert_chain_file', self.update_validation, 'trusted_ca' ),
            ]:
                if secretinfokey in ctx['secret_info']:
                    handler(hkey, ctx['secret_info'][secretinfokey])
        else:
            raise TypeError("impossible? error: V2TLS handed a %s" % ctx.kind)

        for ctxkey, handler, hkey in [
            ( 'alpn_protocols', self.update_alpn, 'alpn_protocols' ),
            ( 'cert_required', self.__setitem__, 'require_client_certificate' ),
        ]:
            value = ctx.get(ctxkey, None)

            if value is not None:
                handler(hkey, value)
