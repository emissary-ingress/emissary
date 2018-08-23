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

from typing import List, TYPE_CHECKING

from ...ir.irlistener import IRListener

if TYPE_CHECKING:
    from . import V1Config


class V1Listener(dict):
    def __init__(self, config: 'V1Config', listener: IRListener) -> None:
        super().__init__()

        self["address"] = "tcp://0.0.0.0:%d" % listener.service_port

        if listener.use_proxy_proto:
            self["use_proxy_proto"] = True

        if "use_remote_address" in listener:
            self["use_remote_address"] = listener.use_remote_address

        if 'tls_context' in listener:
            ctx = listener.tls_context

            if ctx:
                lctx = {
                    "cert_chain_file": ctx.cert_chain_file,
                    "private_key_file": ctx.private_key_file
                }

                if "alpn_protocols" in ctx:
                    lctx["alpn_protocols"] = ctx["alpn_protocols"]

                if "cacert_chain_file" in ctx:
                    lctx["cacert_chain_file"] = ctx["cacert_chain_file"]

                if "cert_required" in ctx:
                    lctx["require_client_certificate"] = ctx["cert_required"]

        self["filters"] = []

    @classmethod
    def generate(self, config: 'V1Config') -> List['V1Listener']:
        listeners: List['V1Listener'] = []

        for listener in config.ir.listeners:
            listeners.append(V1Listener(config, listener))

        return listeners
