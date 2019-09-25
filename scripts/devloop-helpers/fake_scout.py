from typing import Dict

import datetime
import json
import uuid

from flask import Flask, jsonify, request

from ambassador.utils import parse_yaml


class FakeScoutApp (Flask):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.counts: Dict[str, int] = {}


app = FakeScoutApp(__name__)

def merge_dicts(x, y):
    z = x.copy()
    z.update(y)
    return z


@app.route('/scout', methods=['POST'])
def report():
    payload = request.json

    print("\n---- %s" % datetime.datetime.now().isoformat())
    print(json.dumps(payload, sort_keys=True, indent=4))

    application = str(payload.get('application', '')).lower()

    if application not in app.counts:
        app.counts[application] = 0

    app.counts[application] += 1

    result = {
        "latest_version": "0.52.1",
        "application": application,
        "cached": False,
        "count": app.counts[application],
        "timestamp": datetime.datetime.now().timestamp(),
        "notices": [{ "level": "warning", "message": "Scout response is faked!" }]
    }

    return jsonify(result), 200


def main():
    print("fake_scout listening on port 9999")
    app.run(host='0.0.0.0', port=9999, debug=True)


if __name__ == '__main__':
    main()
