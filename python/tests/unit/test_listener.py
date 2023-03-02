import os
from dataclasses import dataclass
from typing import List, Optional

import pytest
import yaml

from ambassador import IR
from ambassador.compile import Compile
from ambassador.config import Config
from ambassador.envoy import EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import EmptySecretHandler
from tests.utils import (
    Compile,
    default_http3_listener_manifest,
    econf_compile,
    logger,
    skip_edgestack,
)


def _ensure_alt_svc_header_injected(listener, expectedAltSvc):
    """helper function to ensure that the alt-svc header is getting injected properly"""
    filter_chains = listener["filter_chains"]

    for filter_chain in filter_chains:
        hcm_typed_config = filter_chain["filters"][0]["typed_config"]
        virtual_hosts = hcm_typed_config["route_config"]["virtual_hosts"]
        for host in virtual_hosts:
            response_headers_to_add = host["response_headers_to_add"]
            assert len(response_headers_to_add) == 1
            header = response_headers_to_add[0]["header"]
            assert header["key"] == "alt-svc"
            assert header["value"] == expectedAltSvc


def _verify_no_added_response_headers(listener):
    """helper function to ensure response_headers_to_add do not exist"""
    filter_chains = listener["filter_chains"]

    for filter_chain in filter_chains:
        hcm_typed_config = filter_chain["filters"][0]["typed_config"]
        virtual_hosts = hcm_typed_config["route_config"]["virtual_hosts"]
        for host in virtual_hosts:
            assert "response_headers_to_add" not in host


def _generateListener(name: str, protocol: Optional[str], protocol_stack: Optional[List[str]]):
    yaml = f"""
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
    name: {name}
    namespace: ambassador
spec:
    port: 8443
    {f"protocolStack: {protocol_stack}" if protocol == None else f"protocol: {protocol}"}
    securityModel: XFP
    hostBinding:
        namespace:
            from: ALL
"""
    return yaml


class TestListener:
    @dataclass
    class SocketProtocolTestCase:
        name: str
        protocol: Optional[str]
        protocolStack: Optional[List[str]]
        expectedSocketProtocol: Optional[str]

    @pytest.mark.compilertest
    @pytest.mark.parametrize(
        "case",
        ids=lambda case: case.name,
        argvalues=[
            # test with emissary defined protcolStacks via pre-definied protocol enum
            SocketProtocolTestCase(
                name="http_protocol",
                protocol="HTTP",
                protocolStack=None,
                expectedSocketProtocol="TCP",
            ),
            SocketProtocolTestCase(
                name="https_protocol",
                protocol="HTTPS",
                protocolStack=None,
                expectedSocketProtocol="TCP",
            ),
            SocketProtocolTestCase(
                name="httpproxy_protocol",
                protocol="HTTPPROXY",
                protocolStack=None,
                expectedSocketProtocol="TCP",
            ),
            SocketProtocolTestCase(
                name="httpsproxy_protocol",
                protocol="HTTPSPROXY",
                protocolStack=None,
                expectedSocketProtocol="TCP",
            ),
            SocketProtocolTestCase(
                name="tcp_protocol",
                protocol="TCP",
                protocolStack=None,
                expectedSocketProtocol="TCP",
            ),
            SocketProtocolTestCase(
                name="tls_protocol",
                protocol="TLS",
                protocolStack=None,
                expectedSocketProtocol="TCP",
            ),
            # test with custom stacks
            SocketProtocolTestCase(
                name="tcp_stack",
                protocol=None,
                protocolStack=["TLS", "HTTP", "TCP"],
                expectedSocketProtocol="TCP",
            ),
            SocketProtocolTestCase(
                name="udp_stack",
                protocol=None,
                protocolStack=["TLS", "HTTP", "UDP"],
                expectedSocketProtocol="UDP",
            ),
            SocketProtocolTestCase(
                name="invalid_stack",
                protocol=None,
                protocolStack=["TLS", "HTTP"],
                expectedSocketProtocol=None,
            ),
            SocketProtocolTestCase(
                name="empty_stack", protocol=None, protocolStack=[], expectedSocketProtocol=None
            ),
        ],
    )
    def test_socket_protocol(self, case: SocketProtocolTestCase):
        """ensure that we can identify the listener socket protocol based on the provided protocol and protocolStack"""

        yaml = _generateListener(case.name, case.protocol, case.protocolStack)

        compiled_ir = Compile(logger, yaml, k8s=True)
        result_ir = compiled_ir["ir"]
        listeners = list(result_ir.listeners.values())
        errors = result_ir.aconf.errors

        if case.expectedSocketProtocol == None:
            assert len(errors) == 1
            assert len(listeners) == 0
        else:
            assert len(listeners) == 1
            assert listeners[0].socket_protocol == case.expectedSocketProtocol

    @pytest.mark.compilertest
    def test_http3_valid_quic_listener(self):
        """ensure that a valid http3 listener is created using QUIC"""

        yaml = default_http3_listener_manifest()
        econf = econf_compile(yaml)

        listeners = econf["static_resources"]["listeners"]

        assert len(listeners) == 2

        # verify listener options
        listener = listeners[0]
        assert "udp_listener_config" in listener
        assert "quic_options" in listener["udp_listener_config"]
        assert listener["udp_listener_config"]["downstream_socket_config"]["prefer_gro"] == True

        # verify filter chains
        filter_chains = listener["filter_chains"]
        assert len(filter_chains) == 1
        filter_chain = filter_chains[0]

        assert filter_chain["filter_chain_match"]["transport_protocol"] == "quic"
        assert filter_chain["transport_socket"]["name"] == "envoy.transport_sockets.quic"

        # verify HCM typed_config
        typed_config = filter_chain["filters"][0]["typed_config"]
        assert typed_config["codec_type"] == "HTTP3"
        assert "http3_protocol_options" in typed_config

    @pytest.mark.compilertest
    def test_http3_missing_tls_context(self):
        """UDP listener supporting the Quic protocol requires that a the "transport_socket" be set
        in the filter_chains due to the fact that QUIC requires TLS. Envoy will reject the configuration
        if it is not found. This test ensures that the HTTP/3 Listener is dropped when a valid TLSContext is not available.
        """

        yaml = (
            """
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
            + default_http3_listener_manifest()
        )

        ## we don't use the Compile utils here because we want to make sure that a fake secret is not injected
        aconf = Config()
        fetcher = ResourceFetcher(logger, aconf)
        fetcher.parse_yaml(yaml, k8s=True)
        aconf.load_all(fetcher.sorted())
        secret_handler = EmptySecretHandler(logger, source_root=None, cache_dir=None, version="V3")
        ir = IR(aconf, secret_handler=secret_handler)
        econf = EnvoyConfig.generate(ir, cache=None).as_dict()

        # the tcp/tls is more forgiving and doesn't crash envoy which is the current behavior
        # we observe pre v3. So we just verify that the only listener is the TCP listener.
        listeners = econf["static_resources"]["listeners"]
        assert len(listeners) == 2
        tcp_listener = listeners[0]
        assert tcp_listener["address"]["socket_address"]["protocol"] == "TCP"

    @pytest.mark.compilertest
    def test_http3_companion_listeners(self):
        """ensure that when we have companion http3 (udp)/tcp listeners bound to same port that we properly set
        port reuse, and ensure the TCP listener broadcast http/3 support
        """

        yaml = (
            """
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
            + default_http3_listener_manifest()
        )

        econf = econf_compile(yaml)

        listeners = econf["static_resources"]["listeners"]

        assert len(listeners) == 3

        ## check TCP Listener
        tcp_listener = listeners[0]
        assert tcp_listener["address"]["socket_address"]["protocol"] == "TCP"

        tcp_filter_chains = tcp_listener["filter_chains"]
        assert len(tcp_filter_chains) == 2

        default_alt_svc = 'h3=":443"; ma=86400, h3-29=":443"; ma=86400'
        _ensure_alt_svc_header_injected(tcp_listener, default_alt_svc)

        ## check UDP Listener
        udp_listener = listeners[1]
        assert udp_listener["address"]["socket_address"]["protocol"] == "UDP"

        udp_filter_chains = udp_listener["filter_chains"]
        assert len(udp_filter_chains) == 1

        _verify_no_added_response_headers(udp_listener)

    @pytest.mark.compilertest
    def test_http3_non_matching_ports(self):
        """support having the http (tcp) listener to be bound to different address:port, by default
        the alt-svc will not be injected. Note, this test ensures that envoy can be configured
        this way and will not crash. However, due to developer not setting the `alt-svc` most clients
        will not be able to upgrade to HTTP/3.
        """

        yaml = (
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-8500
  namespace: default
spec:
  port: 8500
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
"""
            + default_http3_listener_manifest()
        )

        econf = econf_compile(yaml)

        listeners = econf["static_resources"]["listeners"]

        assert len(listeners) == 3

        ## check TCP Listener
        tcp_listener = listeners[0]
        assert tcp_listener["address"]["socket_address"]["protocol"] == "TCP"

        tcp_filter_chains = tcp_listener["filter_chains"]
        assert len(tcp_filter_chains) == 2

        _verify_no_added_response_headers(tcp_listener)

        ## check UDP Listener
        udp_listener = listeners[1]
        assert udp_listener["address"]["socket_address"]["protocol"] == "UDP"

        udp_filter_chains = udp_listener["filter_chains"]
        assert len(udp_filter_chains) == 1

        _verify_no_added_response_headers(udp_listener)

    @dataclass
    class FilterChainGenTestCase:
        # name should match the file name in testdata/listeners (excluding the _out.yaml and _in.yaml suffixes)
        name: str
        # description allows developer to provide any information on use-case without having to study the in/out files
        # as well as displaying it in the pytest output for easier debugging.
        description: str

    @skip_edgestack()
    @pytest.mark.compilertest
    @pytest.mark.parametrize(
        "case",
        ids=lambda case: case.name,
        argvalues=[
            FilterChainGenTestCase(
                name="host_missing_tls",
                description="When applying the quick-start of 8080 and 8443 with a Host that doesn't have tls configured it will generate basically "
                "the same FilterChain/Vhost/Routes between the two listeners. The HTTPS Listener will not generate a "
                "Filter Chain checking for matches on TLS and/or SNI. A fallback cert is not used because a Host is configured "
                "although arguable incorrectly. Ideally, the HTTPS listener (8443) should not attach anything or use the fallback "
                "cert as-if no Host was provided but since this was existing behavior it has been left that way for now.",
            ),
            FilterChainGenTestCase(
                name="no_host",
                description="If no Host is provided the http (8080) listener will create the normal 'shared http' filter chain "
                "and the https (8443) listener will generate two filter chains; a 'shared http' filter chain to catch non-tls traffic "
                "and a filter chain matching on tls and using the fall-back cert with NO sni matching.",
            ),
            FilterChainGenTestCase(
                name="prefix_wildcard_and_hostname_with_port",
                description="Properly handle Host with prefix wildcards (*.local) and hosts with portnames (*.local:8500). "
                "In this scenario both host will be coalesced into the same FilterChain due to matching SNI but "
                "will get their own virtual hosts due to needing to match on a :authority header. Mappings will "
                "associate to the most specific vhost based on the Host.hostname and Mapping.hostname fields",
            ),
        ],
    )
    def test_listener_filterchain_generation(self, case: FilterChainGenTestCase):
        """Ensure that the Listener FilterChain and correct vhosts are generated based on the
        provided Listener, Host and Mappings to ensure mutliple scenarios are covered such as
        clients sending hostname:port and addressing h2/h3 connection re-use on parent domains
        """

        def _cleanse_secrets(listeners: dict):
            """
            For these tests the full path of the secret is not what is being tested. This will
            mutate the listeners and remove secret paths and replace with static values so that when
            diffing against expected results they don't cause variance in tests.
            """

            for listIndex, listener in enumerate(listeners):
                for fcIndex, fc in enumerate(listener["filter_chains"]):
                    certs = (
                        fc.get("transport_socket", {})
                        .get("typed_config", {})
                        .get("common_tls_context", {})
                        .get("tls_certificates", {})
                    )
                    for i, cert in enumerate(certs):
                        if "filename" in cert.get("certificate_chain", {}):
                            cert["certificate_chain"]["filename"] = "test.crt"
                        if "filename" in cert.get("private_key", {}):
                            cert["private_key"]["filename"] = "test.key"
                        certs[i] = cert
                    listeners[listIndex]["filter_chains"][fcIndex] = fc

        testdata_dir = os.path.join(
            os.path.dirname(os.path.abspath(__file__)), "testdata", "listeners"
        )

        applied_yaml = open(os.path.join(testdata_dir, f"{case.name}_in.yaml"), "r").read()
        expected = yaml.safe_load(open(os.path.join(testdata_dir, f"{case.name}_out.yaml"), "r"))

        econf = econf_compile(applied_yaml)
        assert econf

        expListeners = expected.get("listeners", {})
        assert expListeners != {}

        listeners = econf.get("static_resources", {}).get("listeners", [])
        _cleanse_secrets(listeners)

        assert expListeners == listeners
