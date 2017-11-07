#!/usr/bin/env python

import sys

import requests

def test_demo(base, v2_wanted):
    url = "%s/demo/" % base

    got = {}

    for x in range(100):
        result = requests.get(url)
        version = 'unknown'

        if result.status_code != 200:
            version='failure %d' % result.status_code
        elif result.text.startswith('VERSION '):
            version=result.text[len('VERSION '):]
        else:
            version='unknown %s' % result.text

        got.setdefault(version, 0)
        got[version] += 1

    print(got)
    v2_seen = got.get('2.0.0', 0)
    rc = (abs(v2_seen - v2_wanted) < 2)

    print("wanted v2_wanted %d" % v2_wanted)
    print("saw    v2_seen   %d" % v2_seen)
    print("returning %s" % rc)

    return rc

if __name__ == "__main__":
    base = sys.argv[1]
    v2_percent = int(sys.argv[2])

    if not base.startswith("http://"):
        base = "http://%s" % base

    if test_demo(base, v2_percent):
        sys.exit(0)
    else:
        print("FAILED")
        sys.exit(1)
