from typing import List, Optional, Tuple

import logging
from retry import retry
import sys
import time

import pexpect
import pytest
import yaml

from kat.harness import load_manifest
from kat.utils import namespace_manifest

from utils import run_and_assert, apply_kube_artifacts, delete_kube_artifacts, install_ambassador
from utils import qotm_manifests, create_qotm_mapping, create_namespace, get_code_with_retry

# So this is a horrible test of endpoints not being fed to the Python code when
# endpoint routing isn't active. This should really just be a unit test of some
# Golang code, but the entrypoint watcher, the kates accumulator, and the kates
# client code are too incestuously intertwined for that right now.
#
# Instead (gah) we'll use an Actual Ambassador Cluster for this.
#
# XXX NOTE WELL NOTE WELL NOTE WELL XXX
# This test is currently limited by just looking at the most recent snapshot,
# and by not vetting that the deltas seen correspond exactly to the Endpoints 
# seen and expected. Future improvements there, for sure.
#
# The Right Fix is to tweak the snapshot logic in diagd so that there's a way
# to tell it to keep _all_ the snapshots, then pull all of them in the test and
# merge all the endpoints and deltas. That's... a bit more work.

########
# THIS horrible bit uses port forwarding to wait for our Ambassador to be running.
# Oh the joy.

child = None    # see start_port_forward

def start_port_forward(logfile, namespace, target_name, target_port=80) -> int:
    # Use a global to hold the child process so that it won't get killed
    # when we go out of scope.
    global child

    logfile.write(f"START PORT FORWARD to {target_name}:{target_port} in namespace {namespace}\n")

    cmd = f"kubectl port-forward --namespace {namespace} {target_name} :{target_port}"
    logfile.write(f"Running: {cmd}\n")

    child = pexpect.spawn(cmd, encoding="utf-8")
    child.logfile = logfile

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, r"^Forwarding from 127\.0\.0\.1:(\d+) -> (\d+)" ])

    if i == 0:
        logfile.write("EXIT: port-forward died?\n")
        return -1

    if i == 1:
        logfile.write("EXIT: port-forward timeout out?\n")
        return -1

    local_port = int(child.match.group(1))
    dest_port = int(child.match.group(2))

    return int(local_port)


class SnapshotInfo:
    def __init__(self, step, namespace="default", ambassador_pod="ambassador"):
        # Start by grabbing the latest snapshot.
        # 
        # XXX Potential flake here: suppose our changes are split across two
        # snapshots? We're basically hoping that that won't happen. It's a risk,
        # but manageable by carefully picking what we change. Probably.
        # 
        # Also, the time this kubectl cp takes actually works in our favor here.

        cmd = [ "kubectl", "cp", "--namespace", namespace, 
                    f"{ambassador_pod}:/tmp/ambassador/snapshots/snapshot.yaml", 
                    "/tmp/snapshot.yaml" ]
        
        run_and_assert(cmd)

        diag = list(yaml.safe_load_all(open("/tmp/snapshot.yaml")))
        assert diag != None, f"{step}: need non-empty diagnostics"

        d0 = diag[0]
        assert d0 != None, f"{step}: need non-empty first diagnostics element"

        k8s = d0.get("Kubernetes", {})
        assert k8s != None, f"{step}: need non-empty Kubernetes diagnostics"

        self.endpoints = k8s.get("Endpoints", [])

        self.deltas = d0.get("Deltas", [])
        self.filter_deltas(lambda delta: delta["kind"] == "Endpoints")

    def filter_deltas(self, filter_function):
        self.deltas = list(filter(filter_function, self.deltas))

    @property
    def some_endpoints(self):
        return (len(self.endpoints) != 0)
    
    @property
    def no_endpoints(self):
        return (len(self.endpoints) == 0)

    @property
    def some_deltas(self):
        return (len(self.deltas) != 0)
    
    @property
    def no_deltas(self):
        return (len(self.deltas) == 0)


class TestEndpointFiltering:
    CustomResolver = """
apiVersion: getambassador.io/v2
kind: KubernetesEndpointResolver
metadata:
  name: custom-resolver
spec: {}
"""    

    AmbassadorModuleEndpoints = """
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    resolver: endpoint
"""    

    AmbassadorModuleService = """
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    resolver: kubernetes-service
"""    

    CustomEndpoints = """
# All the IP addresses, pod names, etc., are basically made up. These
# aren't meant to be functional, just to exercise the machinery of 
# filting things in the watcher.
apiVersion: v1
kind: Endpoints
metadata:
  name: ffs1
  namespace: {self.namespace}
subsets:
- addresses:
  - ip: 10.32.47.185
    nodeName: ip-10-32-45-82.example.com
    targetRef:
      kind: Pod
      name: ffs1-instance-185
      namespace: {self.namespace}
  - ip: 10.32.60.196
    nodeName: ip-10-32-56-56.example.com
    targetRef:
      kind: Pod
      name: ffs1-instance-196
      namespace: {self.namespace}
  ports:
  - name: https
    port: 8443
    protocol: TCP
  - name: http
    port: 8080
    protocol: TCP
---
apiVersion: v1
kind: Endpoints
metadata:
  name: ffs2
  namespace: {self.namespace}
subsets:
- addresses:
  - ip: 10.32.47.185
    nodeName: ip-10-32-45-82.example.com
    targetRef:
      kind: Pod
      name: ffs2-instance-185
      namespace: {self.namespace}
  - ip: 10.32.60.196
    nodeName: ip-10-32-56-56.example.com
    targetRef:
      kind: Pod
      name: ffs2-instance-196
      namespace: {self.namespace}
  ports:
  - name: ambassador-admin
    port: 8877
    protocol: TCP
"""

    def test_endpoint_filtering(self):
        global child

        # Everything gets dumped into this single namespace.
        self.namespace = "endpoint-filtering-ns"

        # --------
        self.STEP("setup namespace")

        # Make sure that we have an empty namespace to work with.
        self.guarantee_empty_namespace()

        # --------
        self.STEP("setup Ambassador")

        # OK. Start by installing Ambassador. Single namespace, nothing special about
        # endpoint routing, do _not_ set legacy mode.
        install_ambassador(namespace=self.namespace)

        # Install QotM with a mapping.
        self.apply_kube_artifacts(qotm_manifests)
        create_qotm_mapping(namespace=self.namespace)

        # Make sure we can talk to QotM (using a port-forward, ew).
        #
        # XXX This is _such_ a crock. But it's necessary, for all the same reasons that
        # we use the KAT requirements to make sure things are running before we try our
        # test.
        port_forward_port = -1
        tries = 5

        while tries > 0:
            port_forward_port = start_port_forward(sys.stdout, self.namespace, "service/ambassador")
    
            if port_forward_port > 0:
                break

            tries -= 1
            print(f"port-forward failed, tries left: {tries}")

            child.terminate()
            child = None

            if tries > 0:
                time.sleep(5)

        assert port_forward_port > 0, "could not establish port forwarding"

        # Assert 200 OK at /qotm/ endpoint
        qotm_url = f"http://localhost:{port_forward_port}/qotm/"
        code = get_code_with_retry(qotm_url)
        assert code == 200, f"Expected 200 OK, got {code}"
        print(f"{qotm_url} is ready")

        # Drop the port forward.
        child.terminate()
        child = None

        # --------
        self.STEP("initialize test")

        # There should be no endpoints and no endpoint deltas now -- but. If the _very 
        # first_ snapshot contains any Endpoints (entirely possible if the cluster isn't
        # empty, as it shouldn't be, and you get unlucky on timing), the snapshot may
        # post some DELETE deltas for those Endpoints. Flynn is going to leave this as-is
        # for now, since it should be harmless in practice -- a good thing to fix later 
        # though.
        #
        # So. Give the system a few tries to get to steady state.
        tries = 5

        while tries > 0:
            self.get_snapshot_info()

            if self._info and self._info.no_endpoints and self._info.no_deltas:
                # Woot.
                print(f"{self._step}: initialized")
                break

            # Still some cruft left over. Cycle.
            print(f"{self._step}: cycling for initialization, try #{6 - tries}/5")
            time.sleep(1)
            tries -= 1

        assert tries > 0, f"{self._step}: initialization incomplete after 5 tries"

        # --------
        self.STEP("switch QotM to kubernetes-endpoint")

        # When we switch the mapping to use the default endpoint resolver,
        # we should see some endpoints with corresponding ADD deltas.
        self.patch_qotm_resolver("kubernetes-endpoint")
        self.assert_some_endpoints_add_deltas()

        # --------
        self.STEP("switch QotM to kubernetes-service")

        # When we switch the mapping explicitly back to the service resolver,
        # we should see the endpoints vanish, with corresponding DELETE deltas.
        self.patch_qotm_resolver("kubernetes-service")
        self.assert_no_endpoints_delete_deltas()

        # --------
        self.STEP("install custom resolver")

        # When we install the custom endpoint resolver, nothing should change,
        # because nothing is using it yet.
        self.apply_kube_artifacts(TestEndpointFiltering.CustomResolver)
        self.assert_no_endpoints_no_deltas()

        # --------
        self.STEP("add random endpoint (1)")

        # When we add a random endpoint, again nothing should change, because
        # nothing is using an endpoint resolver.
        self.apply_kube_artifacts(TestEndpointFiltering.CustomEndpoints.format(self=self))
        self.assert_no_endpoints_no_deltas()

        # --------
        self.STEP("switch QotM to custom resolver")

        # When we switch the QotM mapping to explicitly use the the custom 
        # endpoint resolver, we should see endpoints appear, with corresponding
        # ADD deltas.
        self.patch_qotm_resolver("custom-resolver")
        self.assert_some_endpoints_add_deltas()

        # --------
        self.STEP("delete random endpoint")

        # When we delete the random endpoint, we should see it vanish -- we'll
        # still have some endpoints, but we should _also_ see DELETE deltas.
        self.delete_kube_artifacts(TestEndpointFiltering.CustomEndpoints.format(self=self))
        self.assert_some_endpoints_delete_deltas()

        # --------
        self.STEP("switch QotM to default resolver")

        # When we switch the QotM mapping back to use the default resolver,
        # we should see the endpoints vanish, with corresponding DELETE deltas.
        self.patch_qotm_resolver(None)
        self.assert_no_endpoints_delete_deltas()

        # --------
        self.STEP("switch default resolver to custom resolver")

        # When we switch the default for Ambassador as a whole to the custom
        # endpoint resolver, we should see endpoints appear again.
        self.apply_kube_artifacts(TestEndpointFiltering.AmbassadorModuleEndpoints)
        self.assert_some_endpoints_add_deltas()

        # --------
        self.STEP("add random endpoint (2)")

        # When we add a random endpoint this time, we should see it appear, since
        # we are using an endpoint resolver.
        self.apply_kube_artifacts(TestEndpointFiltering.CustomEndpoints.format(self=self))
        self.assert_some_endpoints_add_deltas()

        # --------
        self.STEP("switch default resolver back to service resolver")

        # When we switch the default for Ambassador as a whole explicitly to
        # back to the service resolver, we should see the endpoints vanish, with
        # corresponding DELETE deltas.
        self.apply_kube_artifacts(TestEndpointFiltering.AmbassadorModuleService)
        self.assert_no_endpoints_delete_deltas()

        # --------
        self.STEP("drop Ambassador module")

        # When we drop the Ambassador module entirely, nothing should change.
        self.delete_kube_artifacts(TestEndpointFiltering.AmbassadorModuleService)
        self.assert_no_endpoints_no_deltas()

    def STEP(self, stepname: str) -> None:
        self._step = stepname

        print(f"======== START STEP {self._step}")

    def guarantee_empty_namespace(self) -> None:
        nslist = run_and_assert([ "kubectl", "get", "namespace" ])

        if (nslist != None) and (self.namespace in nslist):
            # Smite.
            run_and_assert([ "kubectl", "delete", "namespace", self.namespace ])

        create_namespace(self.namespace)


    def patch_qotm_resolver(self, resolver: Optional[str]) -> None:
        # Assume a deletion...
        patch_type = "json"
        patch = '[ { "op": "remove", "path": "/spec/resolver" } ]'
        
        # ...then flip to addition if 'resolver' is defined.
        if resolver is not None:
            patch_type = "merge"
            patch = '{"spec": {"resolver": "%s"}}' % resolver

        cmd = [ "kubectl", "patch", "--namespace", self.namespace, "mapping/qotm-mapping",
                    "--type", patch_type, "--patch", patch ]

        run_and_assert(cmd)

    def apply_kube_artifacts(self, artifacts: str) -> None:
        apply_kube_artifacts(namespace=self.namespace, artifacts=artifacts)

    def delete_kube_artifacts(self, artifacts: str) -> None:
        delete_kube_artifacts(namespace=self.namespace, artifacts=artifacts)

    def get_snapshot_info(self) -> None:
        # Sleep 1 second to give the system a better chance to stabilize. This is a 
        # thing because the endpoint deltas we expect to see forced should really be 
        # the last thing that happens before no more updates happen. 
        # 
        # XXX Or maybe this is just wishful thinking. See the "NOTE WELL" at the top.
        time.sleep(1)
        self._info = SnapshotInfo(self._step, namespace=self.namespace)

    def assert_no_endpoints_no_deltas(self) -> None:
        self.get_snapshot_info()
        self.assert_no_endpoints()
        self.assert_no_deltas()

    def assert_some_endpoints_add_deltas(self) -> None:
        self.get_snapshot_info()
        self.assert_some_endpoints()

        self.filter_deltas(drop_type="add")
        self.assert_no_deltas("should have only ADD deltas")

    def assert_some_endpoints_delete_deltas(self) -> None:
        self.get_snapshot_info()
        self.assert_some_endpoints()

        self.filter_deltas(drop_type="delete")
        self.assert_no_deltas("should have only DELETE deltas")

    def assert_no_endpoints_delete_deltas(self) -> None:
        self.get_snapshot_info()
        self.assert_no_endpoints()

        self.filter_deltas(drop_type="delete")
        self.assert_no_deltas("should have only DELETE deltas")

    def assert_some_endpoints(self, msg="should have some endpoints") -> None:
        assert self._info.some_endpoints, f"{self._step}: {msg}"

    def assert_no_endpoints(self, msg="should have no endpoints") -> None:
        assert self._info.no_endpoints, f"{self._step}: {msg}\n{self._info.endpoints}"

    def assert_some_deltas(self, msg="should have some deltas") -> None:
        assert self._info.some_deltas, f"{self._step}: {msg}"

    def assert_no_deltas(self, msg="should have no deltas") -> None:
        assert self._info.no_deltas, f"{self._step}: {msg}\n{self._info.deltas}"

    def filter_deltas(self, keep_type: Optional[str]=None, drop_type: Optional[str]=None) -> None:
        filter_function = None
        
        if keep_type:
            filter_function = lambda delta: delta["deltaType"] == keep_type
        elif drop_type:
            filter_function = lambda delta: delta["deltaType"] != drop_type
        else:
            assert False, f"{self._step}: filter_deltas requires either keep_type or drop_type"

        self._info.filter_deltas(filter_function)


if __name__ == '__main__':
    pytest.main(sys.argv)