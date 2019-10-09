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
        self.base = "http://%s" % target

    def auth(self, d):
        x = dict(d)
        x.update({ "username": "username", "password": "password" })

        return x

    def build(self, qid=None, quote=None, username=None, password=None):
        url = "%s/qotm/" % self.base

        if qid:
            url += "quote"

            if qid != "new":
                url += "/%s" % qid

        extra_args = {}

        if username and password:
            extra_args["auth"] = ( username, password )

        if quote:
            extra_args["json"] = { "quote": quote }

        return ( url, extra_args )

    def decipher(self, r):
        code = r.status_code
        result = None

        try:
            result = r.json()
        except:
            pass

        return code, result

    def get(self, qid=None, username=None, password=None):
        url, args = self.build(qid=qid, username=username, password=password)

        return self.decipher(requests.get(url, **args))

    def put(self, qid=None, quote=None, username=None, password=None):
        url, args = self.build(qid=qid, quote=quote, username=username, password=password)

        return self.decipher(requests.put(url, **args))

    def post(self, qid=None, quote=None, username=None, password=None):
        url, args = self.build(qid=qid, quote=quote, username=username, password=password)

        return self.decipher(requests.post(url, **args))

def ok_based_on_code(code):
    # If it's a 2yz, expect True. If not, expect that we won't get a JSON response 
    # at all.
    return True if ((code // 100) == 2) else None

def test_qotm(base, test_list):
    q = QotM(base)
    ran = 0
    succeeded = 0
    saved = {}

    for test_info in test_list:
        test_name = test_info['name']
        method = test_info['method']
        args = test_info['args']
        checks = test_info.get('checks')
        updates = test_info.get('updates')

        expected_code = 200
        expected_ok = True

        expect = test_info.get('expect')

        if expect:
            if isinstance(expect, int):
                expected_code = expect
                expected_ok = ok_based_on_code(expected_code)
            elif isinstance(expect, dict):
                expected_code = expect.get('code', 200)
                expected_ok = expect.get('ok', ok_based_on_code(expected_code))
            else:
                print("%s: bad 'expected' value '%s'" % (test_name, json.dumps(expect)))
                continue

        auth_name = test_info.get('auth', None)

        if auth_name == 'default':
            args['username'] = 'username'
            args['password'] = 'password'
        elif auth_name:
            print("%s: bad 'auth' value '%s'" % (test_name, auth_name))
            continue

        fn = getattr(q, method)

        interpolated_args = {}
        missed_values = 0

        for name, value in args.items():
            if isinstance(value, str) and value.startswith("$"):
                aname = value[1:]
                value = saved.get(aname, None)

                if not value:
                    print("%s: saved variable %s for %s is empty" % (test_name, aname, name))
                    missed_values += 1

            interpolated_args[name] = value

        code, result = fn(**interpolated_args)

        # print("%s got %d, %s" % (test_name, code, result))

        ran += 1

        if code != expected_code:
            print("%s: wanted %d, got %d" % (test_name, expected_code, code))
            continue

        if expected_ok != None:
            if result['ok'] != expected_ok:
                print("%s: wanted %d, got %d" % (test_name, expected_ok, result['ok']))
                continue

        if updates:
            missed_updates = 0

            for name in updates:
                value = result.get(name, None)

                if not value:
                    print("%s: wanted %s but didn't find it" % (test_name, name))
                    missed_updates += 1
                else:
                    saved[name] = value
                    print("saved %s: %s" % (name, value))

            if missed_updates:
                continue

        if checks:
            missed_checks = 0

            for name, wanted in checks.items():
                got = result.get(name, None)

                if got != wanted:
                    print("%s: wanted %s %s, got %s" % (test_name, name, wanted, got))
                    missed_checks += 1

            if missed_checks:
                continue

        print("%s => %d: passed" % (test_name, expected_code))
        succeeded += 1

    print("Ran       %d" % ran)
    print("Succeeded %d" % succeeded)
    print("Failed    %d" % (ran - succeeded))

    return ran - succeeded

if __name__ == "__main__":
    base = sys.argv[1]
    yaml_path = sys.argv[2]

    test_list = yaml.safe_load(open(yaml_path, "r"))

    if 'tests' in test_list:
        test_list = test_list['tests']

    sys.exit(test_qotm(base, test_list))
