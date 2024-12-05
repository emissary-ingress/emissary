import datetime
import json
import logging
import os
import subprocess
import tempfile
from base64 import b64encode
from collections import namedtuple

import pytest
from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import dsa, ec, rsa

from ambassador import IR, Cache, Config, EnvoyConfig
from ambassador.compile import Compile
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler, parse_bool

logger = logging.getLogger("ambassador")


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


def default_http3_listener_manifest():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-http3-8443
  namespace: default
spec:
  port: 8443
  protocolStack:
    - TLS
    - HTTP
    - UDP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL  
  """


def default_udp_listener_manifest():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-udp-8443
  namespace: default
spec:
  port: 8443
  protocolStack:
    - TLS
    - UDP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL  
  """


def default_tcp_listener_manifest():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-tcp-8443
  namespace: default
spec:
  port: 8443
  protocolStack:
    - TLS
    - TCP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL  
  """


def module_and_mapping_manifests(module_confs, mapping_confs):
    yaml = (
        default_listener_manifests()
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:"""
    )
    if module_confs:
        for module_conf in module_confs:
            yaml = (
                yaml
                + """
    {}
""".format(
                    module_conf
                )
            )
    else:
        yaml = yaml + " {}\n"

    yaml = (
        yaml
        + """
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
    )
    if mapping_confs:
        for mapping_conf in mapping_confs:
            yaml = (
                yaml
                + """
  {}""".format(
                    mapping_conf
                )
            )
    return yaml


def _require_no_errors(ir: IR):
    assert ir.aconf.errors == {}, f"{repr(ir.aconf.errors)}"


def _secret_handler():
    source_root = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-source")
    cache_dir = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
    return NullSecretHandler(logger, source_root.name, cache_dir.name, "fake")


def get_envoy_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return EnvoyConfig.generate(ir)


def compile_with_cachecheck(yaml, errors_ok=False):
    # Compile with and without a cache. Neither should produce errors.
    cache = Cache(logger)
    secret_handler = _secret_handler()
    r1 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler)
    r2 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, cache=cache)

    if not errors_ok:
        _require_no_errors(r1["ir"])
        _require_no_errors(r2["ir"])

    # Both should produce equal Envoy config as sorted json.
    r1j = json.dumps(r1["xds"].as_dict(), sort_keys=True, indent=2)
    r2j = json.dumps(r2["xds"].as_dict(), sort_keys=True, indent=2)
    assert r1j == r2j

    # All good.
    return r1


EnvoyFilterInfo = namedtuple("EnvoyFilterInfo", ["name", "type"])

EnvoyHCMInfo = EnvoyFilterInfo(
    name="envoy.filters.network.http_connection_manager",
    type="type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
)

EnvoyTCPInfo = EnvoyFilterInfo(
    name="envoy.filters.network.tcp_proxy",
    type="type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy",
)


def econf_compile(yaml):
    compiled = compile_with_cachecheck(yaml)
    return compiled["xds"].as_dict()


def econf_foreach_listener(econf, fn, listener_count=1):
    listeners = econf["static_resources"]["listeners"]

    wanted_plural = "" if (listener_count == 1) else "s"
    assert (
        len(listeners) == listener_count
    ), f"Expected {listener_count} listener{wanted_plural}, got {len(listeners)}"

    for listener in listeners:
        fn(listener)


def econf_foreach_listener_chain(
    listener, fn, chain_count=2, need_name=None, need_type=None, dump_info=None
):
    # We need a specific number of filter chains. Normally it's 2,
    # since the compiler tests don't generally supply Listeners or Hosts,
    # so we get secure and insecure chains.
    filter_chains = listener["filter_chains"]

    if dump_info:
        dump_info(filter_chains)

    wanted_plural = "" if (chain_count == 1) else "s"
    assert (
        len(filter_chains) == chain_count
    ), f"Expected {chain_count} filter chain{wanted_plural}, got {len(filter_chains)}"

    for chain in filter_chains:
        # We expect one filter on this chain.
        filters = chain["filters"]
        got_count = len(filters)
        got_plural = "" if (got_count == 1) else "s"
        assert got_count == 1, f"Expected just one filter, got {got_count} filter{got_plural}"

        # The http connection manager is the only filter on the chain from the one and only vhost.
        filter = filters[0]

        if need_name:
            assert filter["name"] == need_name

        typed_config = filter["typed_config"]

        if need_type:
            assert (
                typed_config["@type"] == need_type
            ), f"bad type: got {repr(typed_config['@type'])} but expected {repr(need_type)}"

        fn(typed_config)


def econf_foreach_hcm(econf, fn, chain_count=2):
    for listener in econf["static_resources"]["listeners"]:
        if listener["name"].startswith("ambassador-listener-ready"):
            # don't want to test the ready listener since it's different from the default 8080/8443
            # listeners and is already tested in test_ready.py
            continue
        hcm_info = EnvoyHCMInfo

        econf_foreach_listener_chain(
            listener, fn, chain_count=chain_count, need_name=hcm_info.name, need_type=hcm_info.type
        )


def econf_foreach_cluster(econf, fn, name="cluster_httpbin_default"):
    for cluster in econf["static_resources"]["clusters"]:
        if cluster["name"] != name:
            continue

        found_cluster = True
        r = fn(cluster)
        if not r:
            break
    assert found_cluster


def assert_valid_envoy_config(config_dict, extra_dirs=[]):
    with tempfile.TemporaryDirectory() as tmpdir:
        econf = open(os.path.join(tmpdir, "econf.json"), "xt")
        econf.write(json.dumps(config_dict))
        econf.close()
        img = os.environ.get("ENVOY_DOCKER_TAG")
        assert img
        cmd = [
            "docker",
            "run",
            "--rm",
            f"--volume={tmpdir}:/ambassador:ro",
            *[f"--volume={extra_dir}:{extra_dir}:ro" for extra_dir in extra_dirs],
            img,
            "/usr/local/bin/envoy-static-stripped",
            "--config-path",
            "/ambassador/econf.json",
            "--mode",
            "validate",
        ]
        p = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        if p.returncode != 0:
            print(p.stdout.decode())
        p.check_returncode()


def create_crl_pem_b64(issuerCert, issuerKey, revokedCerts):
    cert = x509.load_pem_x509_certificate(issuerCert.encode("utf-8"))
    key = serialization.load_pem_private_key(issuerKey.encode("utf-8"), password=None)

    assert isinstance(key, (rsa.RSAPrivateKey, dsa.DSAPrivateKey, ec.EllipticCurvePrivateKey))

    when = datetime.datetime(
        year=2022, month=5, day=16, hour=1, minute=1, second=1, tzinfo=datetime.timezone.utc
    )

    thirty_days_out = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(days=30)

    crl_builder = (
        x509.CertificateRevocationListBuilder()
        .issuer_name(cert.subject)
        .last_update(when)
        .next_update(thirty_days_out)
    )

    for revokedCert in revokedCerts:
        clientCert = x509.load_pem_x509_certificate(revokedCert.encode("utf-8"))
        revoked_cert = (
            x509.RevokedCertificateBuilder()
            .serial_number(clientCert.serial_number)
            .revocation_date(when)
            .build()
        )
        crl_builder = crl_builder.add_revoked_certificate(revoked_cert)

    crl = crl_builder.sign(private_key=key, algorithm=hashes.SHA256())
    return b64encode(crl.public_bytes(serialization.Encoding.PEM)).decode("utf-8")


def skip_edgestack():
    isEdgeStack = parse_bool(os.environ.get("EDGE_STACK", "false"))

    return pytest.mark.skipif(
        isEdgeStack,
        reason=f"Skipping because EdgeStack behaves differently and tested separately",
    )


def edgestack():
    isEdgeStack = parse_bool(os.environ.get("EDGE_STACK", "false"))
    return pytest.mark.skipif(
        not isEdgeStack,
        reason=f"Skipping because this is an EdgeStack specific case and is tested separately",
    )
