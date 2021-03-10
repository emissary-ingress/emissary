from typing import Dict, Optional, Tuple, TYPE_CHECKING

import logging
import sys

logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s test-dump %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

logger = logging.getLogger('ambassador')
logger.setLevel(logging.DEBUG)

import json

from ambassador import Config, IR
from ambassador.envoy import V2Config

from ambassador.utils import SecretInfo, SavedSecret, SecretHandler
from ambassador.fetch import ResourceFetcher

if TYPE_CHECKING:
    from ambassador.ir.irtlscontext import IRTLSContext


class SecretRecorder(SecretHandler):
    def __init__(self, logger: logging.Logger) -> None:
        super().__init__(logger, "-source_root-", "-cache_dir-")
        self.needed: Dict[Tuple[str, str], SecretInfo] = {}

    # Record what was requested, and always return True.
    def load_secret(self, context: 'IRTLSContext',
                    secret_name: str, namespace: str) -> Optional[SecretInfo]:
        self.logger.debug(f"SecretRecorder: Trying to load secret {secret_name} in namespace {namespace} from TLSContext {context}")
        secret_key = ( secret_name, namespace )

        if secret_key not in self.needed:
            self.needed[secret_key] = SecretInfo(secret_name, namespace, '-crt-', '-key-', decode_b64=False)

        return self.needed[secret_key]

    # Never cache anything.
    def cache_secret(self, context: 'IRTLSContext', secret_info: SecretInfo):
        return SavedSecret(secret_info.name, secret_info.namespace, '-crt-path-', '-key-path-',
                           { 'tls_crt': '-crt-', 'tls_key': '-key-' })


scc = SecretHandler(logger, "test-dump", "ss")

yamlpath = sys.argv[1] if len(sys.argv) > 1 else "consul-3.yaml"

aconf = Config()
fetcher = ResourceFetcher(logger, aconf)
fetcher.parse_watt(open(yamlpath, "r").read())

aconf.load_all(fetcher.sorted())

open("test-aconf.json", "w").write(aconf.as_json())

# sys.exit(0)

ir = IR(aconf, secret_handler=scc)

open("test-ir.json", "w").write(ir.as_json())

econf = V2Config(ir)

open("test-v2.json", "w").write(econf.as_json())
