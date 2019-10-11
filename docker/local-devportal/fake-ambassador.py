# Load libraries
import time
import sys
from bottle import Bottle, run, debug, redirect
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

root.mount("/docs/", HostProxy("http://localhost:8680/docs/"))
root.mount("/openapi/", HostProxy("http://localhost:8680/openapi/"))

# Handle http requests to the root address
@root.route('/')
def index():
  redirect("/docs/")

# Handle http requests to the root address
@root.route('/_shutdown')
def index():
  print("\n\nFake ambassador: Shutting down\n\n")
  sys.stderr.close()

@root.route('/ambassador/v0/diag/')
def diag():
  return dict(
    groups=dict(
      foo=dict(
        _active=True,
        kind="IRHTTPMappingGroup",
        mappings=[dict(
          location="service.example.p3",
          prefix="/yuhu",
          rewrite="",
          name="yuhu",
        )]),
      bar=dict(
        _active=True,
        kind="IRHTTPMappingGroup",
        mappings=[dict(
          location="service.another.p3",
          prefix="/another",
          rewrite="",
          name="another",
        )]),
      ),
  )

@root.route('/yuhu/.ambassador-internal/openapi-docs')
@root.route('/another/.ambassador-internal/openapi-docs')
def swagger():
  return dict(
    swagger="2.0",
    info=dict(version="0.314", title="An example Open API service"),
    host="localhost:8877/example/",
    paths={
      "/foo": dict(
        get=dict(
          tags=["foo", "bar"],
          description="An example request with no markdown",
          responses={"200":dict(description="all is good")})),
      "/bar": dict(
        get=dict(
          tags=["bar"],
          summary="Short summary",
          description="# Another example request\n\n with a markdown description",
          responses={"200":dict(description="all is good")})),
    }
)

@root.route('/example/foo')
@root.route('/example/bar')
def foo():
  return dict(version="1.0",logic="missing")

try:
  run(root, host='0.0.0.0', port=8877)
except:
  print("Done")
