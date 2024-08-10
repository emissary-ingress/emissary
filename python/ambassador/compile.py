# Copyright 2020 Datawire. All rights reserved.
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

import logging
from typing import Optional

from typing_extensions import NotRequired, TypedDict

from .cache import Cache
from .config import Config
from .envoy import EnvoyConfig
from .fetch import ResourceFetcher
from .ir import IR
from .ir.ir import IRFileChecker
from .utils import NullSecretHandler, SecretHandler


class _CompileResult(TypedDict):
    ir: IR
    xds: NotRequired[EnvoyConfig]


def null_file_checker(path: str) -> bool:
    return True


def Compile(
    logger: logging.Logger,
    input_text: str,
    cache: Optional[Cache] = None,
    file_checker: Optional[IRFileChecker] = None,
    secret_handler: Optional[SecretHandler] = None,
    k8s: bool = False,
) -> _CompileResult:
    """
    Compile is a helper function to take a bunch of YAML and compile it into an
    IR and, optionally, an Envoy config.

    The output is a dictionary:

    {
        "ir": the IR data structure
        "xds": the Envoy config
    }

    :param input_text: The input text (WATT snapshot JSON or K8s YAML per 'k8s')
    :param k8s: If true, input_text is K8s YAML, otherwise it's WATT snapshot JSON
    :param ir: Generate the IR IFF True
    """

    if not file_checker:
        file_checker = null_file_checker

    if not secret_handler:
        secret_handler = NullSecretHandler(logger, None, None, "fake")

    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)

    if k8s:
        fetcher.parse_yaml(input_text, k8s=True)
    else:
        fetcher.parse_watt(input_text)

    aconf.load_all(fetcher.sorted())

    ir = IR(aconf, cache=cache, file_checker=file_checker, secret_handler=secret_handler)

    out: _CompileResult = {"ir": ir}

    if ir:
        out["xds"] = EnvoyConfig.generate(ir, cache=cache)

    return out
