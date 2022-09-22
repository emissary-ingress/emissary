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
from typing import TYPE_CHECKING, Callable, Dict, List, Optional, Union
from typing import cast as typecast

from ...ir.irtlscontext import IRTLSContext

# This stuff isn't really accurate, but it'll do for now.
#
# XXX It's also a sure sign that this crap needs to be a proper class
# structure instead of nested dicts of dicts of unions of dicts of .... :P

EnvoyCoreSource = Dict[str, str]

EnvoyTLSCert = Dict[str, EnvoyCoreSource]
ListOfCerts = List[EnvoyTLSCert]

EnvoyValidationElements = Union[EnvoyCoreSource, bool]
EnvoyValidationContext = Dict[str, EnvoyValidationElements]

EnvoyTLSParams = Dict[str, Union[str, List[str]]]

EnvoyCommonTLSElements = Union[List[str], ListOfCerts, EnvoyValidationContext, EnvoyTLSParams]
EnvoyCommonTLSContext = Dict[str, EnvoyCommonTLSElements]

ElementHandler = Callable[[str, str], None]


class V2TLSContext(Dict):
    TLSVersionMap = {
        "v1.0": "TLSv1_0",
        "v1.1": "TLSv1_1",
        "v1.2": "TLSv1_2",
        "v1.3": "TLSv1_3",
    }

    def __init__(
        self, ctx: Optional[IRTLSContext] = None, host_rewrite: Optional[str] = None
    ) -> None:
        del host_rewrite  # quiesce warning

        super().__init__()

        self.is_fallback = False

        if ctx:
            self.add_context(ctx)

    def get_common(self) -> EnvoyCommonTLSContext:
        return self.setdefault("common_tls_context", {})

    def get_params(self) -> EnvoyTLSParams:
        common = self.get_common()

        # This boils down to "params = common.setdefault('tls_params', {})" with typing.
        empty_params = typecast(EnvoyTLSParams, {})
        params = typecast(EnvoyTLSParams, common.setdefault("tls_params", empty_params))

        return params

    def get_certs(self) -> ListOfCerts:
        common = self.get_common()

        # We have to explicitly cast this empty list to a list of strings.
        empty_cert_list: List[str] = []
        cert_list = common.setdefault("tls_certificates", empty_cert_list)

        # cert_list is of type EnvoyCommonTLSElements right now, so we need to cast it.
        return typecast(ListOfCerts, cert_list)

    def update_cert_zero(self, key: str, value: str) -> None:
        certs = self.get_certs()

        if not certs:
            certs.append({})

        src: EnvoyCoreSource = {"filename": value}
        certs[0][key] = src

    def update_alpn(self, key: str, value: str) -> None:
        common = self.get_common()
        common[key] = [value]

    def update_tls_version(self, key: str, value: str) -> None:
        params = self.get_params()

        if value in V2TLSContext.TLSVersionMap:
            params[key] = V2TLSContext.TLSVersionMap[value]

    def update_tls_cipher(self, key: str, value: List[str]) -> None:
        params = self.get_params()

        params[key] = value

    def update_validation(self, key: str, value: str) -> None:
        empty_context: EnvoyValidationContext = {}

        # This looks weirder than you might expect, because self.get_common().setdefault() is a truly
        # crazy Union type, so we need to cast it to an EnvoyValidationContext to be able to work
        # with it.
        validation = typecast(
            EnvoyValidationContext,
            self.get_common().setdefault("validation_context", empty_context),
        )

        src: EnvoyCoreSource = {"filename": value}
        validation[key] = src

    def add_context(self, ctx: IRTLSContext) -> None:
        if TYPE_CHECKING:
            # This is needed because otherwise self.__setitem__ confuses things.
            handler: Callable[[str, str], None]  # pragma: no cover

        if ctx.is_fallback:
            self.is_fallback = True

        for secretinfokey, handler, hkey in [
            ("cert_chain_file", self.update_cert_zero, "certificate_chain"),
            ("private_key_file", self.update_cert_zero, "private_key"),
            ("cacert_chain_file", self.update_validation, "trusted_ca"),
        ]:
            if secretinfokey in ctx["secret_info"]:
                handler(hkey, ctx["secret_info"][secretinfokey])

        for ctxkey, handler, hkey in [
            ("alpn_protocols", self.update_alpn, "alpn_protocols"),
            ("cert_required", self.__setitem__, "require_client_certificate"),
            ("min_tls_version", self.update_tls_version, "tls_minimum_protocol_version"),
            ("max_tls_version", self.update_tls_version, "tls_maximum_protocol_version"),
            ("sni", self.__setitem__, "sni"),
        ]:
            value = ctx.get(ctxkey, None)

            if value is not None:
                handler(hkey, value)

        # This is a separate loop because self.update_tls_cipher is not the same type
        # as the other updaters: it takes a list of strings for the value, not a single
        # string. Getting mypy to be happy with that is _annoying_.

        for ctxkey, list_handler, hkey in [
            ("cipher_suites", self.update_tls_cipher, "cipher_suites"),
            ("ecdh_curves", self.update_tls_cipher, "ecdh_curves"),
        ]:
            value = ctx.get(ctxkey, None)

            if value is not None:
                list_handler(hkey, value)

    def pretty(self) -> str:
        common_ctx = self.get("common_tls_context", {})
        certs = common_ctx.get("tls_certificates", [])
        cert0 = certs[0] if certs else {}
        chain0 = cert0.get("certificate_chain", {})
        filename = chain0.get("filename", None)

        if filename:
            basename = os.path.basename(filename)[0:8] + "..."
            dirname = os.path.basename(os.path.dirname(filename))
            filename = f".../{dirname}/{basename}"

        return "<V2TLSContext%s chain_file %s>" % (
            " (fallback)" if self.is_fallback else "",
            filename,
        )
