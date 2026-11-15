from typing import List, Optional

import pytest

from tests.utils import econf_compile

HOSTNAME = "sigalg.example.com"
SIGALGS = ["ecdsa_secp256r1_sha256", "ed25519"]


def _manifests(sigalg_key: Optional[str]) -> str:
    """A Listener, a Host with inline spec.tls, and a Mapping. min_tls_version is always
    set so the inline TLS branch runs and tls_params exists even when signature
    algorithms are omitted."""
    sigalg_line = f"\n    {sigalg_key}: {SIGALGS}" if sigalg_key else ""
    return f"""
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
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: sigalg-host
  namespace: default
spec:
  hostname: {HOSTNAME}
  acmeProvider:
    authority: none
  tlsSecret:
    name: sigalg-cert
  tls:
    min_tls_version: v1.2{sigalg_line}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: sigalg-mapping
  namespace: default
spec:
  hostname: "*"
  prefix: /
  service: target
"""


def _tls_params_for_host(econf: dict, hostname: str) -> List[dict]:
    """Return tls_params from every filter chain that matches the Host's SNI."""
    found = []
    for listener in econf["static_resources"]["listeners"]:
        for chain in listener.get("filter_chains", []):
            if hostname not in chain.get("filter_chain_match", {}).get("server_names", []):
                continue
            ctx = chain["transport_socket"]["typed_config"]["common_tls_context"]
            found.append(ctx.get("tls_params", {}))
    assert found, f"no TLS filter chain matched {hostname}"
    return found


@pytest.mark.compilertest
@pytest.mark.parametrize("sigalg_key", ["signature_algorithms", "signatureAlgorithms"])
def test_host_inline_tls_signature_algorithms(sigalg_key):
    """Inline Host.spec.tls signature algorithms, by snake_case name or camelCase
    alias, reach common_tls_context.tls_params.signature_algorithms."""
    econf = econf_compile(_manifests(sigalg_key))

    for params in _tls_params_for_host(econf, HOSTNAME):
        assert params.get("tls_minimum_protocol_version") == "TLSv1_2"
        assert params.get("signature_algorithms") == SIGALGS


@pytest.mark.compilertest
def test_host_inline_tls_signature_algorithms_omitted():
    """Without the field, no explicit list is emitted, so Envoy keeps its defaults."""
    econf = econf_compile(_manifests(None))

    for params in _tls_params_for_host(econf, HOSTNAME):
        assert params.get("tls_minimum_protocol_version") == "TLSv1_2"
        assert "signature_algorithms" not in params
