#!/usr/bin/env python

#
# Simple command line WebSocket client.
#

import argparse
import json
import os
import socket
import sys
import websocket


enable_verbose = False

def vprint(msg):
    if enable_verbose:
        print("> %s" % msg)


def connect(url, subprotocol):
    header = {}
    if not subprotocol:
        vprint("Connecting to %s" % url)
    else:
        vprint("Connecting to %s (subprotocol: %s)" % (url, subprotocol))
        header['Sec-WebSocket-Protocol'] = subprotocol

    try:
        return websocket.create_connection(url, header=header)
    except socket.error as e:
        print("Connection failed: %r" % e, file=sys.stderr)
        sys.exit(1)


def main(argv):
    os.close(0) # we don't need stdin

    parser = argparse.ArgumentParser(
        description="WebSocket command line client")
    parser.add_argument("url", nargs=1, help="URL for WebSocket to connect to")
    parser.add_argument("--pretty-json", help="Pretty-print received JSON",
                        action="store_const", const=True, default=False)
    parser.add_argument("--subprotocol", "-s", help="WebSocket subprotocol")
    parser.add_argument("--verbose", "-v", help="Verbose printing",
                        action="store_true")

    args = parser.parse_args()

    if args.verbose:
        global enable_verbose
        enable_verbose = True

    ws = connect(args.url[0], args.subprotocol)
    try:
        for msg_str in iter(lambda: ws.recv(), None):
            try:
                if args.pretty_json:
                    msg_json = json.loads(msg_str)
                    json.dump(msg_json, sys.stdout, indent=2, sort_keys=True,
                              separators=(',', ': '))
                    # append a newline
                    print()
                else:
                    print(msg_str)
                break
            except:
                print("Failed to parse: %s" % msg_str, file=sys.stderr)
    except websocket.WebSocketConnectionClosedException:
        vprint("Connection closed")
    except KeyboardInterrupt:
        vprint("Interrupted")
    finally:
        if ws:
            vprint("Closing")
            ws.close()

if __name__ == "__main__":
    sys.exit(main(sys.argv) or 0)
