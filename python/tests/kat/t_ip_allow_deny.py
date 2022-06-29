from typing import Generator, Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP, Node


################
# NOTE: The IPAllow and IPDeny tests are not entirely straightforward. In
# particular:
#
# 1. They currently use an annotation for their Ambassador modules to keep
#    them distinct between the two tests. If you don't like annotations for
#    this, you'll have to set up separate namespaces.
#
# 2. Our Listener _must_ set l7Depth == 1 in order for the tests to work:
#
#    - When we hit /target/ with XFF "99.99.0.1", Envoy receives exactly that.
#      Since l7Depth is 1, Envoy accepts that as the valid address
#      of the remote end of the connection, RBAC accepts that as matching the
#      99.99.0.0/16 CIDR block, and the request is allowed or denied as
#      appropriate. Great. But when it's accepted, the rules for XFF are that
#      Envoy must append the peer address to the XFF list before forwarding, so
#      the upstream sees XFF "99.99.0.1,$katIP". In the /target/ case, the
#      upstream is a KAT backend HTTP service -- it doesn't care about XFF, and
#      just responds OK.
#
#    - When we hit /localhost/ with XFF "99.99.0.1", though, _Ambassador is the
#      upstream_. So everything up to rewriting XFF as "99.99.0.1,$katIP" is the
#      same, but Envoy hands that upstream to... itself. Since l7Depth is still 1,
#      Envoy throws away the 99.99.0.1 part and believes that the connection is
#      coming from $katIP, which does _not_ match the 99.99.0.0/16 CIDR block --
#      but the raw peer address _is_ in fact 127.0.0.1, so _that_ matches the
#      peer: 127.0.0.1 principal.


class IPAllow(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.add_default_http_listener = False
        self.add_default_https_listener = False

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: {self.name.k8s}-listener
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8080
  protocol: HTTP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
  # Allow one trusted hop, so that KAT can fake addresses with XFF (see NOTE above).
  l7Depth: 1
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /target/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-localhost-mapping
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /localhost/
  rewrite: /target/             # See NOTE above
  service: 127.0.0.1:8080       # See NOTE above
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: [{self.ambassador_id}]
config:
  ip_allow:
    - peer:   127.0.0.1      # peer address must be localhost
    - remote: 99.99.0.0/16   # honors PROXY and XFF
"""
        )

    def queries(self):
        # 0. Straightforward: hit /target/ and /localhost/ with nothing special, get 403s.
        yield Query(self.url("target/00"), expected=403)
        yield Query(self.url("localhost/01"), expected=403)

        # 1. Hit /target/ and /localhost/ with X-Forwarded-For specifying something good, get 200s.
        yield Query(self.url("target/10"), headers={"X-Forwarded-For": "99.99.0.1"})
        yield Query(self.url("localhost/11"), headers={"X-Forwarded-For": "99.99.0.1"})

        # 2. Hit /target/ and /localhost/ with X-Forwarded-For specifying something bad, get a 403.
        yield Query(self.url("target/20"), headers={"X-Forwarded-For": "99.98.0.1"}, expected=403)
        yield Query(
            self.url("localhost/21"), headers={"X-Forwarded-For": "99.98.0.1"}, expected=403
        )

        # Done. Note that the /localhost/ endpoint is wrapping around to make a localhost call back
        # to Ambassador to check the peer: principal -- see the NOTE above.

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without X-Forwarded-For they can't work.
        yield (
            "url",
            Query(self.url("ambassador/v0/check_ready"), headers={"X-Forwarded-For": "99.99.0.1"}),
        )
        yield (
            "url",
            Query(self.url("ambassador/v0/check_alive"), headers={"X-Forwarded-For": "99.99.0.1"}),
        )


class IPDeny(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.add_default_http_listener = False
        self.add_default_https_listener = False

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: {self.name.k8s}-listener
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8080
  protocol: HTTP
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
  # Allow one trusted hop, so that KAT can fake addresses with XFF (see NOTE above).
  l7Depth: 1
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /target/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-localhost-mapping
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /localhost/
  rewrite: /target/             # See NOTE above
  service: 127.0.0.1:8080       # See NOTE above
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: [{self.ambassador_id}]
config:
  ip_deny:
    - peer:   127.0.0.1      # peer address cannot be localhost (weird, huh?)
    - remote: 99.98.0.0/16   # honors PROXY and XFF
"""
        )

    def queries(self):
        # 0. Straightforward: hit /target/ and /localhost/ with nothing special, get 403s.
        yield Query(self.url("target/00"), expected=200)
        yield Query(self.url("localhost/01"), expected=403)  # This should _never_ work.

        # 1. Hit /target/ and /localhost/ with X-Forwarded-For specifying something bad, get 403s.
        yield Query(self.url("target/10"), headers={"X-Forwarded-For": "99.98.0.1"}, expected=403)
        yield Query(
            self.url("localhost/11"), headers={"X-Forwarded-For": "99.98.0.1"}, expected=403
        )

        # 2. Hit /target/ with X-Forwarded-For specifying something not so bad, get a 200. /localhost/
        #    will _still_ get a 403 though.
        yield Query(self.url("target/20"), headers={"X-Forwarded-For": "99.99.0.1"}, expected=200)
        yield Query(
            self.url("localhost/21"), headers={"X-Forwarded-For": "99.99.0.1"}, expected=403
        )

        # Done. Note that the /localhost/ endpoint is wrapping around to make a localhost call back
        # to Ambassador to check the peer: principal -- see the NOTE above.

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without X-Forwarded-For they can't work.
        yield (
            "url",
            Query(self.url("ambassador/v0/check_ready"), headers={"X-Forwarded-For": "99.99.0.1"}),
        )
        yield (
            "url",
            Query(self.url("ambassador/v0/check_alive"), headers={"X-Forwarded-For": "99.99.0.1"}),
        )
