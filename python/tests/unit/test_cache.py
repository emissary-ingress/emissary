from typing import Any, Callable, Dict, List, Optional, Set, Tuple

import difflib
import json
import logging
import os
import random
import re
import sys
import yaml
from pathlib import Path

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
    def __init__(self, logger: logging.Logger, tmpdir: Path, yaml_file: str,
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
        self.secret_handler = NullSecretHandler(logger, str(tmpdir/"secrets"/"src"), str(tmpdir/"secrets"/"cache"), "0")

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

    def build(self) -> Tuple[IR, EnvoyConfig]:
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

        econf = EnvoyConfig.generate(ir, cache=self.cache)

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


@pytest.mark.compilertest
def test_circular_link(tmp_path):
    builder = Builder(logger, tmp_path, "cache_test_1.yaml")
    builder.build()
    assert builder.cache

    # This Can't Happen(tm) in Ambassador, but it's important that it not go
    # off the rails. Find a Mapping...
    mapping_key = "Mapping-v2-foo-4-default"
    m = builder.cache[mapping_key]
    assert m

    # ...then walk the link chain until we get to a V2-Cluster.
    worklist = [ m.cache_key ]
    cluster_key: Optional[str] = None

    while worklist:
        key = worklist.pop(0)

        if key.startswith('V3-Cluster'):
            cluster_key = key
            break

        if key in builder.cache.links:
            for owned in builder.cache.links[key]:
                worklist.append(owned)

    assert cluster_key is not None, f"No V3-Cluster linked from {m}?"

    c = builder.cache[cluster_key]

    assert c is not None, f"No V3-Cluster in the cache for {c}"

    builder.cache.link(c, m)
    builder.cache.invalidate(mapping_key)

    builder.build()
    builder.check_last("after invalidating circular links")


@pytest.mark.compilertest
def test_multiple_rebuilds(tmp_path):
    builder = Builder(logger, tmp_path, "cache_test_1.yaml")

    for i in range(10):
        builder.build()

        if i > 0:
            builder.check_last(f"rebuild {i-1} -> {i}")


@pytest.mark.compilertest
def test_simple_targets(tmp_path):
    builder = Builder(logger, tmp_path, "cache_test_1.yaml")

    builder.build()
    builder.build()

    builder.check_last("immediate rebuild")

    builder.invalidate("Mapping-v2-foo-4-default")

    builder.build()

    builder.check_last("after delete foo-4")


@pytest.mark.compilertest
def test_smashed_targets(tmp_path):
    builder = Builder(logger, tmp_path, "cache_test_2.yaml")

    builder.build()
    builder.build()

    builder.check_last("immediate rebuild")

    # Invalidate two things that share common links.
    builder.invalidate("Mapping-v2-foo-4-default")
    builder.invalidate("Mapping-v2-foo-6-default")

    builder.build()

    builder.check_last("after invalidating foo-4 and foo-6")


@pytest.mark.compilertest
def test_delta_1(tmp_path):
    builder1 = Builder(logger, tmp_path, "cache_test_1.yaml")
    builder2 = Builder(logger, tmp_path, "cache_test_1.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    builder1.apply_yaml("cache_delta_1.yaml")
    builder2.apply_yaml("cache_delta_1.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after delta", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, tmp_path, "cache_result_1.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


@pytest.mark.compilertest
def test_delta_2(tmp_path):
    builder1 = Builder(logger, tmp_path, "cache_test_2.yaml")
    builder2 = Builder(logger, tmp_path, "cache_test_2.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    builder1.apply_yaml("cache_delta_2.yaml")
    builder2.apply_yaml("cache_delta_2.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after delta", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, tmp_path, "cache_result_2.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


@pytest.mark.compilertest
def test_delta_3(tmp_path):
    builder1 = Builder(logger, tmp_path, "cache_test_1.yaml")
    builder2 = Builder(logger, tmp_path, "cache_test_1.yaml", enable_cache=False)

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

    builder3 = Builder(logger, tmp_path, "cache_result_3.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


@pytest.mark.compilertest
def test_delete_4(tmp_path):
    builder1 = Builder(logger, tmp_path, "cache_test_1.yaml")
    builder2 = Builder(logger, tmp_path, "cache_test_1.yaml", enable_cache=False)

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    # Delete a resource.
    builder1.delete_yaml("cache_delta_1.yaml")
    builder2.delete_yaml("cache_delta_1.yaml")

    b1 = builder1.build()
    b2 = builder2.build()

    builder1.check("after deletion", b1, b2, strip_cache_keys=True)

    builder3 = Builder(logger, tmp_path, "cache_result_4.yaml")
    b3 = builder3.build()

    builder3.check("final", b3, b1)


@pytest.mark.compilertest
def test_long_cluster_1(tmp_path):
    # Create a cache for Mappings whose cluster names are too long
    # to be envoy cluster names and must be truncated.
    builder1 = Builder(logger, tmp_path, "cache_test_3.yaml")
    builder2 = Builder(logger, tmp_path, "cache_test_3.yaml", enable_cache=False)

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


@pytest.mark.compilertest
def test_mappings_same_name_delta(tmp_path):
    # Tests that multiple Mappings with the same name (but in different namespaces)
    # are properly added/removed from the cache when they are updated.
    builder = Builder(logger, tmp_path, "cache_test_4.yaml")
    b = builder.build()
    econf = b[1].as_dict()

    # loop through all the clusters in the resulting envoy config and pick out two Mappings from our test set (first and lase)
    # to ensure their clusters were generated properly.
    cluster1_ok = False
    cluster2_ok = False
    for cluster in econf['static_resources']['clusters']:
        cname = cluster.get('name', None)
        assert cname is not None, \
            f"Error, cluster missing cluster name in econf"
        # The 6666 in the cluster name comes from the Mapping.spec.service's port
        if cname == "cluster_bar_0_example_com_6666_bar0":
            cluster1_ok = True
        elif cname == "cluster_bar_9_example_com_6666_bar9":
            cluster2_ok = True
        if cluster1_ok and cluster2_ok:
            break
    assert cluster1_ok and cluster2_ok, 'clusters could not be found with the correct envoy config'

    # Update the yaml for these Mappings to simulate a reconfiguration
    # We should properly remove the cache entries for these clusters when that happens.
    builder.apply_yaml("cache_test_5.yaml")
    b = builder.build()
    econf = b[1].as_dict()

    cluster1_ok = False
    cluster2_ok = False
    for cluster in econf['static_resources']['clusters']:
        cname = cluster.get('name', None)
        assert cname is not None, \
            f"Error, cluster missing cluster name in econf"
        # We can check the cluster name to identify if the clusters were updated properly
        # because in the deltas for the yaml we applied, we changed the port to 7777
        # If there was an issue removing the initial ones from the cache then we should see
        # 6666 in this field and not find the cluster names below.
        if cname == "cluster_bar_0_example_com_7777_bar0":
            cluster1_ok = True
        elif cname == "cluster_bar_9_example_com_7777_bar9":
            cluster2_ok = True
        if cluster1_ok and cluster2_ok:
            break
    assert cluster1_ok and cluster2_ok, 'clusters could not be found with the correct econf after updating their config'


MadnessVerifier = Callable[[Tuple[IR, EnvoyConfig]], bool]


class MadnessMapping:
    name: str
    pfx: str
    service: str

    def __init__(self, name, pfx, svc) -> None:
        self.name = name
        self.pfx = pfx
        self.service = svc

        # This is only OK for service names without any weirdnesses.
        self.cluster = "cluster_" + re.sub(r'[^0-9A-Za-z_]', '_', self.service) + "_default"

    def __str__(self) -> str:
        return f"MadnessMapping {self.name}: {self.pfx} => {self.service}"

    def yaml(self) -> str:
        return f"""
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: {self.name}
    namespace: default
spec:
    prefix: {self.pfx}
    service: {self.service}
"""


class MadnessOp:
    name: str
    op: str
    mapping: MadnessMapping
    verifiers: List[MadnessVerifier]
    tmpdir: Path

    def __init__(self, name: str, op: str, mapping: MadnessMapping, verifiers: List[MadnessVerifier], tmpdir: Path) -> None:
        self.name = name
        self.op = op
        self.mapping = mapping
        self.verifiers = verifiers
        self.tmpdir = tmpdir

    def __str__(self) -> str:
        return self.name

    def exec(self, builder1: Builder, builder2: Builder, dumpfile: Optional[str]=None) -> bool:
        verifiers: List[MadnessVerifier] = []

        if self.op == "apply":
            builder1.apply_yaml_string(self.mapping.yaml())
            builder2.apply_yaml_string(self.mapping.yaml())

            verifiers.append(self._cluster_present)
        elif self.op == "delete":
            builder1.delete_yaml_string(self.mapping.yaml())
            builder2.delete_yaml_string(self.mapping.yaml())

            verifiers.append(self._cluster_absent)
        else:
            raise Exception(f"Unknown op {self.op}")

        logger.info("======== builder1:")
        logger.info("INPUT: %s" % builder1.current_yaml())

        b1 = builder1.build()

        logger.info("IR: %s" % json.dumps(b1[0].as_dict(), indent=2, sort_keys=True))

        logger.info("======== builder2:")
        logger.info("INPUT: %s" % builder2.current_yaml())

        b2 = builder2.build()

        logger.info("IR: %s" % json.dumps(b2[0].as_dict(), indent=2, sort_keys=True))

        if dumpfile:
            json.dump(b1[0].as_dict(), open(str(self.tmpdir/f"{dumpfile}-1.json"), "w"), indent=2, sort_keys=True)
            json.dump(b2[0].as_dict(), open(str(self.tmpdir/f"{dumpfile}-2.json"), "w"), indent=2, sort_keys=True)

        if not builder1.check(self.name, b1, b2, strip_cache_keys=True):
            return False

        verifiers += self.verifiers

        for v in verifiers:
            # for b in [ b1 ]:
            for b in [ b1, b2 ]:
                # The verifiers are meant to do assertions. The return value is
                # about short-circuiting the loop, not logging the errors.
                if not v(b):
                    return False

        return True

    def _cluster_present(self, b: Tuple[IR, EnvoyConfig]) -> bool:
        ir, econf = b

        ir_has_cluster = ir.has_cluster(self.mapping.cluster)
        assert ir_has_cluster, f"{self.name}: needed IR cluster {self.mapping.cluster}, have only {', '.join(ir.clusters.keys())}"

        return ir_has_cluster

    def _cluster_absent(self, b: Tuple[IR, EnvoyConfig]) -> bool:
        ir, econf = b

        ir_has_cluster = ir.has_cluster(self.mapping.cluster)
        assert not ir_has_cluster, f"{self.name}: needed no IR cluster {self.mapping.cluster}, but found it"

        return not ir_has_cluster

    def check_group(self, b: Tuple[IR, EnvoyConfig], current_mappings: Dict[MadnessMapping, bool]) -> bool:
        ir, econf = b
        match = False

        group = ir.groups.get("3644d75eb336f323bec43e48d4cfd8a950157607", None)

        if current_mappings:
            # There are some active mappings. Make sure that the group exists, that it has the
            # correct mappings, and that the mappings have sane weights.
            assert group, f"{self.name}: needed group 3644d75eb336f323bec43e48d4cfd8a950157607, but none found"

            # We expect the mappings to be sorted in the group, because every change to the
            # mappings that are part of the group should result in the whole group being torn
            # down and recreated.
            wanted_services = sorted([ m.service for m in current_mappings.keys() ])
            found_services = [ m.service for m in group.mappings ]

            match1 = (wanted_services == found_services)
            assert match1, f"{self.name}: wanted services {wanted_services}, but found {found_services}"

            weight_delta = 100 // len(current_mappings)
            wanted_weights: List[int] = [ (i + 1) * weight_delta for i in range(len(current_mappings)) ]
            wanted_weights[-1] = 100

            found_weights: List[int] = [ m._weight for m in group.mappings ]

            match2 = (wanted_weights == found_weights)
            assert match2, f"{self.name}: wanted weights {wanted_weights}, but found {found_weights}"

            return match1 and match2
        else:
            # There are no active mappings, so make sure that the group doesn't exist.
            assert not group, f"{self.name}: needed no group 3644d75eb336f323bec43e48d4cfd8a950157607, but found one"
            match = True

        return match

@pytest.mark.compilertest
def test_cache_madness(tmp_path):
    builder1 = Builder(logger, tmp_path, "/dev/null")
    builder2 = Builder(logger, tmp_path, "/dev/null", enable_cache=False)

    logger.info("======== builder1:")
    logger.info("INPUT: %s" % builder1.current_yaml())

    b1 = builder1.build()

    logger.info("IR: %s" % json.dumps(b1[0].as_dict(), indent=2, sort_keys=True))

    logger.info("======== builder2:")
    logger.info("INPUT: %s" % builder2.current_yaml())

    b2 = builder2.build()

    logger.info("IR: %s" % json.dumps(b2[0].as_dict(), indent=2, sort_keys=True))

    builder1.check("baseline", b1, b2, strip_cache_keys=True)

    # We're going to mix and match some changes to the config,
    # in a random order.

    all_mappings = [
        MadnessMapping("mapping1", "/foo/", "service1"),
        MadnessMapping("mapping2", "/foo/", "service2"),
        MadnessMapping("mapping3", "/foo/", "service3"),
        MadnessMapping("mapping4", "/foo/", "service4"),
        MadnessMapping("mapping5", "/foo/", "service5"),
    ]

    current_mappings: Dict[MadnessMapping, bool] = {}

    # grunge = [ all_mappings[i] for i in [ 0, 3, 2 ] ]

    # for i in range(len(grunge)):
    #     mapping = grunge[i]

    for i in range(0, 100):
        mapping = random.choice(all_mappings)
        op: MadnessOp

        if mapping in current_mappings:
            del(current_mappings[mapping])
            op = MadnessOp(name=f"delete {mapping.pfx} -> {mapping.service}", op="delete", mapping=mapping,
                           verifiers=[ lambda b: op.check_group(b, current_mappings) ],
                           tmpdir=tmp_path)
        else:
            current_mappings[mapping] = True
            op = MadnessOp(name=f"apply {mapping.pfx} -> {mapping.service}", op="apply", mapping=mapping,
                           verifiers=[ lambda b: op.check_group(b, current_mappings) ],
                           tmpdir=tmp_path)

        print("==== EXEC %d: %s => %s" % (i, op, sorted([ m.service for m in current_mappings.keys() ])))
        logger.info("======== EXEC %d: %s", i, op)

        # if not op.exec(builder1, None, dumpfile=f"ir{i}"):
        if not op.exec(builder1, builder2, dumpfile=f"ir{i}"):
            break


if __name__ == '__main__':
    pytest.main(sys.argv)
