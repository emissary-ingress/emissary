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

from typing import Callable, Dict, List, Optional, Union
from typing import cast as typecast

from ...ir.irtlscontext import IRTLSContext

# This stuff isn't really accurate, but it'll do for now.
EnvoyCoreSource = Dict[str, str]

EnvoyTLSCert = Dict[str, EnvoyCoreSource]
ListOfCerts = List[EnvoyTLSCert]

EnvoyValidationElements = Union[EnvoyCoreSource, bool]
EnvoyValidationContext = Dict[str, EnvoyValidationElements]

EnvoyCommonTLSElements = Union[List[str], ListOfCerts, EnvoyValidationContext]
EnvoyCommonTLSContext = Dict[str, EnvoyCommonTLSElements]

ElementHandler = Callable[[str, str], None]

class V2TLSContext(Dict):
    def __init__(self, ctx: Optional[IRTLSContext]=None, host_rewrite: Optional[str]=None) -> None:
        super().__init__()

        if ctx:
            self.add_context(ctx)

    def get_common(self) -> EnvoyCommonTLSContext:
        return self.setdefault('common_tls_context', {})

    def get_certs(self) -> ListOfCerts:
        common = self.get_common()

        # We have to explicitly cast this empty list to a list of strings.
        empty_cert_list: List[str] = []
        cert_list = common.setdefault('tls_certificates', empty_cert_list)

        # cert_list is of type EnvoyCommonTLSElements right now, so we need to cast it.
        return typecast(ListOfCerts, cert_list)

    def update_cert_zero(self, key: str, value: str) -> None:
        certs = self.get_certs()

        if not certs:
            certs.append({})

        src: EnvoyCoreSource = { 'filename': value }
        certs[0][key] = src

    def update_alpn(self, key: str, value: str) -> None:
        common = self.get_common()
        common[key] = [ value ]

    def update_tls_version(self, key: str, value: str) -> None:
        common = self.get_common()
        common.setdefault('tls_params', {})
        if value == "v1.0":
            common['tls_params'][key] = "TLSv1_0"
        if value == "v1.1":
            common['tls_params'][key] = "TLSv1_1"
        if value == "v1.2":
            common['tls_params'][key] = "TLSv1_2"
        if value == "v1.3":
            common['tls_params'][key] = "TLSv1_3"

    def update_validation(self, key: str, value: str) -> None:
        empty_context: EnvoyValidationContext = {}
        validation = typecast(EnvoyValidationContext, self.get_common().setdefault('validation_context', empty_context))

        src: EnvoyCoreSource = { 'filename': value }
        validation[key] = src

    def add_context(self, ctx: IRTLSContext) -> None:
        handler: ElementHandler = self.__setitem__

        for secretinfokey, handler, hkey in [
            ( 'cert_chain_file', self.update_cert_zero, 'certificate_chain' ),
            ( 'private_key_file', self.update_cert_zero, 'private_key' ),
            ( 'cacert_chain_file', self.update_validation, 'trusted_ca' ),
        ]:
            if secretinfokey in ctx['secret_info']:
                handler(hkey, ctx['secret_info'][secretinfokey])

        for ctxkey, handler, hkey in [
            ( 'alpn_protocols', self.update_alpn, 'alpn_protocols' ),
            ( 'cert_required', self.__setitem__, 'require_client_certificate' ),
            ( 'min_tls_version', self.update_tls_version, 'tls_minimum_protocol_version' ),
            ( 'max_tls_version', self.update_tls_version, 'tls_maximum_protocol_version' ),
        ]:
            value = ctx.get(ctxkey, None)

            if value is not None:
                handler(hkey, value)
