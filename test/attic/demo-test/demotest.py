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

import json
import os
import requests
import time
import yaml

DEFAULT_ITERATIONS=500

# Yes, it's a terrible idea to use skip cert verification for TLS.
# We really don't care for this test though.
import urllib3
urllib3.disable_warnings()

def call(url, headers=None, iterations=1):
    got = {}
    response_headers = {}

    for x in range(iterations):
        # Yes, it's a terrible idea to use skip cert verification for TLS.
        # We really don't care for this test though.
        result = requests.get(url, headers=headers, verify=False)
        version = 'unknown'

        sys.stdout.write('.')
        sys.stdout.flush()

        if result.status_code != 200:
            version='failure %d' % result.status_code
        elif result.text.startswith('VERSION '):
            version=result.text[len('VERSION '):]
        else:
            version='unknown %s' % result.text

        got.setdefault(version, 0)
        got[version] += 1
        response_headers = result.headers

    sys.stdout.write("\n")
    sys.stdout.flush()

    return got, response_headers

def to_percentage(count, iterations):
    bias = iterations // 2
    return ((count * 100) + bias) // iterations

def test_demo(base, v2_wanted):
    url = "%s/demo/" % base

    attempts = 3
    iterations = DEFAULT_ITERATIONS

    while attempts > 0:
        print("2.0.0: attempts left %d" % attempts)
        got, _ = call(url, iterations=iterations)

        print(got)
        v2_seen = to_percentage(got.get('2.0.0', 0), iterations)
        delta = abs(v2_seen - v2_wanted)
        rc = (delta <= 2)

        print("2.0.0: wanted %d, got %d (delta %d) => %s" % 
              (v2_wanted, v2_seen, delta, "pass" if rc else "FAIL"))

        if rc:
            return rc

        attempts -= 1
        print("waiting for retry")
        time.sleep(5)

    return False

def test_from_yaml(base, yaml_path):
    spec = yaml.safe_load(open(yaml_path, "r"))

    url = spec['url'].replace('{BASE}', base)

    test_num = 0
    rc = True

    for test in spec['tests']:
        test_num += 1
        name = test.get('name', "%s.%d" % (os.path.basename(yaml_path), test_num))

        headers = test.get('headers', None)
        host = test.get('host', None)
        versions = test.get('versions', None)
        expect_response_headers = test.get('expect_response_headers', None)
        iterations = test.get('iterations', DEFAULT_ITERATIONS)

        if not versions and not expect_response_headers:
            print("missing versions or expect_response_headers in %s?" % name)
            print("%s" % yaml.safe_dump(test))
            return False

        if host:
            if not headers:
                headers = {}

            headers['Host'] = host

        attempts = 3

        while attempts > 0:
            print("%s: attempts left %d" % (name, attempts))
            print("%s: headers %s" % (name, headers))

            got, response_headers = call(url, headers=headers, iterations=iterations)

            print("%s: %s" % (name, json.dumps(got)))

            test_ok = True

            if versions:
                for version, wanted_count in versions.items():
                    # Convert iterations to percent.
                    got_count = to_percentage(got.get(version, 0), iterations)
                    delta = abs(got_count - wanted_count)

                    print("%s %s: wanted %d, got %d (delta %d)" % 
                          (name, version, wanted_count, got_count, delta))

                    if delta > 2:
                        test_ok = False

            if expect_response_headers:
                for expect_header, header_value in expect_response_headers.items():
                    if expect_header in response_headers:
                        if response_headers[expect_header] != header_value:
                            print("Response header %s was expected with value %s but got %s" % 
                                  (expect_header, header_value, response_headers[expect_header]))
                            test_ok = False
                    else:
                        print("Response header %s was not returned" % (expect_header))
                        test_ok = False

            if test_ok:
                print("%s: passed" % name)
                break
            else:
                attempts -= 1

                if attempts <= 0:
                    print("%s: FAILED" % name)
                    rc = False

    return rc

if __name__ == "__main__":
    base = sys.argv[1]

    if not (base.startswith("http://") or base.startswith("https://")):
        base = "http://%s" % base

    v2_percent = None

    try:
        v2_percent = int(sys.argv[2])
    except ValueError:
        pass

    if v2_percent != None:
        rc = test_demo(base, v2_percent)
    else:
        rc = test_from_yaml(base, sys.argv[2])

    if rc:
        sys.exit(0)
    else:
        print("FAILED")
        sys.exit(1)
