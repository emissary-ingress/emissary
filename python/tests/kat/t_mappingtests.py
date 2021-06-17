from kat.harness import EDGE_STACK, variants, Query

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import MappingTest, OptionTest, ServiceType
from kat.utils import namespace_manifest

from ambassador.constants import Constants

# This is the place to add new MappingTests.


def unique(options):
    added = set()
    result = []
    for o in options:
        if o.__class__ not in added:
            added.add(o.__class__)
            result.append(o)
    return tuple(result)


class SimpleMapping(MappingTest):

    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

            for mot in variants(OptionTest):
                yield cls(st, (mot,), name="{self.target.name}-{self.options[0].name}")

            yield cls(st, unique(v for v in variants(OptionTest)
                                 if not getattr(v, "isolated", False)), name="{self.target.name}-all")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))
        yield Query(self.parent.url(f'need-normalization/../{self.name}/'))

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'


class SimpleMappingIngress(MappingTest):

    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def manifests(self) -> str:
        return f"""
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: plain
  name: {self.name.lower()}
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
"""

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), xfail="IHA hostglob")
        yield Query(self.parent.url(f'need-normalization/../{self.name}/'), xfail="IHA hostglob")

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'

# Disabled SimpleMappingIngressDefaultBackend since adding a default fallback mapping would break other
# assertions, expecting to 404 if mappings don't match in Plain.
# class SimpleMappingIngressDefaultBackend(MappingTest):
#
#     parent: AmbassadorTest
#     target: ServiceType
#
#     @classmethod
#     def variants(cls):
#         for st in variants(ServiceType):
#             yield cls(st, name="{self.target.name}")
#
#     def manifests(self) -> str:
#         return f"""
# apiVersion: extensions/v1beta1
# kind: Ingress
# metadata:
#   annotations:
#     kubernetes.io/ingress.class: ambassador
#     getambassador.io/ambassador-id: plain
#   name: {self.name.lower()}
# spec:
#   backend:
#     serviceName: {self.target.path.k8s}
#     servicePort: 80
# """
#
#     def queries(self):
#         yield Query(self.parent.url(self.name))
#
#     def check(self):
#         for r in self.results:
#             if r.backend:
#                 assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
#                 assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}'


class SimpleIngressWithAnnotations(MappingTest):

    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def manifests(self) -> str:
        return f"""
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: plain
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v2
      kind:  Mapping
      name:  {self.name}-nested
      host: "*"
      prefix: /{self.name}-nested/
      service: http://{self.target.path.fqdn}
      ambassador_id: plain
  name: {self.name.lower()}
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
"""

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), xfail="IHA hostglob")
        yield Query(self.parent.url(f'need-normalization/../{self.name}/'), xfail="IHA hostglob")
        yield Query(self.parent.url(self.name + "-nested/"), xfail="IHA hostglob")
        yield Query(self.parent.url(self.name + "-non-existent/"), expected=404, xfail="IHA hostglob")

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] in (f'/{self.name}/', f'/{self.name}-nested/')


class HostHeaderMappingIngress(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def manifests(self) -> str:
        return f"""
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: plain
  name: {self.name.lower()}
spec:
  rules:
  - host: inspector.external
    http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
"""

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.internal"}, expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.external"})


class HostHeaderMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
host: inspector.external
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.internal"}, expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.external"})
        # Test that a host header with a port value that does match the listener's configured port is not
        # stripped for the purpose of routing, so it does not match the Mapping. This is the default behavior,
        # and can be overridden using `strip_matching_host_port`, tested below.
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.external:" + str(Constants.SERVICE_PORT_HTTP)}, expected=404)

# This has to be an `AmbassadorTest` because we're going to set up a Module that
# needs to apply to just this test. If this were a MappingTest, then the Module
# would apply to all other MappingTest's and we don't want that.
class HostHeaderMappingStripMatchingHostPort(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  strip_matching_host_port: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
host: myhostname.com
""")

    def queries(self):
        # Sanity test that a missing or incorrect hostname does not route, and it does route with a correct hostname.
        yield Query(self.url(self.name + "/"), expected=404)
        yield Query(self.url(self.name + "/"), headers={"Host": "yourhostname.com"}, expected=404)
        yield Query(self.url(self.name + "/"), headers={"Host": "myhostname.com"})
        # Test that a host header with a port value that does match the listener's configured port is correctly
        # stripped for the purpose of routing, and matches the mapping.
        yield Query(self.url(self.name + "/"), headers={"Host": "myhostname.com:" + str(Constants.SERVICE_PORT_HTTP)})
        # Test that a host header with a port value that does _not_ match the listener's configured does not have its
        # port value stripped for the purpose of routing, so it does not match the mapping.
        yield Query(self.url(self.name + "/"), headers={"Host": "myhostname.com:11875"}, expected=404)


# This has to be an `AmbassadorTest` because we're going to set up a Module that
# needs to apply to just this test. If this were a MappingTest, then the Module
# would apply to all other MappingTest's and we don't want that.
class MergeSlashesDisabled(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/status/
rewrite: /status/
service: httpbin.default
""")

    def queries(self):
        yield Query(self.url(self.name + "/status/200"))
        # Sanity test that an extra slash in the front of the request URL does not match the mapping,
        # since we did not set merge_slashes on the Ambassador module.
        yield Query(self.url("/" + self.name + "/status/200"), expected=404)
        yield Query(self.url("/" + self.name + "//status/200"), expected=404)
        yield Query(self.url(self.name + "//status/200"), expected=404)


# This has to be an `AmbassadorTest` because we're going to set up a Module that
# needs to apply to just this test. If this were a MappingTest, then the Module
# would apply to all other MappingTest's and we don't want that.
class MergeSlashesEnabled(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  merge_slashes: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/status/
rewrite: /status/
service: httpbin.default
""")

    def queries(self):
        yield Query(self.url(self.name + "/status/200"))
        # Since merge_slashes is on the Ambassador module, extra slashes in the URL should not prevent the request
        # from matching.
        yield Query(self.url("/" + self.name + "/status/200"))
        yield Query(self.url("/" + self.name + "//status/200"))
        yield Query(self.url(self.name + "//status/200"))

# This has to be an `AmbassadorTest` because we're going to set up a Module that
# needs to apply to just this test. If this were a MappingTest, then the Module
# would apply to all other MappingTest's and we don't want that.
class RejectRequestsWithEscapedSlashesDisabled(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/status/
rewrite: /status/
service: httpbin
""")

    def queries(self):
        # Sanity test that escaped slashes are not rejected by default. The upstream
        # httpbin server doesn't know what to do with this request, though, so expect
        # a 404. In another test, we'll expect HTTP 400 with reject_requests_with_escaped_slashes
        yield Query(self.url(self.name + "/status/%2F200"), expected=404)

    def check(self):
        # We should have observed this 404 upstream from httpbin. The presence of this header verifies that.
        print ("headers=%s", repr(self.results[0].headers))
        assert 'X-Envoy-Upstream-Service-Time' in self.results[0].headers


# This has to be an `AmbassadorTest` because we're going to set up a Module that
# needs to apply to just this test. If this were a MappingTest, then the Module
# would apply to all other MappingTest's and we don't want that.
class RejectRequestsWithEscapedSlashesEnabled(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  reject_requests_with_escaped_slashes: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/status/
rewrite: /status/
service: httpbin
""")

    def queries(self):
        # Expect that requests with escaped slashes are rejected by Envoy. We know this is rejected
        # by envoy because in a previous test, without the reject_requests_with_escaped_slashes,
        # this same request got status 404.
        yield Query(self.url(self.name + "/status/%2F200"), expected=400)

    def check(self):
        # We should have not have observed this 400 upstream from httpbin. The absence of this header
        # suggests that (though does not prove, in theory).
        assert 'X-Envoy-Upstream-Service-Time' not in self.results[0].headers


class InvalidPortMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}:80.invalid
""")

    def queries(self):
        yield Query(self.parent.url("ambassador/v0/diag/?json=true&filter=errors"))

    def check(self):
        error_string = 'found invalid port for service'
        found_error = False
        for error_list in self.results[0].json:
            for error in error_list:
                if error.find(error_string) != -1:
                    found_error = True
        assert found_error, "could not find the relevant error - {}".format(error_string)


class WebSocketMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: websocket-echo-server.default
use_websocket: true
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)

        yield Query(self.parent.url(self.name + "/", scheme="ws"), messages=["one", "two", "three"])

    def check(self):
        assert self.results[-1].messages == ["one", "two", "three"], "invalid messages: %s" % repr(self.results[-1].messages)


class TLSOrigination(MappingTest):

    parent: AmbassadorTest
    definition: str

    IMPLICIT = """
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: https://{self.target.path.fqdn}
"""

    EXPLICIT = """
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: {self.target.path.fqdn}
tls: true
"""

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for name, dfn in ("IMPLICIT", cls.IMPLICIT), ("EXPLICIT", cls.EXPLICIT):
                yield cls(v, dfn, name="{self.target.name}-%s" % name)

    def init(self, target, definition):
        MappingTest.init(self, target)
        self.definition = definition

    def config(self):
        yield self.target, self.format(self.definition)

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))

    def check(self):
        for r in self.results:
            assert r.backend.request.tls.enabled


class HostRedirectMapping(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    def init(self):
        # Skip until fixing the hostglob thing.
        self.skip_node = True

        MappingTest.init(self, HTTP())

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: foobar.com
host_redirect: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-2
host: "*"
prefix: /{self.name}-2/
case_sensitive: false
service: foobar.com
host_redirect: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-3
host: "*"
prefix: /{self.name}-3/foo/
service: foobar.com
host_redirect: true
path_redirect: /redirect/
redirect_response_code: 302
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-4
host: "*"
prefix: /{self.name}-4/foo/bar/baz
service: foobar.com
host_redirect: true
prefix_redirect: /foobar/baz
redirect_response_code: 307
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-5
host: "*"
prefix: /{self.name}-5/assets/([a-f0-9]{{12}})/images
prefix_regex: true
service: foobar.com
host_redirect: true
regex_redirect:
  pattern: /{self.name}-5/assets/([a-f0-9]{{12}})/images
  substitution: /images/\\1
redirect_response_code: 308
""")

    def queries(self):
        # [0]
        yield Query(self.parent.url(self.name + "/anything?itworked=true"), expected=301)

        # [1]
        yield Query(self.parent.url(self.name.upper() + "/anything?itworked=true"), expected=404)

        # [2]
        yield Query(self.parent.url(self.name + "-2/anything?itworked=true"), expected=301)

        # [3]
        yield Query(self.parent.url(self.name.upper() + "-2/anything?itworked=true"), expected=301)

        # [4]
        yield Query(self.parent.url(self.name + "-3/foo/anything"), expected=302)

        # [5]
        yield Query(self.parent.url(self.name + "-4/foo/bar/baz/anything"), expected=307)

        # [6]
        yield Query(self.parent.url(self.name + "-5/assets/abcd0000f123/images"), expected=308)

        # [7]
        yield Query(self.parent.url(self.name + "-5/assets/abcd0000f123/images?itworked=true"), expected=308)

    def check(self):
        # [0]
        assert self.results[0].headers['Location'] == [self.format("http://foobar.com/{self.name}/anything?itworked=true")], \
            f"Unexpected Location {self.results[0].headers['Location']}"

        # [1]
        assert self.results[1].status == 404

        # [2]
        assert self.results[2].headers['Location'] == [self.format("http://foobar.com/{self.name}-2/anything?itworked=true")], \
            f"Unexpected Location {self.results[2].headers['Location']}"

        # [3]
        assert self.results[3].headers['Location'] == [self.format("http://foobar.com/" + self.name.upper() + "-2/anything?itworked=true")], \
            f"Unexpected Location {self.results[3].headers['Location']}"

        # [4]
        assert self.results[4].headers['Location'] == [self.format("http://foobar.com/redirect/")], \
            f"Unexpected Location {self.results[4].headers['Location']}"

        # [5]
        assert self.results[5].headers['Location'] == [self.format("http://foobar.com/foobar/baz/anything")], \
            f"Unexpected Location {self.results[5].headers['Location']}"

        # [6]
        assert self.results[6].headers['Location'] == [self.format("http://foobar.com/images/abcd0000f123")], \
            f"Unexpected Location {self.results[6].headers['Location']}"

        # [7]
        assert self.results[7].headers['Location'] == [self.format("http://foobar.com/images/abcd0000f123?itworked=true")], \
            f"Unexpected Location {self.results[7].headers['Location']}"


class CanaryMapping(MappingTest):

    parent: AmbassadorTest
    target: ServiceType
    canary: ServiceType
    weight: int

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (0, 10, 50, 100):
                yield cls(v, v.clone("canary"), w, name="{self.target.name}-{self.weight}")

    def init(self, target: ServiceType, canary: ServiceType, weight):
        MappingTest.init(self, target)
        self.canary = canary
        self.weight = weight

    def config(self):
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")
        yield self.canary, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}-canary
host: "*"
prefix: /{self.name}/
service: http://{self.canary.path.fqdn}
weight: {self.weight}
""")

    def queries(self):
        for i in range(100):
            yield Query(self.parent.url(self.name + "/"))

    def check(self):
        hist = {}

        for r in self.results:
            hist[r.backend.name] = hist.get(r.backend.name, 0) + 1

        if self.weight == 0:
            assert hist.get(self.canary.path.k8s, 0) == 0
            assert hist.get(self.target.path.k8s, 0) == 100
        elif self.weight == 100:
            assert hist.get(self.canary.path.k8s, 0) == 100
            assert hist.get(self.target.path.k8s, 0) == 0
        else:
            canary = 100*hist.get(self.canary.path.k8s, 0)/len(self.results)
            main = 100*hist.get(self.target.path.k8s, 0)/len(self.results)

            assert abs(self.weight - canary) < 25, f'weight {self.weight} routed {canary}% to canary'
            assert abs(100 - (canary + main)) < 2, f'weight {self.weight} routed only {canary + main}% at all?'


class CanaryDiffMapping(MappingTest):

    parent: AmbassadorTest
    target: ServiceType
    canary: ServiceType
    weight: int

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (0, 10, 50, 100):
                yield cls(v, v.clone("canary"), w, name="{self.target.name}-{self.weight}")

    def init(self, target: ServiceType, canary: ServiceType, weight):
        MappingTest.init(self, target)
        self.canary = canary
        self.weight = weight

    def config(self):
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
host_rewrite: canary.1.example.com
""")
        yield self.canary, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}-canary
host: "*"
prefix: /{self.name}/
service: http://{self.canary.path.fqdn}
host_rewrite: canary.2.example.com
weight: {self.weight}
""")

    def queries(self):
        for i in range(100):
            yield Query(self.parent.url(self.name + "/"))

    def check(self):
        request_hosts = ['canary.1.example.com', 'canary.2.example.com']

        hist = {}

        for r in self.results:
            hist[r.backend.name] = hist.get(r.backend.name, 0) + 1
            assert r.backend.request.host in request_hosts, f'Expected host {request_hosts}, got {r.backend.request.host}'

        if self.weight == 0:
            assert hist.get(self.canary.path.k8s, 0) == 0
            assert hist.get(self.target.path.k8s, 0) == 100
        elif self.weight == 100:
            assert hist.get(self.canary.path.k8s, 0) == 100
            assert hist.get(self.target.path.k8s, 0) == 0
        else:
            canary = 100 * hist.get(self.canary.path.k8s, 0) / len(self.results)
            main = 100 * hist.get(self.target.path.k8s, 0) / len(self.results)

            assert abs(self.weight - canary) < 25, f'weight {self.weight} routed {canary}% to canary'
            assert abs(100 - (canary + main)) < 2, f'weight {self.weight} routed only {canary + main}% at all?'


class AddRespHeadersMapping(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: httpbin.default
add_response_headers:
    koo:
        append: False
        value: KooK
    zoo:
        append: True
        value: ZooZ
    test:
        value: boo
    foo: Foo
""")

    def queries(self):
        yield Query(self.parent.url(self.name)+"/response-headers?zoo=Zoo&test=Test&koo=Koot")

    def check(self):
        for r in self.results:
            if r.headers:
                # print(r.headers)
                assert r.headers['Koo'] == ['KooK']
                assert r.headers['Zoo'] == ['Zoo', 'ZooZ']
                assert r.headers['Test'] == ['Test', 'boo']
                assert r.headers['Foo'] == ['Foo']

# To make sure queries to Edge stack related paths adds X-Content-Type-Options = nosniff in the response header
# and not to any other mappings/routes
class EdgeStackMapping(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    def init(self):
        MappingTest.init(self, HTTP())

        if not EDGE_STACK:
            self.skip_node = True

    def config(self):
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.parent.url("edge_stack/admin/"), expected=200)
        yield Query(self.parent.url(self.name + "/"), expected=200)

    def check(self):
        assert self.results[0].headers['X-Content-Type-Options'] == ['nosniff']
        assert "X-Content-Type-Options" not in self.results[1].headers

class RemoveReqHeadersMapping(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: httpbin.default
remove_request_headers:
- zoo
- aoo
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/headers"), headers={
            "zoo": "ZooZ",
            "aoo": "AooA",
            "foo": "FooF"
        })

    def check(self):
        for r in self.results:
            # print(r.json)
            if 'headers' in r.json:
                assert r.json['headers']['Foo'] == 'FooF'
                assert 'Zoo' not in r.json['headers']
                assert 'Aoo' not in r.json['headers']

class AddReqHeadersMapping(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
add_request_headers:
    zoo:
        append: False
        value: Zoo
    aoo:
        append: True
        value: aoo
    boo:
        value: boo
    foo: Foo
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), headers={
            "zoo": "ZooZ",
            "aoo": "AooA",
            "boo": "BooB",
            "foo": "FooF"
        })

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.request.headers['zoo'] == ['Zoo']
                assert r.backend.request.headers['aoo'] == ['AooA','aoo']
                assert r.backend.request.headers['boo'] == ['BooB','boo']
                assert r.backend.request.headers['foo'] == ['FooF','Foo']


class LinkerdHeaderMapping(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.target_no_header = HTTP(name="noheader")
        self.target_add_linkerd_header_only = HTTP(name="addlinkerdonly")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Module
name:  ambassador
config:
  add_linkerd_headers: true
  defaults:
    httpmapping:
        add_request_headers:
            fruit:
                append: False
                value: orange
        remove_request_headers:
        - x-evil-header
---
apiVersion: getambassador.io/v2
kind: Mapping
name: {self.target_add_linkerd_header_only.path.k8s}
host: "*"
prefix: /target_add_linkerd_header_only/
service: {self.target_add_linkerd_header_only.path.fqdn}
add_request_headers: {{}}
remove_request_headers: []
---
apiVersion: getambassador.io/v2
kind: Mapping
name: {self.target_no_header.path.k8s}
host: "*"
prefix: /target_no_header/
service: {self.target_no_header.path.fqdn}
add_linkerd_headers: false
---
apiVersion: getambassador.io/v2
kind: Mapping
name: {self.target.path.k8s}
host: "*"
prefix: /target/
service: {self.target.path.fqdn}
add_request_headers:
    fruit:
        append: False
        value: banana
remove_request_headers:
- x-evilness
""")

    def queries(self):
        # [0] expect Linkerd headers set through mapping
        yield Query(self.url("target/"), headers={ "x-evil-header": "evilness", "x-evilness": "more evilness" }, expected=200)

        # [1] expect no Linkerd headers
        yield Query(self.url("target_no_header/"), headers={ "x-evil-header": "evilness", "x-evilness": "more evilness" }, expected=200)

        # [2] expect Linkerd headers only
        yield Query(self.url("target_add_linkerd_header_only/"), headers={ "x-evil-header": "evilness", "x-evilness": "more evilness" }, expected=200)

    def check(self):
        # [0]
        assert len(self.results[0].backend.request.headers['l5d-dst-override']) > 0
        assert self.results[0].backend.request.headers['l5d-dst-override'] == ["{}:80".format(self.target.path.fqdn)]
        assert len(self.results[0].backend.request.headers['fruit']) > 0
        assert self.results[0].backend.request.headers['fruit'] == [ 'banana']
        assert len(self.results[0].backend.request.headers['x-evil-header']) > 0
        assert self.results[0].backend.request.headers['x-evil-header'] == [ 'evilness' ]
        assert 'x-evilness' not in self.results[0].backend.request.headers

        # [1]
        assert 'l5d-dst-override' not in self.results[1].backend.request.headers
        assert len(self.results[1].backend.request.headers['fruit']) > 0
        assert self.results[1].backend.request.headers['fruit'] == [ 'orange']
        assert 'x-evil-header' not in self.results[1].backend.request.headers
        assert len(self.results[1].backend.request.headers['x-evilness']) > 0
        assert self.results[1].backend.request.headers['x-evilness'] == [ 'more evilness' ]

        # [2]
        assert len(self.results[2].backend.request.headers['l5d-dst-override']) > 0
        assert self.results[2].backend.request.headers['l5d-dst-override'] == ["{}:80".format(self.target_add_linkerd_header_only.path.fqdn)]
        assert len(self.results[2].backend.request.headers['x-evil-header']) > 0
        assert self.results[2].backend.request.headers['x-evil-header'] == [ 'evilness' ]
        assert len(self.results[2].backend.request.headers['x-evilness']) > 0
        assert self.results[2].backend.request.headers['x-evilness'] == [ 'more evilness' ]


class SameMappingDifferentNamespaces(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return namespace_manifest('same-mapping-1') + \
            namespace_manifest('same-mapping-2') + \
            self.format('''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.target.path.k8s}
  namespace: same-mapping-1
spec:
  ambassador_id: {self.ambassador_id}
  host: "*"
  prefix: /{self.name}-1/
  service: {self.target.path.fqdn}.default
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.target.path.k8s}
  namespace: same-mapping-2
spec:
  ambassador_id: {self.ambassador_id}
  host: "*"
  prefix: /{self.name}-2/
  service: {self.target.path.fqdn}.default
''') + super().manifests()

    def queries(self):
        yield Query(self.url(self.name + "-1/"))
        yield Query(self.url(self.name + "-2/"))


class LongClusterNameMapping(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Service
metadata:
  name: thisisaverylongservicenameoverwithsixythreecharacters123456789
spec:
  type: ExternalName
  externalName: httpbin.default.svc.cluster.local
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.target.path.k8s}
spec:
  ambassador_id: {self.ambassador_id}
  host: "*"
  prefix: /{self.name}-1/
  service: thisisaverylongservicenameoverwithsixythreecharacters123456789
''') + super().manifests()

    def queries(self):
        yield Query(self.url(self.name + "-1/"))
