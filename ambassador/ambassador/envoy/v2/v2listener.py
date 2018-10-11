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

from typing import List, Optional, TYPE_CHECKING
from typing import cast as typecast

from multi import multi
from ...ir.irlistener import IRListener
from ...ir.irfilter import IRFilter

from .v2tls import V2TLSContext
from .v2route import V2Route

if TYPE_CHECKING:
    from . import V2Config


ExtAuthRequestHeaders = {
    'Authorization': True,
    'Cookie': True,
    'Forwarded': True,
    'From': True,
    'Host': True,
    'Proxy-Authenticate': True,
    'Proxy-Authorization': True,
    'Set-Cookie': True,
    'User-Agent': True,
    'X-Forwarded-For': True,
    'X-Forwarded-Host': True,
    'X-Forwarded-Proto'
    'X-Gateway-Proto': True,
    'WWW-Authenticate': True,
}

@multi
def v2filter(irfilter):
    return irfilter.kind

@v2filter.when("IRAuth")
def v2filter(auth):
    request_headers = dict(ExtAuthRequestHeaders)

    for hdr in auth.allowed_headers:
        request_headers[hdr] = True

    return {
        'name': 'envoy.ext_authz',
        'config': {
            'http_service': {
                'server_uri': {
                    'uri': 'http://%s' % auth.auth_service,
                    'cluster': auth.cluster.name,
                    'timeout': '3s',
                },
                'path_prefix': auth.path_prefix,
                'allowed_authorization_headers': auth.allowed_headers,
                'allowed_request_headers': sorted(request_headers.keys())
                # 'authorization_headers_to_add': []
            }
        }
    }

@v2filter.when("ir.cors")
def v2filter(cors):
    # apparently v2 has no cors filter
    return None

@v2filter.when("ir.router")
def v2filter(router):
    return { 'name': 'envoy.router' }


class V2Listener(dict):
    def __init__(self, config: 'V2Config', listener: IRListener) -> None:
        super().__init__()

        # TODO: rate limit
        # if "rate_limits" in group:
        #     route["rate_limits"] = group.rate_limits

        # Default some things to the way they should be for the redirect listener
        name = "redirect_listener"
        envoy_ctx: Optional[dict] = None
        access_log: Optional[List[dict]] = None
        require_tls: Optional[str] = 'ALL'
        use_proxy_proto: Optional[bool] = None
        filters: List[dict] = [ { 'name': 'envoy.router' } ]
        routes: List[V2Route] = typecast(List[V2Route], [ {
            'match': {
                'prefix': '/',
            },
            'redirect': {
                'https_redirect': True,
                'path_redirect': '/'
            }
        } ])

        # OK. If this is _not_ the redirect listener, override everything.
        if not listener.redirect_listener:
            # Use the actual listener name
            name = listener.name

            # Use a sane access log spec
            access_log = [ {
                'name': 'envoy.file_access_log',
                'config': {
                    'path': '/dev/fd/1',
                    'format': 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"\n'
                }
            } ]

            # Assemble TLS contexts
            #
            # XXX Wait what? A V2TLSContext can hold only a single context, as far as I can tell...
            envoy_ctx = V2TLSContext()
            for name, ctx in config.ir.tls_contexts.items():
                config.ir.logger.info("envoy_ctx adding %s" % ctx.as_json())
                envoy_ctx.add_context(ctx)

            config.ir.logger.info("envoy_ctx final %s" % envoy_ctx)

            # Assemble filters
            filters = []
            for f in config.ir.filters:
                v2f = v2filter(f)
                if v2f:
                    filters.append(v2f)

            # Grab routes from the config.
            routes = config.routes

            # Don't require TLS.
            require_tls = None

            # Use the actual get_proxy_proto setting
            use_proxy_proto = listener.get('use_proxy_proto')

        # Finally, update the world.
        vhost = {
            'name': 'backend',
            'domains': [ '*' ],
            'routes': routes
        }

        if require_tls:
            vhost['require_tls'] = require_tls

        chain = {
            'filters': [ {
                'name': 'envoy.http_connection_manager',
                'config': {
                    'stat_prefix': 'ingress_http',
                    'access_log': access_log,
                    'http_filters': filters,
                    'route_config': {
                        'virtual_hosts': [ vhost ]
                    }
                }
            } ]
        }

        if envoy_ctx:   # envoy_ctx has to exist _and_ not be empty to be truthy
            chain['tls_context'] = dict(envoy_ctx)

        if use_proxy_proto is not None:
            chain['use_proxy_proto'] = use_proxy_proto

        self.update({
            'name': name,
            'address': {
                'socket_address': {
                    'address': '0.0.0.0',
                    'port_value': listener.service_port,
                    'protocol': 'TCP'
                }
            },
            'filter_chains': [ chain ]
        })

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.listeners = []

        for irlistener in config.ir.listeners:
            listener = config.save_element('listener', irlistener, V2Listener(config, irlistener))
            config.listeners.append(listener)
