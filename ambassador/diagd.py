#!python

import sys

import json
import logging

import VERSION

from flask import Flask, Response, jsonify, request
from AmbassadorConfig import AmbassadorConfig

__version__ = VERSION.Version

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s diagd %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

aconf = AmbassadorConfig(sys.argv[1])

app = Flask(__name__)

@app.route('/ambassador/v0/check_alive', methods=[ 'GET' ])
def check_alive():
    return "ambassador liveness check OK"

@app.route('/ambassador/v0/check_ready', methods=[ 'GET' ])
def check_ready():
    return "ambassador readiness check OK"

@app.route('/ambassador/v0/diag/<path:source>', methods=[ 'GET' ])
def show_intermediate(source=None):
    logging.debug("getting intermediate for '%s'" % source)
    result = aconf.get_intermediate_for(source)
    output = []

    if result['sources']:
        for source in sorted(result['sources'], key=lambda x: "%s.%d" % (x['filename'], x['index'])):
            output.append("# %s[%d]" % (source['filename'], source['index']))
            output.append("---")
            output.append(source['yaml'])

        # link types back to Ambassador and Envoy docs
        # present cluster stats, too
        for type in [ 'listeners', 'filters', 'routes', 'clusters' ]:
            if result[type]:
                output.append("--------")
                output.append("%s:" % type)

                first = True

                for element in sorted(result[type], key=lambda x: x['_source']):
                    source = element['_source']
                    refs = element.get('_referenced_by', [])

                    output.append("# created by %s" % source)

                    if refs:
                        output.append("# referenced by %s" % ", ".join(sorted(refs)))

                    x = dict(**element)
                    del(x['_source'])

                    if '_referenced_by' in x:
                        del(x['_referenced_by'])

                    output.append("%s\n" % json.dumps(x, indent=4))

                    if not first:
                        output.append("")

                    first = False

    return "\n".join(output)

app.run(host='127.0.0.1', port=8888, debug=True)
