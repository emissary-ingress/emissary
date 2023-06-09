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

GO_FILTER_LIBRARY_PATH = "/ambassador/filter.so"
AMBASSADOR_DISABLE_GO_FILTER = os.getenv("AMBASSADOR_DISABLE_GO_FILTER", False)


def go_library_exists(go_library_path: str) -> bool:
    if os.path.exists(go_library_path):
        return True
    return False


@dataclass
class GOFilterConfig:
    library_path: str


class IRGOFilter(IRFilter):
    config: GOFilterConfig

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.go_filter",
        kind: str = "IRGOFilter",
        name: str = "go_filter",
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
        if ir.edge_stack_allowed:
            if AMBASSADOR_DISABLE_GO_FILTER:
                self.logger.info(
                    "AMBASSADOR_DISABLE_GO_FILTER=%s, disabling Go filter...",
                    AMBASSADOR_DISABLE_GO_FILTER,
                )
                return False
            if not go_library_exists(GO_FILTER_LIBRARY_PATH):
                self.logger.error("%s not found, disabling Go filter...", GO_FILTER_LIBRARY_PATH)
                return False
            self.config = GOFilterConfig(library_path=GO_FILTER_LIBRARY_PATH)
            return True
        return False

    def config_dict(self) -> Optional[Dict[str, Any]]:
        return asdict(self.config) if self.config else None

    def as_dict(self) -> Dict[str, Any]:
        d = super(IRGOFilter, self).as_dict()
        d["config"] = self.config_dict() if self.config else {}
        return d
