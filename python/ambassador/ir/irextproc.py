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


def is_extproc_enabled() -> bool:
    env_ext_proc_enabled = os.getenv("EXT_PROC_ENABLED")
    return bool(env_ext_proc_enabled)


@dataclass
class ExtProcFilterConfig:
    enabled: bool
    allow_mode_override: bool
    failure_mode_allow: bool


class IRExtProcFilter(IRFilter):
    config: ExtProcFilterConfig

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.ext_proc_filter",
        kind: str = "IRExtProcFilter",
        name: str = "ext_proc_filter",
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

    # We want to enable this filter only in Edge Stack
    def setup(self, ir: "IR", _: Config) -> bool:
        if not is_extproc_enabled():
            self.logger.error("ext proc filter not enabled, skipping")
            self.config = ExtProcFilterConfig(
                enabled=False, allow_mode_override=False, failure_mode_allow=False
            )
            return True
        self.config = ExtProcFilterConfig(
            enabled=True, allow_mode_override=False, failure_mode_allow=False
        )
        return True

    def config_dict(self) -> Optional[Dict[str, Any]]:
        return asdict(self.config) if self.config else None

    def as_dict(self) -> Dict[str, Any]:
        d = super(IRExtProcFilter, self).as_dict()
        d["config"] = self.config_dict() if self.config else {}
        return d
