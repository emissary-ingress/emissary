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

import hashlib

from .utils import SourcedDict

#############################################################################
## mapping.py -- the mapping configuration object for Ambassador
##
## Mappings are complex enough that they get their own class. Other elements
## will likely follow, but Mapping is a big deal.
##
## Each Mapping object has a group_id that reflects the group of Mappings
## that it is a part of. By definition, two Mappings with the same group_id
## are reflecting a single mapped resource that's going to multiple services.
## This implies that Mapping.group_id() is a very, very, very important 
## thing that can have dramatic customer impact if changed! (At some point,
## we should probably allow the human writing the Mapping to override the
## grouping, in much the same way we allow overriding precedence.)
##
## Each Mapping object also has a weight, which is used for ordering. The
## default is computing in Mapping.route_weight(), but it can be overridden
## using the precedence field in the Mapping object.

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
        precedence = route.get('_precedence', 0)
        prefix = route['prefix'] if 'prefix' in route else route['regex']
        method = route.get('method', 'GET')
        headers = route.get('headers', [])

        len_headers = 0

        for hdr in headers:
            len_headers += len(hdr['name']) + len(hdr.get('value', '*')) + (1 if hdr.get('regex', False) else 0)

        weight = [ precedence, len(prefix), len_headers, prefix, method ]
        weight += [ hdr['name'] + '-' + hdr.get('value', '*') for hdr in headers ]

        if not route.get('__saved', None):
            route['__saved'] = weight

        return tuple(weight)

    TransparentRouteKeys = {
        "auto_host_rewrite": True,
        "case_sensitive": True,
        "envoy_override": True,
        "host_rewrite": True,
        "path_redirect": True,
        "priority": True,
        "timeout_ms": True,
        "use_websocket": True
    }

    def __init__(self, _source="--internal--", _from=None, **kwargs):
        # Save the raw input. After this, self["anything"] will have the
        # value from the input Mapping.
        self.attrs = dict(**kwargs)

        if _from and ('_source' in _from):
            self.attrs['_source'] = _from['_source']
        else:
            self.attrs['_source'] = _source

        # ...and cache some useful first-class stuff.
        self.name = self['name']
        self.kind = self['kind']
        self.method = self.get('method', 'GET')

        # Next up, build up the headers. We do this unconditionally at init
        # time because we need the headers to work out the group ID.
        self.headers = []

        for name, value in self.get('headers', {}).items():
            if value == True:
                self.headers.append({ "name": name })
            else:
                self.headers.append({ "name": name, "value": value, "regex": False })

        for name, value in self.get('regex_headers', {}).items():
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
        self.group_id = Mapping.group_id(self.method, self['prefix'], self.headers)

    def __getitem__(self, key):
        return self.attrs[key]

    def get(self, key, *args):
        if len(args) > 0:
            return self.attrs.get(key, args[0])
        else:
            return self.attrs.get(key)

    def save_cors_element(self, cors_key, route_key, route):
        """If self.get('cors')[cors_key] exists, and
        - is a list, e.g. ["1","2","3"], then route[route_key] is set as "1, 2, 3"
        - is something else, then set route[route_key] as it is

        :param cors_key: key to exist in self.get('cors'), i.e. Ambassador's config
        :param route_key: key to save to in envoy's cors configuration
        :param route: envoy's cors configuration
        """
        cors = self.get('cors')
        if cors.get(cors_key) is not None:
            if type(cors.get(cors_key)) is list:
                route[route_key] = ", ".join(cors.get(cors_key))
            else:
                route[route_key] = cors.get(cors_key)

    def generate_route_cors(self):
        """Generates envoy's cors configuration from ambassador's cors configuration

        :return generated envoy cors configuration
        :rtype: dict
        """

        cors = self.get('cors')
        if cors is None:
            return

        route_cors = {'enabled': True}
        # cors['origins'] cannot be treated like other keys, because if it's a
        # list, then it remains as is, but if it's a string, then it's
        # converted to a list
        origins = cors.get('origins')
        if origins is not None:
            if type(origins) is list:
                route_cors['allow_origin'] = origins
            elif type(origins) is str:
                route_cors['allow_origin'] = origins.split(',')
            else:
                print("invalid cors configuration supplied - {}".format(origins))
                return

        self.save_cors_element('max_age', 'max_age', route_cors)
        self.save_cors_element('credentials', 'allow_credentials', route_cors)
        self.save_cors_element('methods', 'allow_methods', route_cors)
        self.save_cors_element('headers', 'allow_headers', route_cors)
        self.save_cors_element('exposed_headers', 'expose_headers', route_cors)
        return route_cors

    def new_route(self, svc, cluster_name):
        route = SourcedDict(
            _source=self['_source'],
            _group_id=self.group_id,
            _precedence=self.get('precedence', 0),
            prefix_rewrite=self.get('rewrite', '/')
        )

        if self.get('prefix_regex', False):
            route['regex'] = self['prefix']  # if `prefix_regex` is true, then use the `prefix` attribute as the envoy's regex
        else:
            route['prefix'] = self['prefix']

        host_redirect = self.get('host_redirect', False)
        shadow = self.get('shadow', False)

        if not host_redirect and not shadow:
            route['clusters'] = [ { "name": cluster_name,
                                    "weight": self.get("weight", None) } ]
        else:
            route.setdefault('clusters', [])

            if host_redirect and not shadow:
                route['host_redirect'] = svc
                route.setdefault('clusters', [])
            elif shadow:
                # If both shadow and host_redirect are set, we let shadow win.
                #
                # XXX CODE DUPLICATION with config.py!!
                # We're going to need to support shadow weighting later, so use a dict here.
                route['shadow'] = {
                    'name': cluster_name
                }

        if self.headers:
            route['headers'] = self.headers

        add_request_headers = self.get('add_request_headers')
        if add_request_headers:
            route['request_headers_to_add'] = []
            for key, value in add_request_headers.items():
                route['request_headers_to_add'].append({"key": key, "value": value})

        envoy_cors = self.generate_route_cors()
        if envoy_cors:
            route['cors'] = envoy_cors

        rate_limits = self.get('rate_limits')
        if rate_limits != None:
            route['rate_limits'] = []
            for rate_limit in rate_limits:
                rate_limits_actions = []
                rate_limits_actions.append({'type': 'source_cluster'})
                rate_limits_actions.append({'type': 'destination_cluster'})
                rate_limits_actions.append({'type': 'remote_address'})
                rate_limit_descriptor = rate_limit.get('descriptor', None)
                if rate_limit_descriptor:
                    rate_limits_actions.append({'type': 'generic_key', 'descriptor_value': rate_limit_descriptor})
                rate_limit_headers = rate_limit.get('headers', [])
                for rate_limit_header in rate_limit_headers:
                    rate_limits_actions.append({'type': 'request_headers', 'header_name': rate_limit_header, 'descriptor_key': rate_limit_header})
                route['rate_limits'].append({'actions': rate_limits_actions})

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
