import os
from typing import Dict, Generator, Tuple, Union

import tests.integration.manifests as integration_manifests
from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import Query

LOADBALANCER_POD = """
---
apiVersion: v1
kind: Pod
metadata:
  name: {name}
  labels:
    backend: {backend}
    scope: AmbassadorTest
spec:
  containers:
  - name: backend
    image: {images[kat-server]}
    ports:
    - containerPort: 8080
    env:
    - name: BACKEND_8080
      value: {backend_env}
    - name: BACKEND_8443
      value: {backend_env}
"""


class LoadBalancerTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-0
hostname: "*"
prefix: /{self.name}-0/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-1
hostname: "*"
prefix: /{self.name}-1/
service: {self.target.path.fqdn}
resolver:  endpoint
load_balancer:
  policy: round_robin
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-2
hostname: "*"
prefix: /{self.name}-2/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: ring_hash
  header: test-header
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-3
hostname: "*"
prefix: /{self.name}-3/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: ring_hash
  source_ip: True
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-4
hostname: "*"
prefix: /{self.name}-4/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: ring_hash
  cookie:
    name: test-cookie
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-5
hostname: "*"
prefix: /{self.name}-5/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: ring_hash
  cookie:
    name: test-cookie
  header: test-header
  source_ip: True
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-6
hostname: "*"
prefix: /{self.name}-6/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: round_robin
  cookie:
    name: test-cookie
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-7
hostname: "*"
prefix: /{self.name}-7/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: rr
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-8
hostname: "*"
prefix: /{self.name}-8/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: least_request
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-9
hostname: "*"
prefix: /{self.name}-9/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: least_request
  cookie:
    name: test-cookie
"""
        )

    def queries(self):
        yield Query(self.url(self.name + "-0/"))
        yield Query(self.url(self.name + "-1/"))
        yield Query(self.url(self.name + "-2/"))
        yield Query(self.url(self.name + "-3/"))
        yield Query(self.url(self.name + "-4/"))
        yield Query(self.url(self.name + "-5/"), expected=404)
        yield Query(self.url(self.name + "-6/"), expected=404)
        yield Query(self.url(self.name + "-7/"), expected=404)
        yield Query(self.url(self.name + "-8/"))
        yield Query(self.url(self.name + "-9/"), expected=404)


class GlobalLoadBalancing(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        backend = self.name.lower() + "-backend"
        return (
            integration_manifests.format(
                LOADBALANCER_POD,
                name="{}-1".format(self.path.k8s),
                backend=backend,
                backend_env="{}-1".format(self.path.k8s),
            )
            + integration_manifests.format(
                LOADBALANCER_POD,
                name="{}-2".format(self.path.k8s),
                backend=backend,
                backend_env="{}-2".format(self.path.k8s),
            )
            + integration_manifests.format(
                LOADBALANCER_POD,
                name="{}-3".format(self.path.k8s),
                backend=backend,
                backend_env="{}-3".format(self.path.k8s),
            )
            + """
---
apiVersion: v1
kind: Service
metadata:
  labels:
    scope: AmbassadorTest
  name: globalloadbalancing-service
spec:
  ports:
  - name: http
    port: 80
    targetPort: 8080
  selector:
    backend: {backend}
""".format(
                backend=backend
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  resolver: endpoint
  load_balancer:
    policy: ring_hash
    header: LB-HEADER
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-header
hostname: "*"
prefix: /{self.name}-header/
service: globalloadbalancing-service
load_balancer:
  policy: ring_hash
  cookie:
    name: lb-cookie
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-generic
hostname: "*"
prefix: /{self.name}-generic/
service: globalloadbalancing-service
"""
        )

    def queries(self):
        # generic header queries
        for i in range(50):
            yield Query(self.url(self.name) + "-header/")

        # header queries
        for i in range(50):
            yield Query(self.url(self.name) + "-header/", headers={"LB-HEADER": "yes"})

        # cookie queries
        for i in range(50):
            yield Query(
                self.url(self.name) + "-header/", cookies=[{"name": "lb-cookie", "value": "yes"}]
            )

        # generic - generic queries
        for i in range(50):
            yield Query(self.url(self.name) + "-generic/")

        # generic - header queries
        for i in range(50):
            yield Query(self.url(self.name) + "-generic/", headers={"LB-HEADER": "yes"})

        # generic - cookie queries
        for i in range(50):
            yield Query(
                self.url(self.name) + "-generic/", cookies=[{"name": "lb-cookie", "value": "yes"}]
            )

    def check(self):
        assert len(self.results) == 300

        generic_queries = self.results[:50]
        header_queries = self.results[50:100]
        cookie_queries = self.results[100:150]

        generic_generic_queries = self.results[150:200]
        generic_header_queries = self.results[200:250]
        generic_cookie_queries = self.results[250:300]

        # generic header queries - no cookie, no header
        generic_dict: Dict[str, int] = {}
        for result in generic_queries:
            assert result.backend
            generic_dict[result.backend.name] = (
                generic_dict[result.backend.name] + 1 if result.backend.name in generic_dict else 1
            )
        assert len(generic_dict) == 3

        # header queries - no cookie - no sticky expected
        header_dict: Dict[str, int] = {}
        for result in header_queries:
            assert result.backend
            header_dict[result.backend.name] = (
                header_dict[result.backend.name] + 1 if result.backend.name in header_dict else 1
            )
        assert len(header_dict) == 3

        # cookie queries - no headers - sticky expected
        cookie_dict: Dict[str, int] = {}
        for result in cookie_queries:
            assert result.backend
            cookie_dict[result.backend.name] = (
                cookie_dict[result.backend.name] + 1 if result.backend.name in cookie_dict else 1
            )
        assert len(cookie_dict) == 1

        # generic header queries - no cookie, no header
        generic_generic_dict: Dict[str, int] = {}
        for result in generic_generic_queries:
            assert result.backend
            generic_generic_dict[result.backend.name] = (
                generic_generic_dict[result.backend.name] + 1
                if result.backend.name in generic_generic_dict
                else 1
            )
        assert len(generic_generic_dict) == 3

        # header queries - no cookie - sticky expected
        generic_header_dict: Dict[str, int] = {}
        for result in generic_header_queries:
            assert result.backend
            generic_header_dict[result.backend.name] = (
                generic_header_dict[result.backend.name] + 1
                if result.backend.name in generic_header_dict
                else 1
            )
        assert len(generic_header_dict) == 1

        # cookie queries - no headers - no sticky expected
        generic_cookie_dict: Dict[str, int] = {}
        for result in generic_cookie_queries:
            assert result.backend
            generic_cookie_dict[result.backend.name] = (
                generic_cookie_dict[result.backend.name] + 1
                if result.backend.name in generic_cookie_dict
                else 1
            )
        assert len(generic_cookie_dict) == 3


class PerMappingLoadBalancing(AmbassadorTest):
    target: ServiceType
    policy: str

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        backend = self.name.lower() + "-backend"
        return (
            integration_manifests.format(
                LOADBALANCER_POD,
                name="{}-1".format(self.path.k8s),
                backend=backend,
                backend_env="{}-1".format(self.path.k8s),
            )
            + integration_manifests.format(
                LOADBALANCER_POD,
                name="{}-2".format(self.path.k8s),
                backend=backend,
                backend_env="{}-2".format(self.path.k8s),
            )
            + integration_manifests.format(
                LOADBALANCER_POD,
                name="{}-3".format(self.path.k8s),
                backend=backend,
                backend_env="{}-3".format(self.path.k8s),
            )
            + """
---
apiVersion: v1
kind: Service
metadata:
  labels:
    scope: AmbassadorTest
  name: permappingloadbalancing-service
spec:
  ports:
  - name: http
    port: 80
    targetPort: 8080
  selector:
    backend: {backend}
""".format(
                backend=backend
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        for policy in ["ring_hash", "maglev"]:
            self.policy = policy
            yield self, self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-header-{self.policy}
hostname: "*"
prefix: /{self.name}-header-{self.policy}/
service: permappingloadbalancing-service
resolver: endpoint
load_balancer:
  policy: {self.policy}
  header: LB-HEADER
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-sourceip-{self.policy}
hostname: "*"
prefix: /{self.name}-sourceip-{self.policy}/
service: permappingloadbalancing-service
resolver: endpoint
load_balancer:
  policy: {self.policy}
  source_ip: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-cookie-{self.policy}
hostname: "*"
prefix: /{self.name}-cookie-{self.policy}/
service: permappingloadbalancing-service
resolver: endpoint
load_balancer:
  policy: {self.policy}
  cookie:
    name: lb-cookie
    ttl: 125s
    path: /foo
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-cookie-no-ttl-{self.policy}
hostname: "*"
prefix: /{self.name}-cookie-no-ttl-{self.policy}/
service: permappingloadbalancing-service
resolver: endpoint
load_balancer:
  policy: {self.policy}
  cookie:
    name: lb-cookie
"""
            )

    def queries(self):
        for policy in ["ring_hash", "maglev"]:
            # generic header queries
            for i in range(50):
                yield Query(self.url(self.name) + "-header-{}/".format(policy))

            # header queries
            for i in range(50):
                yield Query(
                    self.url(self.name) + "-header-{}/".format(policy), headers={"LB-HEADER": "yes"}
                )

            # source IP queries
            for i in range(50):
                yield Query(self.url(self.name) + "-sourceip-{}/".format(policy))

            # generic cookie queries
            for i in range(50):
                yield Query(self.url(self.name) + "-cookie-{}/".format(policy))

            # cookie queries
            for i in range(50):
                yield Query(
                    self.url(self.name) + "-cookie-{}/".format(policy),
                    cookies=[{"name": "lb-cookie", "value": "yes"}],
                )

            # cookie no TTL queries
            for i in range(50):
                yield Query(
                    self.url(self.name) + "-cookie-no-ttl-{}/".format(policy),
                    cookies=[{"name": "lb-cookie", "value": "yes"}],
                )

    def check(self):
        assert len(self.results) == 600

        for i in [0, 300]:
            generic_header_queries = self.results[0 + i : 50 + i]
            header_queries = self.results[50 + i : 100 + i]
            source_ip_queries = self.results[100 + i : 150 + i]
            generic_cookie_queries = self.results[150 + i : 200 + i]
            cookie_queries = self.results[200 + i : 250 + i]
            cookie_no_ttl_queries = self.results[250 + i : 300 + i]

            # generic header queries
            generic_header_dict: Dict[str, int] = {}
            for result in generic_header_queries:
                assert result.backend
                generic_header_dict[result.backend.name] = (
                    generic_header_dict[result.backend.name] + 1
                    if result.backend.name in generic_header_dict
                    else 1
                )
            assert len(generic_header_dict) == 3

            # header queries
            header_dict: Dict[str, int] = {}
            for result in header_queries:
                assert result.backend
                header_dict[result.backend.name] = (
                    header_dict[result.backend.name] + 1
                    if result.backend.name in header_dict
                    else 1
                )
            assert len(header_dict) == 1

            # source IP queries
            source_ip_dict: Dict[str, int] = {}
            for result in source_ip_queries:
                assert result.backend
                source_ip_dict[result.backend.name] = (
                    source_ip_dict[result.backend.name] + 1
                    if result.backend.name in source_ip_dict
                    else 1
                )
            assert len(source_ip_dict) == 1
            assert list(source_ip_dict.values())[0] == 50

            # generic cookie queries - results must include Set-Cookie header
            generic_cookie_dict: Dict[str, int] = {}
            for result in generic_cookie_queries:
                assert "Set-Cookie" in result.headers
                assert len(result.headers["Set-Cookie"]) == 1
                assert "lb-cookie=" in result.headers["Set-Cookie"][0]
                assert "Max-Age=125" in result.headers["Set-Cookie"][0]
                assert "Path=/foo" in result.headers["Set-Cookie"][0]

                assert result.backend
                generic_cookie_dict[result.backend.name] = (
                    generic_cookie_dict[result.backend.name] + 1
                    if result.backend.name in generic_cookie_dict
                    else 1
                )
            assert len(generic_cookie_dict) == 3

            # cookie queries
            cookie_dict: Dict[str, int] = {}
            for result in cookie_queries:
                assert "Set-Cookie" not in result.headers

                assert result.backend
                cookie_dict[result.backend.name] = (
                    cookie_dict[result.backend.name] + 1
                    if result.backend.name in cookie_dict
                    else 1
                )
            assert len(cookie_dict) == 1

            # cookie no TTL queries
            cookie_no_ttl_dict: Dict[str, int] = {}
            for result in cookie_no_ttl_queries:
                assert "Set-Cookie" not in result.headers

                assert result.backend
                cookie_no_ttl_dict[result.backend.name] = (
                    cookie_no_ttl_dict[result.backend.name] + 1
                    if result.backend.name in cookie_no_ttl_dict
                    else 1
                )
            assert len(cookie_no_ttl_dict) == 1
