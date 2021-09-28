from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP


class ClusterTagTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target_1 = HTTP(name="target1")
        self.target_2 = HTTP(name="target2")

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: cluster-tag-1
spec:
  ambassador_id: {self.ambassador_id}
  hostname: "*"
  prefix: /mapping-1/
  service: {self.target_1.path.fqdn}
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: cluster-tag-2
spec:
  ambassador_id: {self.ambassador_id}
  hostname: "*"
  prefix: /mapping-2/
  service: {self.target_1.path.fqdn}
  cluster_tag: tag-1
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: cluster-tag-3
spec:
  ambassador_id: {self.ambassador_id}
  hostname: "*"
  prefix: /mapping-3/
  service: {self.target_1.path.fqdn}
  cluster_tag: tag-2
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: cluster-tag-4
spec:
  ambassador_id: {self.ambassador_id}
  hostname: "*"
  prefix: /mapping-4/
  service: {self.target_2.path.fqdn}
  cluster_tag: tag-2
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: cluster-tag-5
spec:
  ambassador_id: {self.ambassador_id}
  hostname: "*"
  prefix: /mapping-5/
  service: {self.target_1.path.fqdn}
  cluster_tag: some-really-long-tag-that-is-really-long
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: cluster-tag-6
spec:
  ambassador_id: {self.ambassador_id}
  hostname: "*"
  prefix: /mapping-6/
  service: {self.target_2.path.fqdn}
  cluster_tag: some-really-long-tag-that-is-really-long
''') + super().manifests()

    def assert_cluster(self, cluster, target_ip):
        assert cluster is not None
        assert cluster["targets"][0]["ip"] == target_ip

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true"))

    def check(self):
        result = self.results[0]
        clusters = result.json["cluster_info"]

        cluster_1 = clusters["cluster_clustertagtest_http_target1_default"]
        self.assert_cluster(cluster_1, "clustertagtest-http-target1")

        cluster_2 = clusters["cluster_tag_1_clustertagtest_http_target1_default"]
        self.assert_cluster(cluster_2, "clustertagtest-http-target1")

        cluster_3 = clusters["cluster_tag_2_clustertagtest_http_target1_default"]
        self.assert_cluster(cluster_3, "clustertagtest-http-target1")

        cluster_4 = clusters["cluster_tag_2_clustertagtest_http_target2_default"]
        self.assert_cluster(cluster_4, "clustertagtest-http-target2")

        cluster_5 = clusters["cluster_some_really_long_tag_that_is_really_long_clustertagtest_http_target1_default"]
        self.assert_cluster(cluster_5, "clustertagtest-http-target1")

        cluster_6 = clusters["cluster_some_really_long_tag_that_is_really_long_clustertagtest_http_target2_default"]
        self.assert_cluster(cluster_6, "clustertagtest-http-target2")
