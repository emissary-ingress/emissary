from typing import Tuple, Union

import yaml

from kat.harness import Query, EDGE_STACK

from abstract_tests import AmbassadorTest, assert_default_errors, HTTP, Node, ServiceType
from kat.utils import namespace_manifest


class Empty(AmbassadorTest):
    single_namespace = True
    namespace = "empty-namespace"
    extra_ports = [8877]

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now"

    @classmethod
    def variants(cls):
        yield cls()

    def manifests(self) -> str:
        return namespace_manifest("empty-namespace") + super().manifests()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield from ()

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)
        yield Query(self.url("_internal/v0/ping", scheme="http", port=8877), expected=403)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []

        # We shouldn't have any missing-CRD-types errors any more.
        for source, error in errors:
          if (('could not find' in error) and ('CRD definitions' in error)):
            assert False, f"Missing CRDs: {error}"

          if 'Ingress resources' in error:
            assert False, f"Ingress resource error: {error}"

        # The default errors assume that we have missing CRDs, and that's not correct any more,
        # so don't try to use assert_default_errors here.

class AmbassadorIDTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: getambassador.io/v2
kind:  Module
name:  ambassador
config: 
  use_ambassador_namespace_for_service_resolution: true
"""
        for prefix, amb_id in (("findme", "[{self.ambassador_id}]"),
                               ("findme-array", "[{self.ambassador_id}, missme]"),
                               ("findme-array2", "[missme, {self.ambassador_id}]"),
                               ("missme", "[missme]"),
                               ("missme-array", "[missme1, missme2]")):
            yield self.target, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.path.k8s}-{prefix}
host: "*"
prefix: /{prefix}/
service: {self.target.path.fqdn}
ambassador_id: {amb_id}
            """, prefix=self.format(prefix), amb_id=self.format(amb_id))

    def queries(self):
        yield Query(self.url("findme/"))
        yield Query(self.url("findme-array/"))
        yield Query(self.url("findme-array2/"))
        yield Query(self.url("missme/"), expected=404)
        yield Query(self.url("missme-array/"), expected=404)


class InvalidResources(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.resource_names = []

        self.models = [ """
apiVersion: getambassador.io/v2
kind:  AuthService
metadata:
  name:  {self.path.k8s}-as-bad1-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  service_bad: {self.target.path.fqdn}
""","""
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name:  {self.path.k8s}-m-good-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  host: "*"
  prefix: /good-<<WHICH>>/
  service: {self.target.path.fqdn}
""", """
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name:  {self.path.k8s}-m-bad-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  host: "*"
  prefix_bad: /bad-<<WHICH>>/
  service: {self.target.path.fqdn}
""", """
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  {self.path.k8s}-md-bad-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  config_bad: []
""", """
apiVersion: getambassador.io/v2
kind:  RateLimitService
metadata:
  name:  {self.path.k8s}-r-bad-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  service_bad: {self.target.path.fqdn}
""", """
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorTCPMapping
metadata:
  name:  {self.path.k8s}-tm-bad1-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  service: {self.target.path.fqdn}
  port_bad: 8888
""", """
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorTCPMapping
metadata:
  name:  {self.path.k8s}-tm-bad2-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  service_bad: {self.target.path.fqdn}
  port: 8888
""", """
apiVersion: getambassador.io/v2
kind:  TracingService
metadata:
  name:  {self.path.k8s}-ts-bad1-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  driver_bad: zipkin
  service: {self.target.path.fqdn}
""", """
apiVersion: getambassador.io/v2
kind:  TracingService
metadata:
  name:  {self.path.k8s}-ts-bad2-<<WHICH>>
spec:
  ambassador_id: ["{self.ambassador_id}"]
  driver: zipkin
  service_bad: {self.target.path.fqdn}
"""
        ]


    def config(self):
        counter = 0

        for m_yaml in self.models:
            counter += 1
            m = yaml.safe_load(self.format(m_yaml.replace('<<WHICH>>', 'annotation')))

            for k in m["metadata"].keys():
                m[k] = m["metadata"][k]
            del(m["metadata"])

            for k in m["spec"].keys():
                if k == "ambassador_id":
                    continue

                m[k] = m["spec"][k]
            del(m["spec"])

            if 'good' not in m["name"]:
                # These all show up as "invalidresources.default.N" because they're
                # annotations.
                self.resource_names.append(f"invalidresources.default.{counter}")

            yield self, yaml.dump(m)

    def manifests(self):
        manifests = []

        for m in self.models:
            m_yaml = self.format(m.replace("<<WHICH>>", "crd"))

            manifests.append("---")
            manifests.append(m_yaml)

            m_obj = yaml.safe_load(m_yaml)

            if 'good' not in m_obj["metadata"]["name"]:
                self.resource_names.append(m_obj["metadata"]["name"] + ".default.1")

        return super().manifests() + "\n".join(manifests)

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"))

        yield Query(self.url("good-annotation/"), expected=200)
        yield Query(self.url("bad-annotation/"), expected=404)
        yield Query(self.url("good-crd/"), expected=200)
        yield Query(self.url("bad-crd/"), expected=404)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []

        assert errors, "Invalid resources must generate errors, but we didn't get any"

        error_dict = {}

        for resource, error in errors:
            error_dict[resource] = error.split("\n", 1)[0]
        
        for name in self.resource_names:
            assert name in error_dict, f"no error found for {name}"

            error = error_dict[name]

            # This is a little weird. The way fast-reconfigure works with the Golang
            # stuff, the empty config we pass in our bad Module turns into None. Python
            # validation still catches it, but the error message is different.

            if error != "not a valid Module: None is not of type 'object'":
                assert 'required' in error, f"error for {name} should talk about required properties: {error}"


class ServerNameTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Module
name:  ambassador
config:
  server_name: "test-server"
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.path.k8s}/server-name
host: "*"
prefix: /server-name
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("server-name/"), expected=301)

    def check(self):
        assert self.results[0].headers["Server"] == [ "test-server" ]


class SafeRegexMapping(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
prefix: /{self.name}/
prefix_regex: true
host: "[a-zA-Z].*"
host_regex: true
regex_headers:
  X-Foo: "^[a-z].*"
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url(self.name + "/"), headers={"X-Foo": "hello"})
        yield Query(self.url(f'need-normalization/../{self.name}/'), headers={"X-Foo": "hello"})
        yield Query(self.url(self.name + "/"), expected=404)
        yield Query(self.url(f'need-normalization/../{self.name}/'), expected=404)

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'


class UnsafeRegexMapping(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}
prefix: /{self.name}/
prefix_regex: true
host: "[a-zA-Z].*"
host_regex: true
regex_headers:
  X-Foo: "^[a-z].*"
service: http://{self.target.path.fqdn}
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  regex_type: unsafe
""")

    def queries(self):
        yield Query(self.url(self.name + "/"), headers={"X-Foo": "hello"})
        yield Query(self.url(f'need-normalization/../{self.name}/'), headers={"X-Foo": "hello"})
        yield Query(self.url(self.name + "/"), expected=404)
        yield Query(self.url(f'need-normalization/../{self.name}/'), expected=404)

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'
