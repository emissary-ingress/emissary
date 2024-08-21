import json
import os
from dataclasses import dataclass
from typing import List, Optional

import pytest
import yaml

from tests.utils import compile_with_cachecheck


@dataclass
class MappingGroupTestOutput:
    group_id: str
    host: Optional[str]
    prefix: str
    mappings: List[str]


test_cases = [
    "mapping_selector",
    "mapping_selector_and_authority",
    "mapping_selector_and_hostname",
    "mapping_selector_and_host",
    "mapping_selector_to_multiple_hosts",
    "mapping_selector_irrelevant_labels",
]


@pytest.mark.compilertest
@pytest.mark.parametrize("test_case", test_cases)
def test_mapping_canary_group_selectors(test_case):
    testdata_dir = os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "testdata", "canary_groups"
    )

    with open(os.path.join(testdata_dir, f"{test_case}_in.yaml"), "r") as f:
        test_yaml = f.read()

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    mapping_groups = []
    for g in ir.groups.values():
        if g.prefix.startswith("/ambassador") or g.prefix.startswith("/.ambassador"):
            continue

        mg = MappingGroupTestOutput(
            group_id=g.group_id,
            host=g.host,
            prefix=g.prefix,
            mappings=[m.name for m in g.mappings],
        )
        mapping_groups.append(mg)

    with open(os.path.join(testdata_dir, f"{test_case}_out.yaml"), "r") as f:
        out = yaml.safe_load(f)

    expected_output = [MappingGroupTestOutput(**group_yaml) for group_yaml in out["mapping_groups"]]
    assert sorted(mapping_groups, key=lambda g: g.group_id) == sorted(
        expected_output, key=lambda g: g.group_id
    )
