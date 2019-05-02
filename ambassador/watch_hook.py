#!/usr/bin/python

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

        IRServiceResolverFactory.load_all(self, aconf)
        TLSModuleFactory.load_all(self, aconf)
        self.save_tls_contexts(aconf)

        self.ambassador_module = IRAmbassador(self, aconf)

    # Don't bother actually saving resources that come up when working with
    # the faked modules.
    def save_resource(self, resource: 'IRResource') -> 'IRResource':
        return resource

# XXX More fakery here. This duplicates code from ircluster.py.
class Service:
    def __init__(self, logger, service: str, allow_scheme=True, ctx_name: str=None) -> None:
        original_service = service

        originate_tls = False

        self.scheme = 'http'
        self.errors: List[str] = []
        self.name_fields: List[str] = []
        self.ctx_name = ctx_name

        if allow_scheme and service.lower().startswith("https://"):
            service = service[len("https://"):]

            originate_tls = True
            self.name_fields.append('otls')

        elif allow_scheme and service.lower().startswith("http://"):
            service = service[ len("http://"): ]

            if ctx_name:
                self.errors.append(f'Originate-TLS context {ctx_name} being used even though service {service} lists HTTP')
                originate_tls = True
                self.name_fields.append('otls')
            else:
                originate_tls = False

        elif ctx_name:
            # No scheme (or schemes are ignored), but we have a context.
            originate_tls = True
            self.name_fields.append('otls')
            self.name_fields.append(ctx_name)

        if '://' in service:
            idx = service.index('://')
            scheme = service[0:idx]

            if allow_scheme:
                self.errors.append(f'service {service} has unknown scheme {scheme}, assuming {self.scheme}')
            else:
                self.errors.append(f'ignoring scheme {scheme} for service {service}, since it is being used for a non-HTTP mapping')

            service = service[idx + 3:]

        # # XXX Should this be checking originate_tls? Why does it do that?
        # if originate_tls and host_rewrite:
        #     name_fields.append("hr-%s" % host_rewrite)

        # Parse the service as a URL. Note that we have to supply a scheme to urllib's
        # parser, because it's kind of stupid.

        logger.debug(f'Service: {original_service} otls {originate_tls} ctx {ctx_name} -> {self.scheme}, {service}')
        p = urlparse('random://' + service)

        # Is there any junk after the host?

        if p.path or p.params or p.query or p.fragment:
            self.errors.append(f'service {service} has extra URL components; ignoring everything but the host and port')

        # p is read-only, so break stuff out.

        self.hostname = p.hostname
        self.port = p.port

        # If the port is unset, fix it up.
        if not self.port:
            self.port = 443 if originate_tls else 80


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
