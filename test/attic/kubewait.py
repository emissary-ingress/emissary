#!python

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import sys

import os
import re
import time

import dpath.util
import requests

from kubernetes import client, config, watch

def kube_v1():
    # Assume we got nothin'.
    k8s_api = None

    # XXX: is there a better way to check if we are inside a cluster or not?
    if "KUBERNETES_SERVICE_HOST" in os.environ:
        # If this goes horribly wrong and raises an exception (it shouldn't),
        # we'll crash, and Kubernetes will kill the pod. That's probably not an
        # unreasonable response.
        config.load_incluster_config()
        k8s_api = client.CoreV1Api()
    else:
        # Here, we might be running in docker, in which case we'll likely not
        # have any Kube secrets, and that's OK.
        try:
            config.load_kube_config()
            k8s_api = client.CoreV1Api()
        except FileNotFoundError:
            # Meh, just ride through.
            print("No K8s")
            pass

    return k8s_api

class Waitable (object):
    def __init__(self, retries=60, delay=2):
        self.retries = retries
        self.remaining = 0
        self.delay = delay

    def check(self):
        raise Exception("You need to subclass Waitable and customize check, name, and not_ready")

    def name(self):
        raise Exception("You need to subclass Waitable and customize check, name, and not_ready")

    def not_ready(self):
        raise Exception("You need to subclass Waitable and customize check, name, and not_ready")

    def wait(self, retries=None, delay=None):
        self.remaining = retries if retries else self.retries

        if not delay:
            delay = self.delay

        good=False

        while self.remaining > 0:
            self.remaining -= 1

            if self.check():
                good=True
                break

            print("%s remaining %02d: %s" % (self.name, self.remaining, self.not_ready))
            time.sleep(delay)

        return good

class WaitForPods (Waitable):
    def __init__(self, namespace="default", **kwargs):
        super(WaitForPods, self).__init__(**kwargs)
        self.pending = -1
        self.namespace = namespace
        self.k8s_api = kube_v1()

    @property
    def name(self):
        return "WaitForPods"

    @property
    def not_ready(self):
        if self.pending < 0:
            return "no pods being checked"
        else:
            return "%d not running" % self.pending

    def check(self):
        self.pending = 0

        for pod in self.k8s_api.list_namespaced_pod("default").items:
            # print("%s found %s -- %s" % (self.name, pod.metadata.name, pod.status.phase))
            if pod.status.phase != "Running":
                self.pending += 1

        rc = self.pending == 0

        # print("%s check returning %s" % (self.name, rc))

        return rc

class WaitForURL (Waitable):
    def __init__(self, url, expected, name=None, not_ready="not yet ready", **kwargs):
        super(WaitForURL, self).__init__(**kwargs)
        self.url = url
        self.expected = re.compile(expected)
        self._name = name
        self._not_ready = not_ready

    @property
    def name(self):
        return self._name if self._name else "WaitForURL"

    @property
    def not_ready(self):
        return self._not_ready

    def check(self):
        result = requests.get(self.url)

        if (result.status_code // 100) != 2:
            print("%s: GET failed (%d)" % (self.name, result.status_code))
            return False

        text = result.text

        if self.expected.search(text):
            print("%s: Matched on '%s'" % (self.name, text))
            return True
        else:
            print("%s: no match on '%s'" % (self.name, text))
            return False

if __name__ == "__main__":
    if WaitForURL(sys.argv[1], sys.argv[2]).wait():
        print("Everything OK")
    else:
        print("Uhoh")
