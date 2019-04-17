#!/usr/bin/python3
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

import json, traceback, urllib
from collections import OrderedDict
from ambassador.utils import parse_yaml
from ambassador.config import Config
from multi import multi

class ConsulResolver:

    def __init__(self, source, address, datacenter):
        self.source = source
        self.address = address
        self.datacenter = datacenter

class Mapping:

    def __init__(self, source, namespace, service, resolver):
        self.source = source
        self.namespace = namespace
        self.service = service
        self.resolver = resolver

class Loader:

    def __init__(self):
        self.services = set()
        self.mappings = []
        self.resolvers = OrderedDict()

    def service(self, name, namespace):
        self.services.add(f"{name}.{namespace}")

    @multi
    def load(self, source, namespace, obj):
        yield obj["kind"]

    @load.when("Mapping", "TCPMapping")
    def load(self, source, namespace, m):
        self.mappings.append(Mapping(source, namespace, m["service"], m.get("resolver")))

    @load.when("ConsulResolver")
    def load(self, source, namespace, r):
        self.resolvers[r["name"]] = ConsulResolver(source, r["address"], r["datacenter"])

    @load.when("Module", "AuthService", "TLSContext", "KubernetesServiceResolver", "KubernetesEndpointResolver",
               "RateLimitService", "TracingService")
    def load(self, *args):
        pass

    def print_watches(self):
        # we just watch all the endpoints for now because for some
        # reason it is slow to watch individual ones
        k8s_watches = [{"kind": "endpoints",
                        "namespace": Config.ambassador_namespace if Config.single_namespace else "",
                        "field-selector": "metadata.namespace!=kube-system"}]
        consul_watches = []
        for m in self.mappings:
            if m.resolver is not None:
                r = self.resolvers.get(m.resolver)
                if r is None:
                    logger.error(f"mapping {m.source} has unknown resolver: {m.resolver}")
                else:
                    consul_watches.append({"consul-address": r.address,
                                           "datacenter": r.datacenter,
                                           "service-name": f"{m.service}"})

        watchset = {
            "kubernetes-watches": k8s_watches,
            "consul-watches": consul_watches
        }
        json.dump(watchset, sys.stdout)

snapshot = json.load(sys.stdin)
# XXX: should make everything lowercase in watt
services = snapshot.get("Kubernetes", {}).get("service") or []

loader = Loader()

for svc in services:
    metadata = svc.get("metadata", {})
    namespace = metadata.get("namespace", "default")
    name = metadata["name"]
    loader.service(name, namespace)
    annotations = metadata.get("annotations", {})
    config = annotations.get("getambassador.io/config")
    if config:
        objs = parse_yaml(config)
        for idx, obj in enumerate(objs):
            source = metadata["name"] + "." + namespace + f".{idx}"
            try:
                loader.load(source, namespace, obj)
            except:
                logger.error("error loading object from %s: %s", source, traceback.format_exc())

loader.print_watches()
