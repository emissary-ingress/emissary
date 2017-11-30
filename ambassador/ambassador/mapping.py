import hashlib

from .utils import SourcedDict

class Mapping (object):
    @classmethod
    def group_id(klass, method, prefix, headers):
        # Yes, we're using a  cryptographic hash here. Cope. [ :) ]

        h = hashlib.new('sha1')
        h.update(method.encode('utf-8'))
        h.update(prefix.encode('utf-8'))

        for hdr in headers:
            h.update(hdr['name'].encode('utf-8'))

            if 'value' in hdr:
                h.update(hdr['value'].encode('utf-8'))

        return h.hexdigest()

    @classmethod
    def route_weight(klass, route):
        prefix = route['prefix']
        method = route.get('method', 'GET')
        headers = route.get('headers', [])

        weight = [ len(prefix) + len(headers), prefix, method ]
        weight += [ hdr['name'] for hdr in headers ]

        return tuple(weight)

    TransparentRouteKeys = {
        "host_redirect": True,
        "path_redirect": True,
        "host_rewrite": True,
        "auto_host_rewrite": True,
        "case_sensitive": True,
        "use_websocket": True,
        "timeout_ms": True,
        "priority": True,
    }

    def __init__(self, _source="--internal--", _from=None, **kwargs):
        # Save the raw input...
        self.attrs = dict(**kwargs)

        if _from and ('_source' in _from):
            self.attrs['_source'] = _from['_source']
        else:
            self.attrs['_source'] = _source

        # ...and cache some useful first-class stuff.
        self.name = self['name']
        self.kind = self['kind']
        self.prefix = self['prefix']
        self.method = self.get('method', 'GET')

        # Next up, build up the headers. We do this unconditionally at init
        # time because we need the headers to work out the group ID.
        self.headers = []

        for name, value in self.get('headers', {}).items():
            if value == True:
                self.headers.append({ "name": name })
            else:
                self.headers.append({ "name": name, "value": value, "regex": False })

        for name, value in self.get('regex_headers', []):
            self.headers.append({ "name": name, "value": value, "regex": True })

        if 'host' in self.attrs:
            self.headers.append({
                "name": ":authority",
                "value": self['host'],
                "regex": self.get('host_regex', False)
            })

        if 'method' in self.attrs:
            self.headers.append({
                "name": ":method",
                "value": self['method'],
                "regex": self.get('method_regex', False)
            })

        # OK. After all that we can compute the group ID.
        self.group_id = Mapping.group_id(self.method, self.prefix, self.headers)

    def __getitem__(self, key):
        return self.attrs[key]

    def get(self, key, default):
        return self.attrs.get(key, default)

    def new_route(self, cluster_name):
        route = SourcedDict(
            _source=self['_source'],
            group_id=self.group_id,
            prefix=self.prefix,
            prefix_rewrite=self.get('rewrite', '/'),
            clusters=[ { "name": cluster_name,
                         "weight": self.get("weight", None) } ]
        )

        if self.headers:
            route['headers'] = self.headers

        # Even though we don't use it for generating the Envoy config, go ahead
        # and make sure that any ':method' header match gets saved under the
        # route's '_method' key -- diag uses it to make life easier.

        route['_method'] = self.method

        # We refer to this route, of course.
        route._mark_referenced_by(self['_source'])

        # There's a slew of things we'll just copy over transparently; handle
        # those.

        for key, value in self.attrs.items():
            if key in Mapping.TransparentRouteKeys:
                route[key] = value

        # Done!
        return route

if __name__ == "__main__":
    import sys

    import json
    import os

    import yaml

    for path in sys.argv[1:]:
        try:
            # XXX This is a bit of a hack -- yaml.safe_load_all returns a
            # generator, and if we don't use list() here, any exception
            # dealing with the actual object gets deferred 
            objects = list(yaml.safe_load_all(open(path, "r")))
        except Exception as e:
            print("%s: could not parse YAML: %s" % (path, e))
            continue

        ocount = 0
        for obj in objects:
            ocount += 1
            srckey = "%s.%d" % (path, ocount)

            if obj['kind'] == 'Mapping':
                m = Mapping(srckey, **obj)

                print("%s: %s" % (m.name, m.group_id))

                print(json.dumps(m.new_route("test_cluster"), indent=4, sort_keys=True))
