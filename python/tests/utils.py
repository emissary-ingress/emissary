import logging
import json
import os
import subprocess
import requests
import socket
import tempfile
import time
from collections import namedtuple
from retry import retry

import json
import yaml

from ambassador import Cache, IR
from ambassador.compile import Compile
from ambassador.utils import NullSecretHandler

from tests.manifests import cleartext_host_manifest
from tests.kubeutils import apply_kube_artifacts
from tests.runutils import run_and_assert

logger = logging.getLogger("ambassador")

ENVOY_PATH = os.environ.get('ENVOY_PATH', '/usr/local/bin/envoy')

SUPPORTED_ENVOY_VERSIONS = ["V3"]

def zipkin_tracing_service_manifest():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
  name: tracing
  namespace: ambassador
spec:
  service: zipkin:9411
  driver: zipkin
  config: {}
"""

def default_listener_manifests():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-8080
  namespace: default
spec:
  port: 8080
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-8443
  namespace: default
spec:
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
"""

def module_and_mapping_manifests(module_confs, mapping_confs):
    yaml = default_listener_manifests() + """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:"""
    if module_confs:
        for module_conf in module_confs:
            yaml = yaml + """
    {}
""".format(module_conf)
    else:
        yaml = yaml + " {}\n"

    yaml = yaml + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
  prefix: /httpbin/
  service: httpbin"""
    if mapping_confs:
        for mapping_conf in mapping_confs:
            yaml = yaml + """
  {}""".format(mapping_conf)
    return yaml

def _require_no_errors(ir: IR):
    assert ir.aconf.errors == {}

def _secret_handler():
    source_root = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-source")
    cache_dir = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
    return NullSecretHandler(logger, source_root.name, cache_dir.name, "fake")

def compile_with_cachecheck(yaml, envoy_version="V3", errors_ok=False):
    # Compile with and without a cache. Neither should produce errors.
    cache = Cache(logger)
    secret_handler = _secret_handler()
    r1 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, envoy_version=envoy_version)
    r2 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, cache=cache,
            envoy_version=envoy_version)

    if not errors_ok:
        _require_no_errors(r1["ir"])
        _require_no_errors(r2["ir"])

    # Both should produce equal Envoy config as sorted json.
    r1j = json.dumps(r1[envoy_version.lower()].as_dict(), sort_keys=True, indent=2)
    r2j = json.dumps(r2[envoy_version.lower()].as_dict(), sort_keys=True, indent=2)
    assert r1j == r2j

    # All good.
    return r1

EnvoyFilterInfo = namedtuple('EnvoyFilterInfo', [ 'name', 'type' ])

EnvoyHCMInfo = {
    "V3": EnvoyFilterInfo(
        name="envoy.filters.network.http_connection_manager",
        type="type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
    ),
}

EnvoyTCPInfo = {
    "V3": EnvoyFilterInfo(
        name="envoy.filters.network.tcp_proxy",
        type="type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy"
    ),
}

def econf_compile(yaml, envoy_version="V3"):
    compiled = compile_with_cachecheck(yaml, envoy_version=envoy_version)
    return compiled[envoy_version.lower()].as_dict()

def econf_foreach_listener(econf, fn, envoy_version='V3', listener_count=1):
    listeners = econf['static_resources']['listeners']

    wanted_plural = "" if (listener_count == 1) else "s"
    assert len(listeners) == listener_count, f"Expected {listener_count} listener{wanted_plural}, got {len(listeners)}"

    for listener in listeners:
        fn(listener, envoy_version)

def econf_foreach_listener_chain(listener, fn, chain_count=2, need_name=None, need_type=None, dump_info=None):
    # We need a specific number of filter chains. Normally it's 2,
    # since the compiler tests don't generally supply Listeners or Hosts,
    # so we get secure and insecure chains.
    filter_chains = listener['filter_chains']

    if dump_info:
        dump_info(filter_chains)

    wanted_plural = "" if (chain_count == 1) else "s"
    assert len(filter_chains) == chain_count, f"Expected {chain_count} filter chain{wanted_plural}, got {len(filter_chains)}"

    for chain in filter_chains:
        # We expect one filter on this chain.
        filters = chain['filters']
        got_count = len(filters)
        got_plural = "" if (got_count == 1) else "s"
        assert got_count == 1, f"Expected just one filter, got {got_count} filter{got_plural}"

        # The http connection manager is the only filter on the chain from the one and only vhost.
        filter = filters[0]

        if need_name:
            assert filter['name'] == need_name

        typed_config = filter['typed_config']

        if need_type:
            assert typed_config['@type'] == need_type, f"bad type: {typed_config['@type']}"

        fn(typed_config)

def econf_foreach_hcm(econf, fn, envoy_version='V3', chain_count=2):
    for listener in econf['static_resources']['listeners']:
        hcm_info = EnvoyHCMInfo[envoy_version]

        econf_foreach_listener_chain(
            listener, fn, chain_count=chain_count,
            need_name=hcm_info.name, need_type=hcm_info.type)

def econf_foreach_cluster(econf, fn, name='cluster_httpbin_default'):
    for cluster in econf['static_resources']['clusters']:
        if cluster['name'] != name:
            continue

        found_cluster = True
        r = fn(cluster)
        if not r:
            break
    assert found_cluster

def assert_valid_envoy_config(config_dict):
    with tempfile.NamedTemporaryFile() as temp:
        temp.write(bytes(json.dumps(config_dict), encoding = 'utf-8'))
        temp.flush()
        f_name = temp.name
        cmd = [ENVOY_PATH, '--config-path', f_name, '--mode', 'validate']
        p = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        if p.returncode != 0:
            print(p.stdout)
        p.check_returncode()
