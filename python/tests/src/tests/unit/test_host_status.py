import logging

import pytest

from tests.utils import compile_with_cachecheck, default_listener_manifests

logger = logging.getLogger("ambassador")

HOST_YAML = """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {name}
  namespace: default
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
"""

TLS_HOST_YAML = """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: tls-host
  namespace: default
spec:
  hostname: tls.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: tls-cert
"""

BAD_TLS_HOST_YAML = """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: bad-tls-host
  namespace: default
spec:
  hostname: bad.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: bad-cert
  tls:
    min_tls_version: v9.9
"""

INGRESS_YAML = """
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tls-ingress
  namespace: default
  annotations:
    kubernetes.io/ingress.class: ambassador
spec:
  tls:
  - hosts:
    - ingress.example.com
    secretName: ingress-cert
"""


@pytest.mark.compilertest
@pytest.mark.parametrize("name", ["minimal-host", "host.example.com"])
def test_host_status_ready(name):
    """A valid Host without TLS reports Ready with no certificate source. The dotted
    name checks that the status key keeps the full name."""
    r = compile_with_cachecheck(default_listener_manifests() + HOST_YAML.format(name=name))

    assert r["ir"].k8s_status_updates[f"{name}.default"] == (
        "Host",
        "default",
        {"state": "Ready", "tlsCertificateSource": "None"},
    )


@pytest.mark.compilertest
def test_host_status_ready_with_tls():
    """A Host that terminates TLS from a Secret reports tlsCertificateSource Other."""
    r = compile_with_cachecheck(default_listener_manifests() + TLS_HOST_YAML)

    assert r["ir"].k8s_status_updates["tls-host.default"] == (
        "Host",
        "default",
        {"state": "Ready", "tlsCertificateSource": "Other"},
    )


@pytest.mark.compilertest
def test_host_status_error_has_reason():
    """An invalid Host reports Error with the posted reason, and no timestamp so the
    status stays identical across reconfigures."""
    r = compile_with_cachecheck(default_listener_manifests() + BAD_TLS_HOST_YAML, errors_ok=True)

    kind, namespace, status = r["ir"].k8s_status_updates["bad-tls-host.default"]

    assert (kind, namespace) == ("Host", "default")
    assert status["state"] == "Error"
    assert "Invalid min_tls_version" in status["errorReason"]
    assert "errorTimestamp" not in status


@pytest.mark.compilertest
def test_synthesized_default_host_has_no_status():
    """The fallback default-host has no Kubernetes object, so it gets no status."""
    r = compile_with_cachecheck(default_listener_manifests())

    assert not any(kind == "Host" for kind, _, _ in r["ir"].k8s_status_updates.values())


@pytest.mark.compilertest
def test_ingress_generated_host_has_no_status():
    """Hosts generated from an Ingress have no Host object to update."""
    r = compile_with_cachecheck(default_listener_manifests() + INGRESS_YAML, errors_ok=True)

    assert not any(kind == "Host" for kind, _, _ in r["ir"].k8s_status_updates.values())
