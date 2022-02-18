from typing import Generator, Tuple, Union

from kat.harness import EDGE_STACK, variants, Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP, Node
from tests.integration.manifests import namespace_manifest

from ambassador.constants import Constants

# This is the place to add new MappingTests that run in the default namespace


# This has to be an `AmbassadorTest` because we're going to set up a Module that
# needs to apply to just this test. If this were a MappingTest, then the Module
# would apply to all other MappingTest's and we don't want that.
class HostHeaderMappingStripMatchingHostPort(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  strip_matching_host_port: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  merge_slashes: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/status/
rewrite: /status/
service: httpbin.default
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  reject_requests_with_escaped_slashes: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
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

class LinkerdHeaderMapping(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.target_no_header = HTTP(name="noheader")
        self.target_add_linkerd_header_only = HTTP(name="addlinkerdonly")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: {self.target_add_linkerd_header_only.path.k8s}
hostname: "*"
prefix: /target_add_linkerd_header_only/
service: {self.target_add_linkerd_header_only.path.fqdn}
add_request_headers: {{}}
remove_request_headers: []
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: {self.target_no_header.path.k8s}
hostname: "*"
prefix: /target_no_header/
service: {self.target_no_header.path.fqdn}
add_linkerd_headers: false
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: {self.target.path.k8s}
hostname: "*"
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}
  namespace: same-mapping-1
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /{self.name}-1/
  service: {self.target.path.fqdn}.default
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}
  namespace: same-mapping-2
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.target.path.k8s}
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /{self.name}-1/
  service: thisisaverylongservicenameoverwithsixythreecharacters123456789
''') + super().manifests()

    def queries(self):
        yield Query(self.url(self.name + "-1/"))
