import os
import sys

import json
import pytest
import subprocess
import time

from kat.harness import Query, is_ingress_class_compatible
from abstract_tests import AmbassadorTest, HTTP, ServiceType
from kat.utils import namespace_manifest
from tests.utils import KUBESTATUS_PATH
from ambassador.utils import parse_bool

class IngressStatusTest1(AmbassadorTest):
    status_update = {
        "loadBalancer": {
            "ingress": [{
                "ip": "42.42.42.42"
            }]
        }
    }

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
""" + super().manifests()

    def queries(self):
        if True or sys.platform != 'darwin':
            text = json.dumps(self.status_update)

            update_cmd = [KUBESTATUS_PATH, 'Service', '-n', 'default', '-f', f'metadata.name={self.name.k8s}', '-u', '/dev/fd/0']
            subprocess.run(update_cmd, input=text.encode('utf-8'), timeout=10)
            # If you run these tests individually, the time between running kubestatus
            # and the ingress resource actually getting updated is longer than the
            # time spent waiting for resources to be ready, so this test will fail (most of the time)
            time.sleep(1)

            yield Query(self.url(self.name + "/"))
            yield Query(self.url(f'need-normalization/../{self.name}/'))

    def check(self):
        if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_INGRESS_TEST", "false")):
            pytest.xfail('AMBASSADOR_PYTEST_INGRESS_TEST not set, xfailing...')

        if False and sys.platform == 'darwin':
            pytest.xfail('not supported on Darwin')

        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'

        # check for Ingress IP here
        ingress_cmd = ["kubectl", "get", "-n", "default", "-o", "json", "ingress", self.path.k8s]
        ingress_run = subprocess.Popen(ingress_cmd, stdout=subprocess.PIPE)
        ingress_out, _ = ingress_run.communicate()
        ingress_json = json.loads(ingress_out)
        assert ingress_json['status'] == self.status_update, f"Expected Ingress status to be {self.status_update}, got {ingress_json['status']} instead"


class IngressStatusTest2(AmbassadorTest):
    status_update = {
        "loadBalancer": {
            "ingress": [{
                "ip": "84.84.84.84"
            }]
        }
    }

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
""" + super().manifests()

    def queries(self):
        if True or sys.platform != 'darwin':
            text = json.dumps(self.status_update)

            update_cmd = [KUBESTATUS_PATH, 'Service', '-n', 'default', '-f', f'metadata.name={self.name.k8s}', '-u', '/dev/fd/0']
            subprocess.run(update_cmd, input=text.encode('utf-8'), timeout=10)
            # If you run these tests individually, the time between running kubestatus
            # and the ingress resource actually getting updated is longer than the
            # time spent waiting for resources to be ready, so this test will fail (most of the time)
            time.sleep(1)

            yield Query(self.url(self.name + "/"))
            yield Query(self.url(f'need-normalization/../{self.name}/'))

    def check(self):
        if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_INGRESS_TEST", "false")):
            pytest.xfail('AMBASSADOR_PYTEST_INGRESS_TEST not set, xfailing...')

        if False and sys.platform == 'darwin':
            pytest.xfail('not supported on Darwin')

        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'

        # check for Ingress IP here
        ingress_cmd = ["kubectl", "get", "-n", "default", "-o", "json", "ingress", self.path.k8s]
        ingress_run = subprocess.Popen(ingress_cmd, stdout=subprocess.PIPE)
        ingress_out, _ = ingress_run.communicate()
        ingress_json = json.loads(ingress_out)
        assert ingress_json['status'] == self.status_update, f"Expected Ingress status to be {self.status_update}, got {ingress_json['status']} instead"


class IngressStatusTestAcrossNamespaces(AmbassadorTest):
    status_update = {
        "loadBalancer": {
            "ingress": [{
                "ip": "168.168.168.168"
            }]
        }
    }

    def init(self):
        self.target = HTTP(namespace="alt-namespace")

    def manifests(self) -> str:
        return namespace_manifest("alt-namespace") + """
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
  namespace: alt-namespace
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
""" + super().manifests()

    def queries(self):
        if True or sys.platform != 'darwin':
            text = json.dumps(self.status_update)

            update_cmd = [KUBESTATUS_PATH, 'Service', '-n', 'default', '-f', f'metadata.name={self.name.k8s}', '-u', '/dev/fd/0']
            subprocess.run(update_cmd, input=text.encode('utf-8'), timeout=10)
            # If you run these tests individually, the time between running kubestatus
            # and the ingress resource actually getting updated is longer than the
            # time spent waiting for resources to be ready, so this test will fail (most of the time)
            time.sleep(1)

            yield Query(self.url(self.name + "/"))
            yield Query(self.url(f'need-normalization/../{self.name}/'))

    def check(self):
        if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_INGRESS_TEST", "false")):
            pytest.xfail('AMBASSADOR_PYTEST_INGRESS_TEST not set, xfailing...')

        if False and sys.platform == 'darwin':
            pytest.xfail('not supported on Darwin')

        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'

        # check for Ingress IP here
        ingress_cmd = ["kubectl", "get", "-o", "json", "ingress", self.path.k8s, "-n", "alt-namespace"]
        ingress_run = subprocess.Popen(ingress_cmd, stdout=subprocess.PIPE)
        ingress_out, _ = ingress_run.communicate()
        ingress_json = json.loads(ingress_out)
        assert ingress_json['status'] == self.status_update, f"Expected Ingress status to be {self.status_update}, got {ingress_json['status']} instead"


class IngressStatusTestWithAnnotations(AmbassadorTest):
    status_update = {
        "loadBalancer": {
            "ingress": [{
                "ip": "200.200.200.200"
            }]
        }
    }

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v3alpha1
      kind: Mapping
      name:  {self.name}-nested
      hostname: "*"
      prefix: /{self.name}-nested/
      service: http://{self.target.path.fqdn}
      ambassador_id: {self.ambassador_id}
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
""" + super().manifests()

    def queries(self):
        text = json.dumps(self.status_update)

        update_cmd = [KUBESTATUS_PATH, 'Service', '-n', 'default', '-f', f'metadata.name={self.name.k8s}', '-u', '/dev/fd/0']
        subprocess.run(update_cmd, input=text.encode('utf-8'), timeout=10)
        # If you run these tests individually, the time between running kubestatus
        # and the ingress resource actually getting updated is longer than the
        # time spent waiting for resources to be ready, so this test will fail (most of the time)
        time.sleep(1)

        yield Query(self.url(self.name + "/"))
        yield Query(self.url(self.name + "-nested/"))
        yield Query(self.url(f'need-normalization/../{self.name}/'))

    def check(self):
        if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_INGRESS_TEST", "false")):
            pytest.xfail('AMBASSADOR_PYTEST_INGRESS_TEST not set, xfailing...')

        # check for Ingress IP here
        ingress_cmd = ["kubectl", "get", "-n", "default", "-o", "json", "ingress", self.path.k8s]
        ingress_run = subprocess.Popen(ingress_cmd, stdout=subprocess.PIPE)
        ingress_out, _ = ingress_run.communicate()
        ingress_json = json.loads(ingress_out)
        assert ingress_json['status'] == self.status_update, f"Expected Ingress status to be {self.status_update}, got {ingress_json['status']} instead"


class SameIngressMultipleNamespaces(AmbassadorTest):
    status_update = {
        "loadBalancer": {
            "ingress": [{
                "ip": "210.210.210.210"
            }]
        }
    }

    def init(self):
        self.target = HTTP()
        self.target1 = HTTP(name="target1", namespace="same-ingress-1")
        self.target2 = HTTP(name="target2", namespace="same-ingress-2")

    def manifests(self) -> str:
        return namespace_manifest("same-ingress-1") + """
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
  namespace: same-ingress-1
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}-target1
          servicePort: 80
        path: /{self.name}-target1/
""" + namespace_manifest("same-ingress-2") + """
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
  namespace: same-ingress-2
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}-target2
          servicePort: 80
        path: /{self.name}-target2/
""" + super().manifests()

    def queries(self):
        if True or sys.platform != 'darwin':
            text = json.dumps(self.status_update)

            update_cmd = [KUBESTATUS_PATH, 'Service', '-n', 'default', '-f', f'metadata.name={self.name.k8s}', '-u', '/dev/fd/0']
            subprocess.run(update_cmd, input=text.encode('utf-8'), timeout=10)
            # If you run these tests individually, the time between running kubestatus
            # and the ingress resource actually getting updated is longer than the
            # time spent waiting for resources to be ready, so this test will fail (most of the time)
            time.sleep(1)

            yield Query(self.url(self.name + "-target1/"))
            yield Query(self.url(self.name + "-target2/"))

    def check(self):
        if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_INGRESS_TEST", "false")):
            pytest.xfail('AMBASSADOR_PYTEST_INGRESS_TEST not set, xfailing...')

        if False and sys.platform == 'darwin':
            pytest.xfail('not supported on Darwin')

        for namespace in ['same-ingress-1', 'same-ingress-2']:
            # check for Ingress IP here
            ingress_cmd = ["kubectl", "get", "-n", "default", "-o", "json", "ingress", self.path.k8s, "-n", namespace]
            ingress_run = subprocess.Popen(ingress_cmd, stdout=subprocess.PIPE)
            ingress_out, _ = ingress_run.communicate()
            ingress_json = json.loads(ingress_out)
            assert ingress_json['status'] == self.status_update, f"Expected Ingress status to be {self.status_update}, got {ingress_json['status']} instead"


class IngressStatusTestWithIngressClass(AmbassadorTest):
    status_update = {
        "loadBalancer": {
            "ingress": [{
                "ip": "42.42.42.42"
            }]
        }
    }

    def init(self):
        self.target = HTTP()

        if not is_ingress_class_compatible():
            self.xfail = 'IngressClass is not supported in this cluster'

    def manifests(self) -> str:
        return """
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: {self.name.k8s}-ext
rules:
- apiGroups: ["networking.k8s.io"]
  resources: ["ingressclasses"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: {self.name.k8s}-ext
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {self.name.k8s}-ext
subjects:
- kind: ServiceAccount
  name: {self.path.k8s}
  namespace: {self.namespace}
---
apiVersion: networking.k8s.io/v1beta1
kind: IngressClass
metadata:
  annotations:
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
spec:
  controller: getambassador.io/ingress-controller
---
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  annotations:
    getambassador.io/ambassador-id: {self.ambassador_id}
  name: {self.name.k8s}
spec:
  ingressClassName: {self.name.k8s}
  rules:
  - http:
      paths:
      - backend:
          serviceName: {self.target.path.k8s}
          servicePort: 80
        path: /{self.name}/
""" + super().manifests()

    def queries(self):
        if True or sys.platform != 'darwin':
            text = json.dumps(self.status_update)

            update_cmd = [KUBESTATUS_PATH, 'Service', '-n', 'default', '-f', f'metadata.name={self.name.k8s}', '-u', '/dev/fd/0']
            subprocess.run(update_cmd, input=text.encode('utf-8'), timeout=10)
            # If you run these tests individually, the time between running kubestatus
            # and the ingress resource actually getting updated is longer than the
            # time spent waiting for resources to be ready, so this test will fail (most of the time)
            time.sleep(1)

            yield Query(self.url(self.name + "/"))
            yield Query(self.url(f'need-normalization/../{self.name}/'))

    def check(self):
        if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_INGRESS_TEST", "false")):
            pytest.xfail('AMBASSADOR_PYTEST_INGRESS_TEST not set, xfailing...')

        if False and sys.platform == 'darwin':
            pytest.xfail('not supported on Darwin')

        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'

        # check for Ingress IP here
        ingress_cmd = ["kubectl", "get", "-n", "default", "-o", "json", "ingress", self.path.k8s]
        ingress_run = subprocess.Popen(ingress_cmd, stdout=subprocess.PIPE)
        ingress_out, _ = ingress_run.communicate()
        ingress_json = json.loads(ingress_out)
        assert ingress_json['status'] == self.status_update, f"Expected Ingress status to be {self.status_update}, got {ingress_json['status']} instead"
