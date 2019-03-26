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

import requests
import json
import yaml

class QotM (object):
    def __init__(self, target):
        self.url = "http://%s/qotm/" % target

    def decipher(self, r):
        code = r.status_code
        version = r.headers.get("x-qotm-session", "")

        return code, version

    def get(self):
        return self.decipher(requests.get(self.url))

def to_percentage(count, iterations):
    bias = iterations // 2
    return ((count * 100) + bias) // iterations

def test_qotm_auth(base, test_list, iterations=100):
    q = QotM(base)
    ran = 0
    succeeded = 0
    versions = {}

    for iteration in range(iterations):
        ran += 1
        code, version = q.get()

        # print("%d: code %d: version %s" % (iteration, code, version))

        if not version:
            version = "none"
            sys.stdout.write("-")
        else:
            sys.stdout.write(version[0])
        sys.stdout.flush()

        if code == 200:
            succeeded += 1
            versions.setdefault(version, 0)
            versions[version] += 1

    sys.stdout.write("\n")
    print("Ran       %d" % ran)
    print("Succeeded %d" % succeeded)
    print("Failed    %d" % (ran - succeeded))

    print("Versions: %s" % versions)

    for version in versions.keys():
        versions[version] = to_percentage(versions[version], iterations)

    percentages_matched = True

    for version, wanted in test_list:
        actual = versions.get(version, 0)
        delta = abs(actual - wanted)

        print("%s: wanted %d, got %d (delta %d)" % 
              (version, wanted, actual, delta))

        if delta > 2:
            percentages_matched = False

    # print("percentages_matched %s" % percentages_matched)
    # print("ran == succeeded %s" % (ran == succeeded))

    rc = 0 if ((ran == succeeded) and percentages_matched) else 1

    return rc

if __name__ == "__main__":
    base = sys.argv[1]
    elements = sys.argv[2:]

    test_list = []

    for element in elements:
        if ':' not in element:
            raise Exception("elements must be version:percentage")

        version, percentage = element.split(':')

        test_list.append((version, int(percentage)))

    sys.exit(test_qotm_auth(base, test_list))
