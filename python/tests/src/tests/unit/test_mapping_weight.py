import logging

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador import IR, Config  # noqa: E402
from ambassador.fetch import ResourceFetcher  # noqa: E402
from ambassador.utils import NullSecretHandler  # noqa: E402

# Regression test for https://github.com/emissary-ingress/emissary/issues/5811:
# a Mapping that is alone in its group -- i.e. the only Mapping with its
# prefix/method/headers/host -- used to always have its computed weight
# forced to 100, even when the Mapping explicitly asked for weight: 0 (e.g. a
# canary that's been scaled down but not yet deleted). That's wrong both for
# the diagnostics UI, which reports the computed weight, and for the actual
# Envoy config, which uses it to decide how much traffic the Mapping gets.
#
# Only weight: 0 is honored for a lone Mapping. Any other explicit weight
# (e.g. 15) is still forced to 100, because honoring it would make Envoy drop
# the remaining traffic, which is the problem the forcing was there to prevent.


def _get_ir_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)
    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return ir


def _weight_of(ir, mapping_name):
    for group in ir.groups.values():
        for mapping in group.mappings:
            if mapping.name == mapping_name:
                return len(group.mappings), mapping._weight
    raise AssertionError(f"mapping {mapping_name!r} not found in any group")


@pytest.mark.compilertest
def test_lone_mapping_with_explicit_zero_weight_stays_at_zero():
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: zero-weight-mapping
namespace: default
prefix: /canary-scaled-down/
service: zeroweight:8080
weight: 0
"""
    ir = _get_ir_config(yaml)
    group_size, weight = _weight_of(ir, "zero-weight-mapping")

    assert group_size == 1, f"expected this Mapping to be alone in its group, got {group_size}"
    assert weight == 0, f"expected the explicit weight: 0 to be honored, got {weight}"


@pytest.mark.compilertest
def test_lone_mapping_with_explicit_nonzero_weight_is_still_forced_to_100():
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: partial-weight-mapping
namespace: default
prefix: /partial/
service: partialsvc:8080
weight: 15
"""
    ir = _get_ir_config(yaml)
    group_size, weight = _weight_of(ir, "partial-weight-mapping")

    assert group_size == 1, f"expected this Mapping to be alone in its group, got {group_size}"
    assert weight == 100, f"a lone Mapping with weight 15 must still be 100 so Envoy doesn't drop traffic, got {weight}"


@pytest.mark.compilertest
def test_lone_mapping_without_explicit_weight_defaults_to_100():
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: weightless-mapping
namespace: default
prefix: /weightless/
service: weightless:8080
"""
    ir = _get_ir_config(yaml)
    group_size, weight = _weight_of(ir, "weightless-mapping")

    assert group_size == 1, f"expected this Mapping to be alone in its group, got {group_size}"
    assert weight == 100, f"a Mapping with no explicit weight should still default to 100, got {weight}"
