#!/usr/bin/env python

import logging
import re
import socket
import requests

from flask import Flask, jsonify, request

logPath = "/tmp/flasklog"

MyHostName = socket.gethostname()
MyResolvedName = socket.gethostbyname(socket.gethostname())

logging.basicConfig(
    filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%(asctime)s ambassador-sds 0.0.1 %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

logging.info("ambassador-sds initializing on %s (resolved %s)" % (MyHostName, MyResolvedName))

TOKEN = open("/var/run/secrets/kubernetes.io/serviceaccount/token", "r").read()
ENDPOINT_URL_TEMPLATE = "https://kubernetes/api/v1/namespaces/default/endpoints/%s"
ENDPOINTS = {}

SERVICE_RE = re.compile(r'^[a-z0-9-_]+$')

app = Flask(__name__)

@app.route('/v1/registration/<service_name>', methods=[ 'GET' ])
def handle_endpoint(service_name):
    if not SERVICE_RE.match(service_name):
        return "invalid service name '%s'" % service_name, 503

    url = ENDPOINT_URL_TEMPLATE % service_name

    r = requests.get(url, headers={"Authorization": "Bearer " + TOKEN}, verify=False)

    if r.status_code != 200:
        return jsonify({ "hosts": [] })

    endpoints = r.json()

    hostdicts = []
    ports = []

    subsets = endpoints.get("subsets", [])

    for subset in subsets:
        for portdef in subset.get("ports", []):
            if ((portdef['protocol'] == 'TCP') and
                (('name' not in portdef) or
                 (portdef['name'] == service_name))):
                ports.append(portdef['port'])

        for addrdef in subset.get("addresses", []):
            if "ip" in addrdef:
                for port in ports:
                    hostdicts.append({
                        "ip_address": addrdef["ip"],
                        "port": port,
                        "tags": {}
                    })

    return jsonify({ "hosts": hostdicts })

@app.route('/v1/health')
def root():
    return jsonify({ "ok": True, "msg": "SDS healthy!" })

def main():
    app.run(host='0.0.0.0', port=5000, debug=True)

if __name__ == '__main__':
    main()
