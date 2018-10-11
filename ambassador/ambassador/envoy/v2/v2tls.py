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

from typing import Dict, Optional

from ...ir.irtls import IREnvoyTLS


class V2TLSContext(Dict):
    def __init__(self, ctx: Optional[IREnvoyTLS]=None, host_rewrite: Optional[str]=None) -> None:
        super().__init__()

        if ctx:
            self.add_context(ctx)

        # if host_rewrite:
        #     self['sni'] = host_rewrite

    def add_context(self, ctx: IREnvoyTLS) -> None:
        common_tls_context = {}

        tls_certificate = {}
        if "cert_chain_file" in ctx:
            tls_certificate["certificate_chain"] = {
                "filename": ctx["cert_chain_file"]
            }
        if "private_key_file" in ctx:
            tls_certificate["private_key"] = {
                "filename": ctx["private_key_file"]
            }

        if tls_certificate:
            if "tls_certificates" not in common_tls_context:
                common_tls_context.update({"tls_certificates": []})

            common_tls_context["tls_certificates"].append(tls_certificate)

        if "alpn_protocols" in ctx:
            common_tls_context["alpn_protocols"] = ctx["alpn_protocols"]

        if "cacert_chain_file" in ctx:
            if "validation_context" not in common_tls_context:
                common_tls_context.update({"validation_context": {}})

            common_tls_context["validation_context"]["trusted_ca"] = {
                "filename": ctx["cacert_chain_file"]
            }

        if "cert_required" in ctx:
            self["require_client_certificate"] = ctx["cert_required"]

        if len(common_tls_context) > 0:
            self.update({"common_tls_context": common_tls_context})
