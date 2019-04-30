from typing import Optional

import sys
import yaml

from flask import Flask, jsonify, request

app = Flask(__name__)

@app.route('/api/snapshot/<generation_counter>/<kind>')
def services(generation_counter, kind):
    try:
        with open(app.snapshot_path, 'r') as snapshot:
            return snapshot.read(), 200
    except Exception as e:
        return "uhoh (%s)" % e, 500


def main(snapshot_path: str):
    app.snapshot_path = snapshot_path

    print("serving on 9999 from %s" % snapshot_path)
    app.run(host='0.0.0.0', port=9999, debug=True)


if __name__ == '__main__':
    snapshot_path = sys.argv[1]

    main(snapshot_path)
