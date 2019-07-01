#!/usr/bin/python

from ambassador.utils import ParsedService as Service

from typing import Dict, List, Optional, Tuple, TYPE_CHECKING
from typing import cast as typecast

import sys

import json
import logging
import os

from urllib.parse import urlparse

loglevel = logging.INFO

args = sys.argv[1:]

if args:
    if args[0] == '--debug':
        loglevel = logging.DEBUG
        args.pop(0)
    elif args[0].startswith('--'):
        raise Exception(f'Usage: {os.path.basename(sys.argv[0])} [--debug] [path]')

logging.basicConfig(
    level=loglevel,
    format="%(asctime)s watch-hook %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

alogger = logging.getLogger('ambassador')
alogger.setLevel(loglevel)

logger = logging.getLogger('watch_hook')
logger.setLevel(loglevel)

from ambassador import Config, IR
from ambassador.config.resourcefetcher import ResourceFetcher
from ambassador.ir.irserviceresolver import IRServiceResolverFactory
from ambassador.ir.irtls import TLSModuleFactory, IRAmbassadorTLS
from ambassador.ir.irambassador import IRAmbassador
from ambassador.utils import SecretInfo, SavedSecret, SecretHandler

if TYPE_CHECKING:
    from ambassador.ir.irtlscontext import IRTLSContext
    from ambassador.ir.irresource import IRResource


# Fake SecretHandler for our fake IR, below.

class SecretRecorder(SecretHandler):
    def __init__(self, logger: logging.Logger) -> None:
        super().__init__(logger, "-source_root-", "-cache_dir-", "0")
        self.needed: Dict[Tuple[str, str], SecretInfo] = {}

    # Record what was requested, and always return True.
    def load_secret(self, context: 'IRTLSContext',
                    secret_name: str, namespace: str) -> Optional[SecretInfo]:
        secret_key = ( secret_name, namespace )

        if secret_key not in self.needed:
            self.needed[secret_key] = SecretInfo(secret_name, namespace, '-crt-', '-key-', decode_b64=False)

        return self.needed[secret_key]

    # Never cache anything.
    def cache_secret(self, context: 'IRTLSContext', secret_info: SecretInfo):
        return SavedSecret(secret_info.name, secret_info.namespace, '-crt-path-', '-key-path-',
                           { 'tls_crt': '-crt-', 'tls_key': '-key-' })


# XXX Sooooo there's some ugly stuff here.
#
# We need to do a little bit of the same work that the IR does for things like
# managing Resolvers and parsing service names. However, we really don't want to
# do all the work of instantiating an IR.
#
# So we kinda fake it. And yeah, it's kind of disgusting.

class FakeIR(IR):
    def __init__(self, logger, aconf):
        self.ambassador_id = Config.ambassador_id
        self.ambassador_namespace = Config.ambassador_namespace
        self.ambassador_nodename = aconf.ambassador_nodename

        self.logger = logger
        self.aconf = aconf

        # If we're asked about a secret, record interest in that secret.
        self.secret_handler = SecretRecorder(self.logger)

        # If we're asked about a file, it's good.
        self.file_checker = lambda path: True

        self.clusters = {}
        self.grpc_services = {}
        self.filters = []
        self.tls_contexts = {}
        self.tls_module = None
        self.listeners = []
        self.groups = {}
        self.resolvers = {}
        self.breakers = {}
        self.outliers = {}
        self.services = {}
        self.tracing = None
        self.ratelimit = None
        self.saved_secrets = {}
        self.secret_info = {}

        self.ambassador_module = IRAmbassador(self, aconf)

        IRServiceResolverFactory.load_all(self, aconf)
        TLSModuleFactory.load_all(self, aconf)
        self.save_tls_contexts(aconf)

        self.ambassador_module.finalize(self, aconf)

    # Don't bother actually saving resources that come up when working with
    # the faked modules.
    def save_resource(self, resource: 'IRResource') -> 'IRResource':
        return resource


#### Mainline.

yaml_stream = sys.stdin

if args:
    yaml_stream = open(args[0], "r")

aconf = Config()
fetcher = ResourceFetcher(logger, aconf)
fetcher.parse_watt(yaml_stream.read())

aconf.load_all(fetcher.sorted())

# We can lift mappings straight from the aconf...
mappings = aconf.get_config('mappings') or {}

# ...but we need the fake IR to deal with resolvers and TLS contexts.
fake = FakeIR(logger, aconf)

logger.debug("FakeIR: %s" % fake.as_json())

resolvers = fake.resolvers
contexts = fake.tls_contexts

logger.debug(f'mappings: {len(mappings)}')
logger.debug(f'resolvers: {len(resolvers)}')
logger.debug(f'contexts: {len(contexts)}')

consul_watches = []
kube_watches = []

global_resolver = fake.ambassador_module.get('resolver', None)

for mname, mapping in mappings.items():
    res_name = mapping.get('resolver', None)
    res_source = 'mapping'

    if not res_name:
        res_name = global_resolver
        res_source = 'defaults'

    ctx_name = mapping.get('tls', None)

    logger.debug(f'Mapping {mname}: resolver {res_name} from {res_source}, service {mapping.service}, tls {ctx_name}')

    if res_name:
        resolver = resolvers.get(res_name, None)
        logger.debug(f'-> resolver {resolver}')

        if resolver:
            svc = Service(logger, mapping.service, ctx_name)

            if resolver.kind == 'ConsulResolver':
                logger.debug(f'Mapping {mname} uses Consul resolver {res_name}')

                # At the moment, we stuff the resolver's datacenter into the association
                # ID for this watch. The ResourceFetcher relies on that.

                consul_watches.append(
                    {
                        "id": resolver.datacenter,
                        "consul-address": resolver.address,
                        "datacenter": resolver.datacenter,
                        "service-name": svc.hostname
                    }
                )
            elif resolver.kind == 'KubernetesEndpointResolver':
                host = svc.hostname
                namespace = Config.ambassador_namespace

                if "." in host:
                    (host, namespace) = host.split(".", 2)[0:2]

                logger.debug(f'...kube endpoints: svc {svc.hostname} -> host {host} namespace {namespace}')

                kube_watches.append(
                    {
                        "kind": "endpoints",
                        "namespace": namespace,
                        "field-selector": f'metadata.name={host}'
                    }
                )

for secret_key, secret_info in fake.secret_handler.needed.items():
    logger.debug(f'need secret {secret_info.name}.{secret_info.namespace}')

    kube_watches.append(
        {
            "kind": "secret",
            "namespace": secret_info.namespace,
            "field-selector": f'metadata.name={secret_info.name}'
        }
    )

# kube_watches.append(
#     {
#         "kind": "secret",
#         "namespace": Config.ambassador_namespace if Config.single_namespace else "",
#         "field-selector": "metadata.namespace!=kube-system,type!=kubernetes.io/service-account-token"
#     }
# )

watchset = {
    "kubernetes-watches": kube_watches,
    "consul-watches": consul_watches
}

save_dir = os.environ.get('AMBASSADOR_WATCH_DIR', '/tmp')

if save_dir:
    json.dump(watchset, open(os.path.join(save_dir, 'watch.json'), "w"))

json.dump(watchset, sys.stdout)
