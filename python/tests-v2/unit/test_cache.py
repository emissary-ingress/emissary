from typing import Any, Callable, Dict, List, Optional, OrderedDict, Set, Tuple

import difflib
import json
import logging
import os
import random
import re
import sys
import yaml

import pytest

logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")
logger.setLevel(logging.DEBUG)

from ambassador import Cache, Config, IR, EnvoyConfig
from ambassador.ir.ir import IRFileChecker
from ambassador.fetch import ResourceFetcher
from ambassador.utils import SecretHandler, NullSecretHandler, Timer


class Builder:
    def __init__(self, logger: logging.Logger, yaml_file: str,
                 enable_cache=True) -> None:
        self.logger = logger
        
        self.test_dir = os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            "test_cache_data"
        )

        self.cache: Optional[Cache] = None

        if enable_cache:
            self.cache = Cache(logger)

        # This is a brutal hack: we load all the YAML, store it as objects, then
        # build IR and econf from the re-serialized YAML from these resources.
        # The reason is that it's kind of the only way we can apply deltas in
        # a meaningful way.
        self.resources: Dict[str, Any] = {}
        self.deltas: Dict[str, Any] = {}

        # Load the initial YAML.
        self.apply_yaml(yaml_file, allow_updates=False)
        self.secret_handler = NullSecretHandler(logger, "/tmp/secrets/src", "/tmp/secrets/cache", "0")

        # Save builds to make this simpler to call.
        self.builds: List[Tuple[IR, EnvoyConfig]] = []

    def current_yaml(self) -> str:
        return yaml.safe_dump_all(list(self.resources.values()))

    def apply_yaml(self, yaml_file: str, allow_updates=True) -> None:
        yaml_data = open(os.path.join(self.test_dir, yaml_file), "r").read()

        self.apply_yaml_string(yaml_data, allow_updates)

    def apply_yaml_string(self, yaml_data: str, allow_updates=True) -> None:
        for rsrc in yaml.safe_load_all(yaml_data):
            # We require kind, metadata.name, and metadata.namespace here.
            kind = rsrc['kind']
            metadata = rsrc['metadata']
            name = metadata['name']
            namespace = metadata['namespace']

            key = f"{kind}-v2-{name}-{namespace}"

            dtype = "add"

            if key in self.resources:
                # This is an attempted update.
                if not allow_updates:
                    raise RuntimeError(f"Cannot update {key}")

                dtype = "update"

                # if self.cache is not None:
                #     self.cache.invalidate(key)

            self.resources[key] = rsrc
            self.deltas[key] = {
                "kind": kind,
                "apiVersion": rsrc["apiVersion"],
                "metadata": {
                    "name": name,
                    "namespace": namespace,
                    "creationTimestamp": metadata.get("creationTimestamp", "2021-11-19T15:11:45Z")
                },
                "deltaType": dtype
            }

    def delete_yaml(self, yaml_file: str) -> None:
        yaml_data = open(os.path.join(self.test_dir, yaml_file), "r").read()

        self.delete_yaml_string(yaml_data)

    def delete_yaml_string(self, yaml_data: str) -> None:
        for rsrc in yaml.safe_load_all(yaml_data):
            # We require kind, metadata.name, and metadata.namespace here.
            kind = rsrc['kind']
            metadata = rsrc['metadata']
            name = metadata['name']
            namespace = metadata['namespace']

            key = f"{kind}-v2-{name}-{namespace}"

            if key in self.resources:
                del(self.resources[key])

                # if self.cache is not None:
                #     self.cache.invalidate(key)

                self.deltas[key] = {
                    "kind": kind,
                    "apiVersion": rsrc["apiVersion"],
                    "metadata": {
                        "name": name,
                        "namespace": namespace,
                        "creationTimestamp": metadata.get("creationTimestamp", "2021-11-19T15:11:45Z")
                    },
                    "deltaType": "delete"
                }

    def build(self, version='V2') -> Tuple[IR, EnvoyConfig]:
        # Do a build, return IR & econf, but also stash them in self.builds.

        watt: Dict[str, Any] = {
            "Kubernetes": {},
            "Deltas": list(self.deltas.values())
        }

        # Clear deltas for the next build.
        self.deltas = {}

        # The Ambassador resource types are all valid keys in the Kubernetes dict.
        # Other things (e.g. if this test gets expanded to cover Ingress or Secrets)
        # may not be.

        for rsrc in self.resources.values():
            kind = rsrc['kind']

            if kind not in watt['Kubernetes']:
                watt['Kubernetes'][kind] = []
            
            watt['Kubernetes'][kind].append(rsrc)

        watt_json = json.dumps(watt, sort_keys=True, indent=4)

        self.logger.debug(f"Watt JSON:\n{watt_json}")

        # OK, we have the WATT-formatted JSON. This next bit of code largely duplicates
        # _load_ir from diagd.
        #
        # XXX That obviously means that it should be factored out for reuse.

        # Grab a new aconf, and use a new ResourceFetcher to load it up.
        aconf = Config()

        fetcher = ResourceFetcher(self.logger, aconf)
        fetcher.parse_watt(watt_json)

        aconf.load_all(fetcher.sorted())

        # Next up: What kind of reconfiguration are we doing?
        config_type, reset_cache, invalidate_groups_for = IR.check_deltas(self.logger, fetcher, self.cache)

        # For the tests in this file, we should see cache resets and full reconfigurations
        # IFF we have no cache.

        if self.cache is None:
            assert config_type == "complete", "check_deltas wants an incremental reconfiguration with no cache, which it shouldn't"
            assert reset_cache, "check_deltas with no cache does not want to reset the cache, but it should"
        else:
            assert config_type == "incremental", "check_deltas with a cache wants a complete reconfiguration, which it shouldn't"
            assert not reset_cache, "check_deltas with a cache wants to reset the cache, which it shouldn't"

        # Once that's done, compile the IR.
        ir = IR(aconf, logger=self.logger,
                cache=self.cache, invalidate_groups_for=invalidate_groups_for,
                file_checker=lambda path: True,
                secret_handler=self.secret_handler)

        assert ir, "could not create an IR"

        econf = EnvoyConfig.generate(ir, version, cache=self.cache)

        assert econf, "could not create an econf"

        self.builds.append(( ir, econf ))

        return ir, econf

    def invalidate(self, key) -> None:
        if self.cache is not None:
            assert self.cache[key] is not None, f"key {key} is not cached"

            self.cache.invalidate(key)

    def check(self, what: str, b1: Tuple[IR, EnvoyConfig], b2: Tuple[IR, EnvoyConfig],
              strip_cache_keys=False) -> bool:
        for kind, idx in [ ( "IR", 0 ), ( "econf", 1 ) ]:
            if strip_cache_keys and (idx == 0):
                x1 = self.strip_cache_keys(b1[idx].as_dict())
                j1 = json.dumps(x1, sort_keys=True, indent=4)

                x2 = self.strip_cache_keys(b2[idx].as_dict())
                j2 = json.dumps(x2, sort_keys=True, indent=4)
            else:
                j1 = b1[idx].as_json()
                j2 = b2[idx].as_json()

            match = (j1 == j2)

            output = ""

            if not match:
                l1 = j1.split("\n")
                l2 = j2.split("\n")

                n1 = f"{what} {kind} 1"
                n2 = f"{what} {kind} 2"

                output += "\n--------\n"

                for line in difflib.context_diff(l1, l2, fromfile=n1, tofile=n2):
                    line = line.rstrip()
                    output += line
                    output += "\n"

            assert match, output

        return match

    def check_last(self, what: str) -> None:
        build_count = len(self.builds)

        b1 = self.builds[build_count - 2]
        b2 = self.builds[build_count - 1]

        self.check(what, b1, b2)

    def strip_cache_keys(self, node: Any) -> Any:
        if isinstance(node, dict):
            output = {}
            for k, v in node.items():
                if k == '_cache_key':
                    continue

                output[k] = self.strip_cache_keys(v)

            return output
        elif isinstance(node, list):
            return [ self.strip_cache_keys(x) for x in node ]

        return node


def test_circular_link():
    builder = Builder(logger, "cache_test_1.yaml")
    builder.build()

    # This Can't Happen(tm) in Ambassador, but it's important that it not go
    # off the rails. Find a Mapping...
    mapping_key = "Mapping-v2-foo-4-default"
    m = builder.cache[mapping_key]

    # ...then walk the link chain until we get to a V2-Cluster.
    worklist = [ m.cache_key ]
    cluster_key: Optional[str] = None

    while worklist:
        key = worklist.pop(0)

        if key.startswith('V2-Cluster'):
            cluster_key = key
            break

        if key in builder.cache.links:
            for owned in builder.cache.links[key]:
                worklist.append(owned)

    assert cluster_key is not None, f"No V2-Cluster linked from {m}?"

    c = builder.cache[cluster_key]

    assert c is not None, f"No V2-Cluster in the cache for {c}"

    builder.cache.link(c, m)
    builder.cache.invalidate(mapping_key)

    builder.build()
    builder.check_last("after invalidating circular links")


def test_multiple_rebuilds():
    builder = Builder(logger, "cache_test_1.yaml")

    for i in range(10):
        builder.build()

        if i > 0:
            builder.check_last(f"rebuild {i-1} -> {i}")


def test_simple_targets():
    builder = Builder(logger, "cache_test_1.yaml")

    builder.build()
    builder.build()

    builder.check_last("immediate rebuild")

    builder.invalidate("Mapping-v2-foo-4-default")

    builder.build()

    builder.check_last("after delete foo-4")


def test_smashed_targets():
    builder = Builder(logger, "cache_test_2.yaml")

    builder.build()
    builder.build()

    builder.check_last("immediate rebuild")

    # Invalidate two things that share common links.
    builder.invalidate("Mapping-v2-foo-4-default")
    builder.invalidate("Mapping-v2-foo-6-default")

    builder.build()

    builder.check_last("after invalidating foo-4 and foo-6")


def test_delta_1():
    builder1 = Builder(logger, "cache_test_1.yaml")
    builder2 = Builder(logger, "cache_test_1.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    builder1.apply_yaml("cache_delta_1.yaml")
    builder2.apply_yaml("cache_delta_1.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after delta", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, "cache_result_1.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


def test_delta_2():
    builder1 = Builder(logger, "cache_test_2.yaml")
    builder2 = Builder(logger, "cache_test_2.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    builder1.apply_yaml("cache_delta_2.yaml")
    builder2.apply_yaml("cache_delta_2.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after delta", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, "cache_result_2.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


def test_delta_3():
    builder1 = Builder(logger, "cache_test_1.yaml")
    builder2 = Builder(logger, "cache_test_1.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    # Load up five delta files and apply them in a random order.
    deltas = [ f"cache_random_{i}.yaml" for i in [ 1, 2, 3, 4, 5 ] ]
    random.shuffle(deltas)

    for delta in deltas:
        builder1.apply_yaml(delta)
        builder2.apply_yaml(delta)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after deltas", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, "cache_result_3.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


def test_delete_4():
    builder1 = Builder(logger, "cache_test_1.yaml")
    builder2 = Builder(logger, "cache_test_1.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    # Delete a resource.
    builder1.delete_yaml("cache_delta_1.yaml")
    builder2.delete_yaml("cache_delta_1.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after deletion", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, "cache_result_4.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)

def test_long_cluster_1():
    # Create a cache for Mappings whose cluster names are too long
    # to be envoy cluster names and must be truncated.
    builder1 = Builder(logger, "cache_test_3.yaml")
    builder2 = Builder(logger, "cache_test_3.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    print("checking baseline...")
    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    # Apply the second Mapping, make sure we use the same cached cluster
    builder1.apply_yaml("cache_delta_3.yaml")
    builder2.apply_yaml("cache_delta_3.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    print("checking after apply...")
    builder1.check("after apply", b1, b2, strip_cache_keys=True)

    print("test_long_cluster_1 done")

if __name__ == '__main__':
    pytest.main(sys.argv)
