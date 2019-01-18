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


class QotM(object):
    def __init__(self, target):
        self.url = "%s/qotm/" % target

    def decipher(self, r):
        return r.status_code

    def get(self, headers):
        return self.decipher(requests.get(self.url, headers=headers))


class ZipkinServices(object):
    def __init__(self, target):
        self.url = "%s/api/v2" % target

    def decipher(self, r):
        code = r.status_code
        result = None

        try:
            result = r.json()
        except:
            pass

        return code, result

    def getServices(self):
        return self.decipher(requests.get("%s/services" % self.url))

    def getTraces(self):
        return self.decipher(requests.get("%s/traces" % self.url))

    def getSpans(self, serviceName):
        return self.decipher(requests.get("%s/spans?serviceName=%s" % (self.url, serviceName)))


def test_qotm_tracing(base, zipkin, test_list, iterations=5):
    q = QotM(base)
    z = ZipkinServices(zipkin)

    for iteration in range(iterations):
        for headers, expected_code in test_list:
            code = q.get(headers)
            if code != expected_code:
                print("%s: expected %d, got %d" % (headers, expected_code, code))

    _, zipkinServices = z.getServices()
    print("Zipkin Services %s" % zipkinServices)

    _, zipkinTraces = z.getTraces()
    print("Zipkin Traces len %d" % len(zipkinTraces))
    print("Zipkin TraceId %s" % zipkinTraces[0][0]['traceId'])

    _, zipkinSpans = z.getSpans(zipkinServices[0])
    print("Zipkin Spans %s" % zipkinSpans)

    return 0 if (
            len(zipkinServices) == 1 and
            zipkinServices[0] == '015-tracing-015-tracing' and
            len(zipkinTraces) >= 5 and  # each iteration should generate a trace
            len(zipkinTraces[0][0]['traceId']) >= 32 and  # traceId is 128bit
            len(zipkinSpans) >= 3
    ) else 1


if __name__ == "__main__":
    base = sys.argv[1]
    zipkin = sys.argv[2]

    test_list = []

    test_list.append((None, 200))

    sys.exit(test_qotm_tracing(base, zipkin, test_list))
