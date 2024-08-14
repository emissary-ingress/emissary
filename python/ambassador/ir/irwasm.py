# Copyright 2023 Datawire. All rights reserved.
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
from dataclasses import asdict, dataclass
from typing import TYPE_CHECKING, Any, Dict, Optional

from ..config import Config
from .irfilter import IRFilter

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


def is_wasm_enabled() -> bool:
    env_ext_proc_enabled = os.getenv("WASM_FILTER_ENABLED")
    return bool(env_ext_proc_enabled)


def wasm_file_exists() -> bool:
    return os.path.exists("/ambassador/mywasmfilter.wasm")


@dataclass
class WASMFilterConfig:
    wasm_file_path: str
    enabled: bool


class IRWASMFilter(IRFilter):
    config: WASMFilterConfig

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.wasm_filter",
        kind: str = "IRWASMFilter",
        name: str = "wasm_filter",
        namespace: Optional[str] = None,
    ) -> None:
        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            kind=kind,
            name=name,
            namespace=namespace,
            type="decoder",
        )

    def setup(self, ir: "IR", _: Config) -> bool:
        if not is_wasm_enabled():
            self.logger.error("wasm filter not enabled, skipping")
            self.config = WASMFilterConfig(
                enabled=False, wasm_file_path="/ambassador/mywasmfilter.wasm"
            )
            return True
        if not wasm_file_exists():
            self.logger.error(
                "%s not found, envoy configuration will fail to apply",
                "/ambassador/mywasmfilter.wasm",
            )
            self.config = WASMFilterConfig(
                enabled=False, wasm_file_path="/ambassador/mywasmfilter.wasm"
            )
            return True
        self.config = WASMFilterConfig(enabled=True, wasm_file_path="/ambassador/mywasmfilter.wasm")
        return True

    def config_dict(self) -> Optional[Dict[str, Any]]:
        return asdict(self.config) if self.config else None

    def as_dict(self) -> Dict[str, Any]:
        d = super(IRWASMFilter, self).as_dict()
        d["config"] = self.config_dict() if self.config else {}
        return d
