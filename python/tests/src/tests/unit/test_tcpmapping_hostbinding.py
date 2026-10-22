from typing import List

from tests.utils import compile_with_cachecheck, econf_compile

# A TLS TCPMapping gets its certificate from a Host whose hostname matches the
# TCPMapping's host. The Listener's hostBinding decides which Hosts it takes,
# so a Host that the Listener does not bind must not hand it a certificate.
# HTTP chains already honor this; these tests cover TCP chains (issue #5732).


def _listener(selector: str) -> str:
    return f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: tcp-listener
  namespace: default
spec:
  port: 6379
  protocol: TLS
  securityModel: SECURE
  hostBinding:
    namespace:
      from: ALL
{selector}
"""


SELECT_TCP_HOSTS = """    selector:
      matchLabels:
        tcp: "yes"
"""

# Both Hosts match tcp.example.com. Hosts are tried in hostname order, so
# without hostBinding filtering "*.example.com" always wins.
WILDCARD_HOST = """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: wildcard-host
  namespace: default
spec:
  hostname: "*.example.com"
  acmeProvider:
    authority: none
  tlsSecret:
    name: secret-wildcard
"""

TCP_HOST = """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: tcp-host
  namespace: default
  labels:
    tcp: "yes"
spec:
  hostname: "tcp.example.com"
  acmeProvider:
    authority: none
  tlsSecret:
    name: secret-tcp
"""

TCP_MAPPING = """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: tcp-backend
  namespace: default
spec:
  port: 6379
  host: tcp.example.com
  service: redis:6379
"""


def _tcp_cert_files(econf) -> List[str]:
    """Return the certificate files served by the filter chains on port 6379."""
    files: List[str] = []

    for listener in econf["static_resources"]["listeners"]:
        if listener["address"]["socket_address"]["port_value"] != 6379:
            continue

        for chain in listener.get("filter_chains", []):
            tls = chain.get("transport_socket", {}).get("typed_config", {})
            for cert in tls.get("common_tls_context", {}).get("tls_certificates", []):
                files.append(cert["certificate_chain"]["filename"])

    return files


def test_tcpmapping_uses_host_selected_by_listener():
    econf = econf_compile(_listener(SELECT_TCP_HOSTS) + WILDCARD_HOST + TCP_HOST + TCP_MAPPING)

    files = _tcp_cert_files(econf)
    assert len(files) == 1, f"expected one certificate on port 6379, got {files}"
    assert "/secret-tcp/" in files[0], f"expected the bound Host's secret, got {files[0]}"


def test_tcpmapping_unrestricted_listener_still_takes_any_host():
    econf = econf_compile(_listener("") + WILDCARD_HOST + TCP_HOST + TCP_MAPPING)

    files = _tcp_cert_files(econf)
    assert len(files) == 1, f"expected one certificate on port 6379, got {files}"
    assert "/secret-wildcard/" in files[0], f"expected the first matching Host, got {files[0]}"


def test_tcpmapping_ignores_hosts_the_listener_does_not_bind():
    # The only Host that matches tcp.example.com is not bound to this Listener,
    # so there's no TLS configuration for the TCPMapping and it is disabled.
    r = compile_with_cachecheck(
        _listener(SELECT_TCP_HOSTS) + WILDCARD_HOST + TCP_MAPPING, errors_ok=True
    )

    files = _tcp_cert_files(r["xds"].as_dict())
    assert files == [], f"expected no certificate on port 6379, got {files}"

    errors = r["ir"].aconf.errors
    assert any(
        "No matching TLSContext found" in e.get("error", "")
        for errs in errors.values()
        for e in errs
    ), f"expected the TCPMapping to be disabled, got errors {errors}"


UNBOUND_TCP_HOST = TCP_HOST.replace('tcp: "yes"', 'tcp: "no"')

TCP_TLSCONTEXT = """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: tcp-context
  namespace: default
spec:
  hosts:
  - tcp.example.com
  secret: secret-context
"""


def test_tcpmapping_ignores_tlscontext_generated_by_unbound_host():
    # The unbound Host matches tcp.example.com exactly, so the TLSContext it
    # generates lists that hostname. The Listener must not serve it either.
    r = compile_with_cachecheck(
        _listener(SELECT_TCP_HOSTS) + UNBOUND_TCP_HOST + TCP_MAPPING, errors_ok=True
    )

    files = _tcp_cert_files(r["xds"].as_dict())
    assert files == [], f"expected no certificate on port 6379, got {files}"


def test_tcpmapping_still_takes_standalone_tlscontext():
    # TLSContexts are ignored when there are no Hosts at all, so add one that
    # the Listener does not bind and whose hostname is not tcp.example.com.
    econf = econf_compile(
        _listener(SELECT_TCP_HOSTS) + WILDCARD_HOST + TCP_TLSCONTEXT + TCP_MAPPING
    )

    files = _tcp_cert_files(econf)
    assert len(files) == 1, f"expected one certificate on port 6379, got {files}"
    assert "/secret-context/" in files[0], f"expected the TLSContext's secret, got {files[0]}"
