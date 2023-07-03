import logging
import typing
from typing import Any, Dict, List

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador import IR, Config, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler
from tests.utils import default_listener_manifests


def _get_cluster_config(clusters, name):
    for cluster in clusters:
        # we're only interested in the cluster for the provided name
        if cluster["name"] == name:
            return cluster
        else:
            continue
    return False


def _get_envoy_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    econf = EnvoyConfig.generate(ir)
    assert econf, "could not create an econf"
    return econf


@pytest.mark.compilertest
def test_healthcheck():
    baseYaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: healthchecktest
  namespace: default
spec:
  hostname: '*'
  service: coolsvcname
  prefix: /test
  resolver: endpoint
  health_checks: {}
"""

    noEndpointYaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: healthchecktest
  namespace: default
spec:
  hostname: '*'
  service: coolsvcname
  prefix: /test
  health_checks: {}
"""
    testcases: List[Dict[str, Any]] = [
        {  # Test that the fields we leave out get assigned default values
            "name": "healthcheck_defaults",
            "input": baseYaml.format([{"health_check": {"http": {"path": "/health"}}}]),
            # When fields such as healthy_threshold that have default values
            # are not supplied by the expected field then we will check that they have their default values
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                    },
                },
            ],
        },
        {  # Check that we can override all of the fields that get default values
            "name": "healthcheck_no_defaults",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                            }
                        },
                        "healthy_threshold": 5,
                        "unhealthy_threshold": 5,
                        "interval": "10s",
                        "timeout": "15s",
                    }
                ]
            ),
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                    },
                    "healthy_threshold": 5,
                    "unhealthy_threshold": 5,
                    "interval": "10s",
                    "timeout": "15s",
                },
            ],
        },
        {  # Check that both a grpc and http healthceck can be used at the same time
            "name": "healthcheck_http_plus_grpc",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "grpc": {
                                "upstream_name": "coolsvcname.default",
                            }
                        }
                    },
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                            }
                        }
                    },
                ]
            ),
            "expected": [
                {
                    "grpc_health_check": {
                        "service_name": "coolsvcname.default",
                    }
                },
                {
                    "http_health_check": {
                        "path": "/health",
                    }
                },
            ],
        },
        {  # Check that we can set the authority on grpc health checks
            "name": "healthcheck_grpc_authority",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "grpc": {
                                "upstream_name": "coolsvcname.default",
                                "authority": "dummy.example",
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {
                    "grpc_health_check": {
                        "service_name": "coolsvcname.default",
                        "authority": "dummy.example",
                    }
                },
            ],
        },
        {  # Check that we can set add/remove headers for a http health check
            "name": "healthcheck_grpc_authority",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "grpc": {
                                "upstream_name": "coolsvcname.default",
                                "authority": "dummy.example",
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {
                    "grpc_health_check": {
                        "service_name": "coolsvcname.default",
                        "authority": "dummy.example",
                    }
                },
            ],
        },
        {  # check that we can set hostname on a http health check
            "name": "healthcheck_http_hostname",
            "input": baseYaml.format(
                [{"health_check": {"http": {"path": "/health", "hostname": "dummy.example"}}}]
            ),
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                        # hostname becomes host in the econf
                        "host": "dummy.example",
                    }
                },
            ],
        },
        {  # check that we can set expected statuses on a http health check
            "name": "healthcheck_http_statuses",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                                "expected_statuses": [
                                    {"min": 101, "max": 199},
                                    {"min": 201, "max": 299},
                                ],
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                        # We increment the end by 1 in the backend since
                        # envoy treats the end as being excluded (which adds confusion so lets just make the start and end inclusive)
                        "expected_statuses": [
                            {"start": 101, "end": 200},
                            {"start": 201, "end": 300},
                        ],
                    }
                },
            ],
        },
        {  # check that an invalid expected status is ignored
            "name": "healthcheck_http_statuses_invalid",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                                "expected_statuses": [
                                    # this one is invalid since the start is larger than the end so we should just drop it.
                                    {"min": 300, "max": 100},
                                    {"min": 201, "max": 299},
                                ],
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                        # We increment the end by 1 in the backend since
                        # envoy treats the end as being excluded (which adds confusion so lets just make the start and end inclusive)
                        "expected_statuses": [
                            {"start": 201, "end": 300},
                        ],
                    }
                },
            ],
        },
        {  # check that if all the expected statuses are invalid then we don't set the field
            "name": "healthcheck_http_statuses_invalid_all",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                                "expected_statuses": [
                                    # these are both invalid so the whole field should be ignored
                                    {"min": 300, "max": 100},
                                    {"min": 400, "max": 300},
                                ],
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {"http_health_check": {"path": "/health"}},
            ],
        },
        {  # check that append headers is true when not provided
            "name": "healthcheck_http_add_headers",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                                "add_request_headers": {
                                    "fruit-one": {"append": False, "value": "banana"},
                                    "fruit-two": {"append": True, "value": "orange"},
                                    "fruit-three": {"value": "peach"},
                                },
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                        "request_headers_to_add": [
                            {"header": {"key": "fruit-one", "value": "banana"}, "append": False},
                            {"header": {"key": "fruit-two", "value": "orange"}, "append": True},
                            {"header": {"key": "fruit-three", "value": "peach"}, "append": True},
                        ],
                    }
                },
            ],
        },
        {  # check remove headers
            "name": "healthcheck_http_remove_headers",
            "input": baseYaml.format(
                [
                    {
                        "health_check": {
                            "http": {
                                "path": "/health",
                                "remove_request_headers": ["fruit-one", "fruit-two", "fruit-three"],
                            }
                        }
                    }
                ]
            ),
            "expected": [
                {
                    "http_health_check": {
                        "path": "/health",
                        "request_headers_to_remove": ["fruit-one", "fruit-two", "fruit-three"],
                    }
                },
            ],
        },
        {  # Test that we throw out the health check config when there is no endpoint resolver
            "name": "healthcheck_no_endpoint",
            "input": noEndpointYaml.format([{"health_check": {"http": {"path": "/health"}}}]),
            "expected": None,
        },
    ]

    for case in testcases:
        caseYaml = case["input"]
        testName = case["name"]
        econf = _get_envoy_config(caseYaml)
        cluster = _get_cluster_config(econf.clusters, "cluster_coolsvcname_default")
        assert cluster != False

        expectedChecks = case["expected"]
        if expectedChecks is None:
            assert "health_checks" not in cluster, "Failed healthcheck test {}".format(testName)
        else:
            assert "health_checks" in cluster, "Failed healthcheck test {}".format(testName)

            hc = cluster["health_checks"]
            for i in range(0, len(hc)):
                actual = hc[i]
                expected = expectedChecks[i]

                check_healthcheck_defaults(expected, actual, testName)

                if "grpc_health_check" in expected:
                    try:
                        check_grpc_healthcheck(
                            expected["grpc_health_check"], actual["grpc_health_check"], testName
                        )
                    except KeyError:
                        assert True == False, "Failed healthcheck test {}".format(testName)
                if "http_health_check" in expected:
                    try:
                        check_http_healthcheck(
                            expected["http_health_check"], actual["http_health_check"], testName
                        )
                    except KeyError:
                        assert True == False, "Failed healthcheck test {}".format(testName)


# Runs a bunch of assert statments to check that the expected
# healthcheck fields match the actual ones
def check_healthcheck_defaults(expected, actual, testName):
    # check all the default values unless we overrode them
    if "healthy_threshold" in expected:
        assert (
            actual["healthy_threshold"] == expected["healthy_threshold"]
        ), "Failed healthcheck test {}".format(testName)
    else:
        assert actual["healthy_threshold"] == 1, "Failed healthcheck test {}".format(testName)

    if "interval" in expected:
        assert actual["interval"] == expected["interval"], "Failed healthcheck test {}".format(
            testName
        )
    else:
        assert actual["interval"] == "5s", "Failed healthcheck test {}".format(testName)

    if "timeout" in expected:
        assert actual["timeout"] == expected["timeout"], "Failed healthcheck test {}".format(
            testName
        )
    else:
        assert actual["timeout"] == "3s", "Failed healthcheck test {}".format(testName)

    if "unhealthy_threshold" in expected:
        assert (
            actual["unhealthy_threshold"] == expected["unhealthy_threshold"]
        ), "Failed healthcheck test {}".format(testName)
    else:
        assert actual["unhealthy_threshold"] == 2, "Failed healthcheck test {}".format(testName)


# Runs a bunch of assert statments to check that the expected
# grpc health check matches the actual one.
def check_grpc_healthcheck(expected, actual, testName):
    if expected is not None:
        assert actual is not None, "Failed healthcheck test {}".format(testName)

        assert (
            actual["service_name"] == expected["service_name"]
        ), "Failed healthcheck test {}".format(testName)

        if "authority" in expected:
            assert (
                actual["authority"] == expected["authority"]
            ), "Failed healthcheck test {}".format(testName)


# Runs a bunch of assert statments to check that the expected
# http health check matches the actual one.
def check_http_healthcheck(expected, actual, testName):
    if expected is not None:
        assert actual is not None, "Failed healthcheck test {}".format(testName)

        assert actual["path"] == expected["path"], "Failed healthcheck test {}".format(testName)

        if "host" in expected:
            assert actual["host"] == expected["host"], "Failed healthcheck test {}".format(testName)

        if "request_headers_to_remove" in expected:
            assert (
                actual["request_headers_to_remove"] == expected["request_headers_to_remove"]
            ), "Failed healthcheck test {}".format(testName)

        if "request_headers_to_add" in expected:
            assert (
                actual["request_headers_to_add"] == expected["request_headers_to_add"]
            ), "Failed healthcheck test {}".format(testName)

        if "expected_statuses" in expected:
            assert (
                actual["expected_statuses"] == expected["expected_statuses"]
            ), "Failed healthcheck test {}".format(testName)
