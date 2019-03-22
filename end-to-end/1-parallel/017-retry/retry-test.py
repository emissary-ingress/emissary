#!/usr/bin/env python

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
import time
import requests


class Retry(object):
    def __init__(self, target):
        self.url = "http://%s/retry/" % target

    def decipher(self, r):
        return r.status_code

    def get(self):
        return self.decipher(requests.get(self.url))


def test_retry(base, iterations=100, target_success_rate=0.9):
    r = Retry(base)
    ran = 0
    succeeded = 0

    for iteration in range(iterations):
        ran += 1
        code = r.get()
        if code == 200:
            succeeded += 1
        else:
            print("expected %d, got %d" % (200, code))

    success_rate = (succeeded / ran)
    sys.stdout.write("\n")
    print("Ran           %d" % ran)
    print("Succeeded     %d" % succeeded)
    print("Failed        %d" % (ran - succeeded))
    print("Success rate  %f%%" % (success_rate))

    # This is a bit flaky, requests are sampled by Envoy and could timeout
    return 0 if (success_rate > target_success_rate) else 1


if __name__ == "__main__":
    base = sys.argv[1]

    sys.exit(test_retry(base))
