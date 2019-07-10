# Load libraries
import time
import sys
from bottle import Bottle, run, debug
from wsgiproxy import HostProxy

FILTER_HEADERS = [
    'Connection',
    'Keep-Alive',
    'Proxy-Authenticate',
    'Proxy-Authorization',
    'TE',
    'Trailers',
    'Transfer-Encoding',
    'Upgrade',
    ]


root=Bottle()
proxy=HostProxy("http://localhost:8680/")
root.mount("/portal/", proxy)

# Handle http requests to the root address
@root.route('/')
def index():
 return 'Go away.'

# Handle http requests to the root address
@root.route('/_shutdown')
def index():
  print("\n\nFake ambassador: Shutting down\n\n")
  sys.exit(0)

@root.route('/ambassador/v0/diag/')
def diag():
  return dict(
    groups=dict(foo=dict(
    _active=True,
    kind="IRHTTPMappingGroup",
    mappings=[dict(
      location="p1.p2.p3",
      prefix="/yuhu",
      )]
)),
  )

@root.route('/yuhu/.ambassador-internal/openapi-docs')
def swagger():
  return dict(
swagger="2.0",
info=dict(version="0.314", title="fake fake fake"),
host="localhost:8877/example/",
paths={"/foo":dict(get=dict(responses={"200":dict(description="all is good")}))}
)

@root.route('/example/foo')
def foo():
  return dict(version="1.0",logic="missing")

try:
  run(root, host='0.0.0.0', port=8877)
except:
  print("Done")
