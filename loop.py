import sys

import requests
import logging

# These two lines enable debugging at httplib level (requests->urllib3->http.client)
# You will see the REQUEST, including HEADERS and DATA, and RESPONSE with HEADERS but without DATA.
# The only thing missing will be the response.body which is not logged.
import http.client as http_client

# http_client.HTTPConnection.debuglevel = 1

# You must initialize logging, otherwise you'll not see debug output.
# logging.basicConfig(level=logging.DEBUG)
# requests_log = logging.getLogger("requests.packages.urllib3")
# requests_log.setLevel(logging.DEBUG)
# requests_log.propagate = True

session = requests.Session()

while True:
    r = session.get(sys.argv[1])

    logging.info("recv %d: %s" % (r.status_code, r.headers))

    if r.status_code != 200:
        logging.info("bzzt")
        sys.stdout.write("X")
    else:
        name = r.json()['hostname']

        if name == "v-one":
            sys.stdout.write("1")
        elif name == "v-two":
            sys.stdout.write("2")
        else:
            sys.stdout.write("?")
    
    sys.stdout.flush()

    # print(r.json()['hostname'])

    # break

