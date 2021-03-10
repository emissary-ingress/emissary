from typing import Optional

import sys
import yaml

from flask import Flask, jsonify, request

from ambassador.utils import parse_yaml, yaml_dumper


app = Flask(__name__)

def merge_dicts(x, y):
    z = x.copy()
    z.update(y)
    return z

@app.route('/api/snapshot/<generation_counter>/<kind>')
def services(generation_counter, kind):
    if kind in app.elements:
        return yaml.dump_all(app.elements[kind], Dumper=yaml_dumper), 200
    else:
        return "no such element", 404


def main(services_path: str, endpoint_path: Optional[str]):
    k8s_resources = parse_yaml(open(services_path, 'r').read())
    total_resources = len(k8s_resources)

    services = [ obj for obj in k8s_resources if obj.get('kind', None) == 'Service' ]

    app.elements = {
        'services': services
    }

    if endpoint_path:
        k8s_resources = parse_yaml(open(endpoint_path, 'r').read())
        total_resources += len(k8s_resources)

        app.elements['endpoints'] = [ obj for obj in k8s_resources if obj.get('kind', None) == 'Endpoints' ]

    print("Total resources: %d" % total_resources)
    print("Services:        %d" % len(app.elements['services']))
    print("Endpoints:       %d" % len(app.elements.get('endpoints', [])))

    app.run(host='0.0.0.0', port=9999, debug=True)


if __name__ == '__main__':
    services_path = sys.argv[1]
    endpoint_path = None

    if len(sys.argv) > 2:
        endpoint_path = sys.argv[2]

    main(services_path, endpoint_path)
