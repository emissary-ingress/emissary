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

from ...ir.irtlscontext import IRTLSContext


class V1TLSContext(Dict[str, str]):
    def __init__(self, ctx: IRTLSContext, host_rewrite: Optional[str]=None) -> None:
        super().__init__()

        if 'secret_info' in ctx:
            for ir_key, v1_key in [
                ("cert_chain_file", "cert_chain_file"),
                ("private_key_file", "private_key_file"),
                ("cacert_chain_file", "ca_cert_file")
            ]:
                if ctx['secret_info'].get(ir_key, None):
                    self[v1_key] = ctx['secret_info'][ir_key]

        for ir_key, v1_key in [
            ("alpn_protocols", "alpn_protocols"),
            ("cert_required", "require_client_certificate")
        ]:
            if ctx.get(ir_key, None):
                self[v1_key] = ctx[ir_key]

        if host_rewrite:
            self['sni'] = host_rewrite

