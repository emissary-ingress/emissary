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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
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
      apiVersion: x.getambassador.io/v3alpha1
      kind: AmbassadorMapping
      name:  {self.name}-nested
      hostname: "*"
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
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

class InvalidPortMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: websocket-echo-server.plain-namespace
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: https://{self.target.path.fqdn}
"""

    EXPLICIT = """
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: foobar.com
host_redirect: true
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-2
hostname: "*"
prefix: /{self.name}-2/
case_sensitive: false
service: foobar.com
host_redirect: true
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-3
hostname: "*"
prefix: /{self.name}-3/foo/
service: foobar.com
host_redirect: true
path_redirect: /redirect/
redirect_response_code: 302
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-4
hostname: "*"
prefix: /{self.name}-4/foo/bar/baz
service: foobar.com
host_redirect: true
prefix_redirect: /foobar/baz
redirect_response_code: 307
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-5
hostname: "*"
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")
        yield self.canary, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-canary
hostname: "*"
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
host_rewrite: canary.1.example.com
""")
        yield self.canary, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-canary
hostname: "*"
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: httpbin.plain-namespace
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.parent.url("edge_stack/admin/"), expected=404)
        yield Query(self.parent.url(self.name + "/"), expected=200)

    def check(self):
        # assert self.results[0].headers['X-Content-Type-Options'] == ['nosniff']
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: httpbin.plain-namespace
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
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
hostname: "*"
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

