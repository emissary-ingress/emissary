#!/usr/bin/python

from typing import Dict, Optional, Tuple, TYPE_CHECKING

import sys

import json
import logging
import os

from urllib.parse import urlparse

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test-dump %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

logger = logging.getLogger('ambassador')
logger.setLevel(logging.INFO)

from ambassador import Config, IR
# from ambassador.envoy import V2Config

# from ambassador.utils import SecretInfo, SavedSecret, SecretHandler
from ambassador.config.resourcefetcher import ResourceFetcher

if TYPE_CHECKING:
    from ambassador.ir.irtlscontext import IRTLSContext


class Service:
    def __init__(self, logger, service: str, allow_scheme=True, ctx_name: str=None) -> None:
        original_service = service

        originate_tls = False

        self.scheme = 'http'
        self.errors = []
        self.name_fields = []
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

yaml_stream = sys.stdin

if len(sys.argv) > 1:
    yaml_stream = open(sys.argv[1], "r")

aconf = Config()
fetcher = ResourceFetcher(logger, aconf)
fetcher.parse_watt(yaml_stream.read())

aconf.load_all(fetcher.sorted())

mappings = aconf.get_config('mappings') or {}
resolvers = aconf.get_config('resolvers') or {}
contexts = aconf.get_config('tls_contexts') or {}
secrets = aconf.get_config('secret') or {}  # 'secret', singular, is not a typo

consul_watches = []

for mname, mapping in mappings.items():
    res_name = mapping.get('resolver', None)
    ctx_name = mapping.get('tls', None)

    if res_name:
        resolver = resolvers.get(res_name, None)

        if resolver:
            if resolver.kind == 'ConsulResolver':
                logger.debug(f'Mapping {mname} uses Consul resolver {res_name}')

                svc = Service(logger, mapping.service, ctx_name)

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

watchset = {
    "kubernetes-watches": [
        {
            "kind": "endpoints",
            "namespace": Config.ambassador_namespace if Config.single_namespace else "",
            "field-selector": "metadata.namespace!=kube-system"
        }
    ],
    "consul-watches": consul_watches
}

save_dir = os.environ.get('AMBASSADOR_WATCH_DIR', None)

if save_dir:
    json.dump(watchset, open(os.path.join(save_dir, 'watch.yaml'), "w"))

json.dump(watchset, sys.stdout)
