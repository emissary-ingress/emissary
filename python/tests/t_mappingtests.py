from kat.harness import variants, Query

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import MappingTest, OptionTest, ServiceType

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
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
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
        yield Query(self.parent.url(self.name + "/"))
        yield Query(self.parent.url(f'need-normalization/../{self.name}/'))

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'


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
      apiVersion: ambassador/v1
      kind:  Mapping
      name:  {self.name}-nested
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
        yield Query(self.parent.url(self.name + "/"))
        yield Query(self.parent.url(f'need-normalization/../{self.name}/'))
        yield Query(self.parent.url(self.name + "-nested/"))
        yield Query(self.parent.url(self.name + "-non-existent/"), expected=404)

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
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
host: inspector.external
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.internal"}, expected=404)
        yield Query(self.parent.url(self.name + "/"), headers={"Host": "inspector.external"})


class InvalidPortMapping(MappingTest):

    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
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
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: echo.websocket.org:80
host_rewrite: echo.websocket.org
use_websocket: true
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)

        yield Query(self.parent.url(self.name + "/"), expected=101, headers={
            "Connection": "Upgrade",
            "Upgrade": "websocket",
            "sec-websocket-key": "DcndnpZl13bMQDh7HOcz0g==",
            "sec-websocket-version": "13"
        })

        yield Query(self.parent.url(self.name + "/", scheme="ws"), messages=["one", "two", "three"])

    def check(self):
        assert self.results[-1].messages == ["one", "two", "three"]


class TLSOrigination(MappingTest):

    parent: AmbassadorTest
    definition: str

    IMPLICIT = """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: https://{self.target.path.fqdn}
"""

    EXPLICIT = """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
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
        MappingTest.init(self, HTTP())

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: foobar.com
host_redirect: true
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-2
prefix: /{self.name}-2/
case_sensitive: false
service: foobar.com
host_redirect: true
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/anything?itworked=true"), expected=301)
        yield Query(self.parent.url(self.name.upper() + "/anything?itworked=true"), expected=404)
        yield Query(self.parent.url(self.name + "-2/anything?itworked=true"), expected=301)
        yield Query(self.parent.url(self.name.upper() + "-2/anything?itworked=true"), expected=301)

    def check(self):
        assert self.results[0].headers['Location'] == [
            self.format("http://foobar.com/{self.name}/anything?itworked=true")
        ]
        assert self.results[1].status == 404
        assert self.results[2].headers['Location'] == [
            self.format("http://foobar.com/{self.name}-2/anything?itworked=true")
        ]
        assert self.results[3].headers['Location'] == [
            self.format("http://foobar.com/" + self.name.upper() + "-2/anything?itworked=true")
        ]


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
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")
        yield self.canary, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-canary
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
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
host_rewrite: canary.1.example.com
""")
        yield self.canary, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-canary
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
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://httpbin.org
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
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://httpbin.org
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
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
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
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  add_linkerd_headers: true
---
apiVersion: ambassador/v1
kind: Mapping
name: {self.target_add_linkerd_header_only.path.k8s}
prefix: /target_add_linkerd_header_only/
service: {self.target_add_linkerd_header_only.path.fqdn}
---
apiVersion: ambassador/v1
kind: Mapping
name: {self.target_no_header.path.k8s}
prefix: /target_no_header/
service: {self.target_no_header.path.fqdn}
add_linkerd_headers: false
add_request_headers:
    fruit:
        append: False
        value: orange
---
apiVersion: ambassador/v1
kind: Mapping
name: {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
add_request_headers:
    fruit:
        append: False
        value: banana
""")

    def queries(self):
        # [0] expect Linkerd headers set through mapping
        yield Query(self.url("target/"), expected=200)

        # [1] expect no Linkerd headers
        yield Query(self.url("target_no_header/"), expected=200)

        # [2] expect Linkerd headers only
        yield Query(self.url("target_add_linkerd_header_only/"), expected=200)
        
    def check(self):
        # [0]
        assert len(self.results[0].backend.request.headers['l5d-dst-override']) > 0
        assert self.results[0].backend.request.headers['l5d-dst-override'] == ["{}:80".format(self.target.path.fqdn)]
        assert len(self.results[0].backend.request.headers['fruit']) > 0
        assert self.results[0].backend.request.headers['fruit'] == [ 'banana']

        # [1]
        assert 'l5d-dst-override' not in self.results[1].backend.request.headers
        assert len(self.results[1].backend.request.headers['fruit']) > 0
        assert self.results[1].backend.request.headers['fruit'] == [ 'orange']

        # [2]
        assert len(self.results[2].backend.request.headers['l5d-dst-override']) > 0
        assert self.results[2].backend.request.headers['l5d-dst-override'] == ["{}:80".format(self.target_add_linkerd_header_only.path.fqdn)]


class SameMappingDifferentNamespaces(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + self.format('''
---
apiVersion: v1
kind: Namespace
metadata:
  name: same-mapping-1
---
apiVersion: v1
kind: Namespace
metadata:
  name: same-mapping-2
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.target.path.k8s}
  namespace: same-mapping-1
spec:
  ambassador_id: {self.ambassador_id}
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
  prefix: /{self.name}-2/
  service: {self.target.path.fqdn}.default
''')

    def queries(self):
        yield Query(self.url(self.name + "-1/"))
        yield Query(self.url(self.name + "-2/"))
