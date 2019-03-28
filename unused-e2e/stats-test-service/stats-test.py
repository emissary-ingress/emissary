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

import json
import socket

UDP_IP = "0.0.0.0"
UDP_PORT = 8125

sock = socket.socket(socket.AF_INET, # Internet
                     socket.SOCK_DGRAM) # UDP
sock.bind((UDP_IP, UDP_PORT))

sys.stdout.write("Listening on %d\n" % UDP_PORT)
sys.stdout.flush()

interesting = {
    'envoy.cluster.cluster_qotm.upstream_rq_time': 0,
    'envoy.cluster.cluster_qotm.upstream_rq_total': 0,
    'envoy.cluster.cluster_qotm.upstream_rq_2xx': 0
}

recvd = 0

while True:
    data, addr = sock.recvfrom(1024) # buffer size is 1024 bytes

    data = data.decode('utf-8').strip()
    peer_ip, peer_port = addr

    if data == 'DUMP':
        sock.sendto(json.dumps(interesting).encode("utf-8"), addr)
    else:
        recvd += 1

        try:
            (key, rest) = data.split(':')
            (value, kind) = rest.split('|')

            if key not in interesting:
                continue
            
            interesting[key] += int(value)
        except Exception:
            continue
    
        if (recvd % 60) == 0:
            trq = interesting['envoy.cluster.cluster_qotm.upstream_rq_total']
            grq = interesting['envoy.cluster.cluster_qotm.upstream_rq_2xx']
            ttm = interesting['envoy.cluster.cluster_qotm.upstream_rq_time']

            if trq > 0:
                ttm = ", %.1f ms avg" % (ttm / trq)
            else:
                ttm = ""

            sys.stdout.write("Requests: %d (%d good)%s\n" % (trq, grq, ttm))
            sys.stdout.flush()


