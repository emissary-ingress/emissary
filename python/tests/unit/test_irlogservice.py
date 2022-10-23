import logging
from typing import Literal

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("emissary-ingress")

from ambassador import IR, Config, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler
from tests.utils import default_listener_manifests

SERVICE_NAME = "cool-log-svcname"


def _get_log_config(yaml, driver: Literal["http", "tcp"]):
    for listener in yaml["static_resources"]["listeners"]:
        for filter_chain in listener["filter_chains"]:
            for f in filter_chain["filters"]:
                for log_filter in f["typed_config"]["access_log"]:
                    if log_filter["name"] == f"envoy.access_loggers.{driver}_grpc":
                        return log_filter
    return False


def _get_envoy_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return EnvoyConfig.generate(ir)


def _get_logfilter_http_default_conf():
    return {
        "@type": f"type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.HttpGrpcAccessLogConfig",
        "common_config": {
            "transport_api_version": "V3",
            "log_name": "logservice",
            "grpc_service": {
                "envoy_grpc": {"cluster_name": "cluster_logging_cool_log_svcname_default"}
            },
            "buffer_flush_interval": "1s",
            "buffer_size_bytes": 16384,
        },
        "additional_request_headers_to_log": [],
        "additional_response_headers_to_log": [],
        "additional_response_trailers_to_log": [],
    }


def _get_logfilter_tcp_default_conf():
    return {
        "@type": f"type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.TcpGrpcAccessLogConfig",
        "common_config": {
            "transport_api_version": "V3",
            "log_name": "logservice",
            "grpc_service": {
                "envoy_grpc": {"cluster_name": "cluster_logging_cool_log_svcname_default"}
            },
            "buffer_flush_interval": "1s",
            "buffer_size_bytes": 16384,
        },
    }


###################### unit test covering http driver ###########################


@pytest.mark.compilertest
def test_irlogservice_http_defaults():
    """tests defaults for log service when http driver is used and ensures that transport protocol is v3"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: http
  driver_config: {}
  grpc: true
"""
    )

    driver: Literal["http", "tcp"] = "http"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)

    assert conf == False

    errors = econf.ir.aconf.errors
    assert "ir.logservice" in errors
    assert (
        errors["ir.logservice"][0]["error"]
        == 'LogService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_irlogservice_http_default_overrides():
    """tests default overrides for log service and ensures that transport protocol is v3"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: http
  grpc: true
  protocol_version: "v3"
  flush_interval_time: 33
  flush_interval_byte_size: 9999
  driver_config:
    additional_log_headers:
    - header_name: "x-dino-power"
    - header_name: "x-dino-request-power"
      during_request: true
      during_response: false
      during_trailer: false
    - header_name: "x-dino-response-power"
      during_request: false
      during_response: true
      during_trailer: false
    - header_name: "x-dino-trailer-power"
      during_request: false
      during_response: false
      during_trailer: true
"""
    )

    driver: Literal["http", "tcp"] = "http"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)
    assert conf

    config = _get_logfilter_http_default_conf()
    config["common_config"]["buffer_flush_interval"] = "33s"
    config["common_config"]["buffer_size_bytes"] = 9999
    config["additional_request_headers_to_log"] = ["x-dino-power", "x-dino-request-power"]
    config["additional_response_headers_to_log"] = ["x-dino-power", "x-dino-response-power"]
    config["additional_response_trailers_to_log"] = ["x-dino-power", "x-dino-trailer-power"]

    assert conf.get("typed_config") == config

    assert "ir.logservice" not in econf.ir.aconf.errors


@pytest.mark.compilertest
def test_irlogservice_http_v2():
    """ensures that no longer supported v2 transport protocol is defaulted to v3"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: http
  driver_config: {}
  grpc: true
  protocol_version: "v2"
"""
    )

    driver: Literal["http", "tcp"] = "http"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)

    assert conf == False

    errors = econf.ir.aconf.errors
    assert "ir.logservice" in errors
    assert (
        errors["ir.logservice"][0]["error"]
        == 'LogService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_irlogservice_http_v3():
    """ensures that when transport protocol v3 is provided, nothing is logged"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: http
  driver_config: {}
  grpc: true
  protocol_version: "v3"
"""
    )

    driver: Literal["http", "tcp"] = "http"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)

    assert conf
    assert conf.get("typed_config") == _get_logfilter_http_default_conf()

    assert "ir.logservice" not in econf.ir.aconf.errors


############### unit test covering tcp driver #######################
@pytest.mark.compilertest
def test_irlogservice_tcp_defaults():
    """tests defaults for log service using tcp driver and ensures that transport protocol is v3"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: tcp
  driver_config: {}
  grpc: true
"""
    )

    driver: Literal["http", "tcp"] = "tcp"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)

    assert conf == False

    errors = econf.ir.aconf.errors
    assert "ir.logservice" in errors
    assert (
        errors["ir.logservice"][0]["error"]
        == 'LogService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_irlogservice_tcp_default_overrides():
    """tests default overrides for log service with tcp driver and ensures that transport protocol is v3"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: tcp
  driver_config: {}
  grpc: true
  protocol_version: "v3"
  flush_interval_time: 33
  flush_interval_byte_size: 9999
"""
    )

    driver: Literal["http", "tcp"] = "tcp"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)
    assert conf

    config = _get_logfilter_tcp_default_conf()
    config["common_config"]["buffer_flush_interval"] = "33s"
    config["common_config"]["buffer_size_bytes"] = 9999

    assert conf.get("typed_config") == config

    assert "ir.logservice" not in econf.ir.aconf.errors


@pytest.mark.compilertest
def test_irlogservice_tcp_v2():
    """ensures that no longer supported v2 transport protocol is defaulted to v3"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: tcp
  driver_config: {}
  grpc: true
  protocol_version: "v2"
"""
    )

    driver: Literal["http", "tcp"] = "tcp"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)

    assert conf == False

    errors = econf.ir.aconf.errors
    assert "ir.logservice" in errors
    assert (
        errors["ir.logservice"][0]["error"]
        == 'LogService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_irlogservice_tcp_v3():
    """ensures that when transport protocol v3 is provided, nothing is logged"""

    yaml = (
        """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
metadata:
  name: myls
  namespace: default
spec:
  service: """
        + SERVICE_NAME
        + """
  driver: tcp
  driver_config: {}
  grpc: true
  protocol_version: "v3"
"""
    )

    driver: Literal["http", "tcp"] = "tcp"

    econf = _get_envoy_config(yaml)
    conf = _get_log_config(econf.as_dict(), driver)

    assert conf
    assert conf.get("typed_config") == _get_logfilter_tcp_default_conf()

    assert "ir.logservice" not in econf.ir.aconf.errors
