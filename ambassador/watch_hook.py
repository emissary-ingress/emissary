#!/usr/bin/python3
from typing import Any, Dict

import sys

import json
import logging
import traceback

from collections import OrderedDict

from multi import multi

from ambassador.utils import parse_yaml
from ambassador.config import Config


########
# This is the quick-and-dirty approach to the watch hook. It needs to be rewritten to use
# the ResourceFetcher and friends...


logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s watch_hook %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

logger = logging.getLogger('ambassador')
logger.setLevel(logging.DEBUG)


class ConsulResolver:

    def __init__(self, source: str, address: str, datacenter: str) -> None:
        self.source = source
        self.address = address
        self.datacenter = datacenter


class Mapping:

    def __init__(self, source: str, namespace: str, service: str, resolver: str) -> None:
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
    def load(self, source: str, namespace: str, obj: Any) -> None:
        del source      # silence warnings
        del namespace

        yield obj["kind"]

    @load.when("Mapping", "TCPMapping")
    def load(self, source: str, namespace: str, m: Dict[str, Any]) -> None:
        self.mappings.append(Mapping(source, namespace, m["service"], m.get("resolver")))

    @load.when("ConsulResolver")
    def load(self, source: str, namespace: str, r: Dict[str, Any]) -> None:
        del namespace   # silence warning

        self.resolvers[r["name"]] = ConsulResolver(source, r["address"], r["datacenter"])

    @load.when("Module", "AuthService", "TLSContext", "KubernetesServiceResolver", "KubernetesEndpointResolver",
               "RateLimitService", "TracingService")
    def load(self, *args) -> None:
        pass

    def print_watches(self):
        # we just watch all the endpoints for now because for some
        # reason it is slow to watch individual ones
        k8s_watches = [
            {
                "kind": "endpoints",
                "namespace": Config.ambassador_namespace if Config.single_namespace else "",
                "field-selector": "metadata.namespace!=kube-system"
            }
        ]

        consul_watches = []

        for m in self.mappings:
            if m.resolver is not None:
                r = self.resolvers.get(m.resolver)

                if r is None:
                    logger.error(f"mapping {m.source} has unknown resolver: {m.resolver}")
                else:
                    consul_watches.append(
                        {
                            "consul-address": r.address,
                            "datacenter": r.datacenter,
                            "service-name": m.service
                        }
                    )

        watchset = {
            "kubernetes-watches": k8s_watches,
            "consul-watches": consul_watches
        }

        json.dump(watchset, sys.stdout)


def main(stream) -> None:
    snapshot = json.load(stream)

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


if __name__ == "__main__":
    main(sys.stdin)
