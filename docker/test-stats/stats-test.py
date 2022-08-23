import sys

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

import datetime
import json
import logging
import os
import socket

__version__ = '0.0.14'

logging.basicConfig(
    level=logging.DEBUG if os.environ.get('STATSD_TEST_DEBUG') else logging.INFO,
    format="%%(asctime)s stats-test %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

UDP_IP = "0.0.0.0"
UDP_PORT = 8125

sock = socket.socket(socket.AF_INET, # Internet
                     socket.SOCK_DGRAM) # UDP
sock.bind((UDP_IP, UDP_PORT))

logging.info("Listening on %d" % UDP_PORT)

interesting_clusters = {
    cname for cname in os.environ.get('STATSD_TEST_CLUSTER').split(":")
}

logging.info(f"Interesting clusters: {interesting_clusters}")

interesting = {}

last_summary = datetime.datetime.now()

def summary():
    r = ""

    for cluster_name in interesting_clusters:
        cluster = interesting.get(cluster_name, {})
        trq = cluster.get(f'upstream_rq_total', -1)
        grq = cluster.get(f'upstream_rq_2xx', -1)
        ttm = cluster.get(f'upstream_rq_time', -1)

        if (trq > 0) and (trq > 0):
            ttm = ", %.1f ms avg" % (ttm / trq)

            r += f'{cluster_name}: {trq} req, {grq} good{ttm}\n'

    return r

while True:
    now = datetime.datetime.now()

    if (now - last_summary) > datetime.timedelta(seconds=30):
        logging.info(f"30sec\n{summary()}")
        last_summary = datetime.datetime.now()

    data, addr = sock.recvfrom(1024) # buffer size is 1024 bytes

    data = data.decode('utf-8').strip()
    peer_ip, peer_port = addr

    logging.debug(f"data: {data}")

    if data == 'RESET':
        logging.info('RESETTING')

        interesting = {}

        sock.sendto(bytes('RESET', 'utf-8'), addr)
    elif data == 'DUMP':
        logging.info('DUMP')

        contents = json.dumps(interesting).encode("utf-8")
        logging.info(f"SEND {contents}")

        sock.sendto(contents, addr)
    elif data == 'SUMMARY':

        contents = summary().encode("utf-8")
        logging.info('SUMMARY:\n{contents}')

        sock.sendto(contents, addr)
    else:
        # Here's a sample 'normal' line:
        # envoy.cluster.cluster_http___statsdtest_http.upstream_rq_200:310|c
        # envoy.cluster.cluster_http___statsdtest_http.upstream_rq_2xx:310|c
        # envoy.cluster.cluster_http___statsdtest_http.upstream_rq_time:3|ms
        #
        # and here's the dogstatsd equivalent:
        # envoy.cluster.upstream_rq:363|c|#envoy.response_code:200,envoy.cluster_name:cluster_http___dogstatsdtest_http
        # envoy.cluster.upstream_rq_xx:363|c|#envoy.response_code_class:2,envoy.cluster_name:cluster_http___dogstatsdtest_http
        # envoy.cluster.upstream_rq_time:2|ms|#envoy.cluster_name:cluster_http___dogstatsdtest_http
        #
        # So first, it needs to start with 'envoy.cluster.'.

        if not data.startswith('envoy.cluster.'):
            # logging.info(f"SKIP: {data}")
            continue

        logging.info(f"CLUSTER: {data}")

        # Strip the leading 'envoy.cluster.'...
        data = data[len('envoy.cluster.'):]

        # Next up, split fields out.
        fields = data.split('|')

        if (len(fields) < 2) or (len(fields) > 3):
            logging.debug(f'bad fields {fields}')
            continue

        key_and_value = fields[0]
        data_type = fields[1]
        dog_elements = {}

        if len(fields) > 2:
            dog_stuff = fields[2]

            if not dog_stuff.startswith('#'):
                logging.debug(f'bad dog_stuff {dog_stuff}')
                continue

            dog_stuff = dog_stuff[1:]

            for dog_element in dog_stuff.split(','):
                dog_key, dog_value = dog_element.split(':', 1)
                dog_elements[dog_key] = dog_value

        key, value = key_and_value.split(':', 1)
        cluster_name = None

        if not dog_elements:
            # No datadog stuff, so we should be able to grab the cluster name
            # from the key.
            cluster_name, key = key.split('.', 1)
        else:
            cluster_name = dog_elements.get('envoy.cluster_name')

        if not cluster_name:
            logging.debug('no cluster_name')
            continue

        # Is this an interesting cluster?
        if cluster_name not in interesting_clusters:
            logging.debug(f'{cluster_name} is uninteresting')
            continue

        # Finally, fix up the dogstatsd stat keys.
        if dog_elements:
            if key.endswith('_rq') and ('envoy.response_code' in dog_elements):
                key = f'{key}_{dog_elements["envoy.response_code"]}'
            elif key.endswith('_xx') and ('envoy.response_code_class' in dog_elements):
                rclass = dog_elements["envoy.response_code_class"]
                key = key.replace('_xx', f'_{rclass}xx')

        # logging.info(f'{cluster_name}: {key} += {value} {data_type}')

        cluster_stats = interesting.setdefault(cluster_name, {})

        if key not in cluster_stats:
            cluster_stats[key] = 0

        cluster_stats[key] += int(value)
