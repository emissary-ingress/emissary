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

from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING
from typing import cast as typecast

import json

# from copy import deepcopy

from multi import multi
from ...ir.irlistener import IRListener
from ...ir.irauth import IRAuth
from ...ir.irbuffer import IRBuffer
from ...ir.irfilter import IRFilter
from ...ir.irratelimit import IRRateLimit
from ...ir.ircors import IRCORS
from ...ir.ircluster import IRCluster

from .v2tls import V2TLSContext
# from .v2route import V2Route

if TYPE_CHECKING:
    from . import V2Config

# Static header keys normally used in the context of an authorization request.
AllowedRequestHeaders = frozenset([
    'authorization',
    'cookie',
    'from',
    'proxy-authorization',
    'user-agent',
    'x-forwarded-for',
    'x-forwarded-host',
    'x-forwarded-proto'
])

# Static header keys normally used in the context of an authorization response.
AllowedAuthorizationHeaders = frozenset([
    'location',
    'authorization',
    'proxy-authenticate',
    'set-cookie',
    'www-authenticate'
])

# This mapping is only used for ambassador/v0.
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
    'x-b3-flags': True,
    'x-b3-parentspanid': True,
    'x-b3-traceid': True,
    'x-b3-sampled': True,
    'x-b3-spanid': True,
    'X-Forwarded-For': True,
    'X-Forwarded-Host': True,
    'X-Forwarded-Proto': True,
    'X-Gateway-Proto': True,
    'x-ot-span-context': True,
    'WWW-Authenticate': True,
}


@multi
def v2filter(irfilter: IRFilter):
    return irfilter.kind


@v2filter.when("IRBuffer")
def v2filter_buffer(buffer: IRBuffer):
    return {
        'name': 'envoy.buffer',
        'config': {
            "max_request_time": "%0.3fs" % (float(buffer.max_request_time) / 1000.0),
            "max_request_bytes": buffer.max_request_bytes
        }        
    }


@v2filter.when("IRAuth")
def v2filter_auth(auth: IRAuth):
    assert auth.cluster
    cluster = typecast(IRCluster, auth.cluster)
    
    assert auth.api_version
    if auth.api_version == "ambassador/v0":
        # This preserves almost exactly the same logic prior to ambassador/v1 implementation.
        request_headers = dict(ExtAuthRequestHeaders)

        for hdr in auth.allowed_headers:
            request_headers[hdr] = True

        # Always allow the default set, above. This may be a slight behavior change from the
        # v0 config, but it seems to aid usability.

        hdrs = set(auth.allowed_headers or [])      # turn list into a set
        hdrs.update(AllowedAuthorizationHeaders)    # merge in a frozenset

        allowed_authorization_headers = sorted(hdrs)    # sorted() turns the set back into a list

        allowed_request_headers = sorted(request_headers.keys())

        return {
            'name': 'envoy.ext_authz',
            'config': {
                'http_service': {
                    'server_uri': {
                        'uri': 'http://%s' % auth.auth_service,
                        'cluster': cluster.name,
                        'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                    },
                    'path_prefix': auth.path_prefix,
                    'allowed_authorization_headers': allowed_authorization_headers,
                    'allowed_request_headers': allowed_request_headers,
                },
                'send_request_data': auth.allow_request_body
            }        
        }
    
    if auth.api_version == "ambassador/v1":
        assert auth.proto
        if auth.proto == "http":
            allowed_authorization_headers = list(set(auth.allowed_authorization_headers).union(AllowedAuthorizationHeaders))
            allowed_request_headers = list(set(auth.allowed_request_headers).union(AllowedRequestHeaders))

            return {
                'name': 'envoy.ext_authz',
                'config': {
                    'http_service': {
                        'server_uri': {
                            'uri': 'http://%s' % auth.auth_service,
                            'cluster': cluster.name,
                            'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                        },
                        'path_prefix': auth.path_prefix,
                        'allowed_authorization_headers': allowed_authorization_headers,
                        'allowed_request_headers': allowed_request_headers,
                    },
                    'send_request_data': auth.allow_request_body
                }        
            }

        if auth.proto == "grpc":
            return {
                'name': 'envoy.ext_authz',
                'config': {
                    'grpc_service': {
                        'envoy_grpc': {
                            'cluster_name': cluster.name
                        },
                        'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                    },
                    'send_request_data': auth.allow_request_body
                }        
            }


@v2filter.when("IRRateLimit")
def v2filter_ratelimit(ratelimit: IRRateLimit):
    config = dict(ratelimit.config)

    if 'timeout_ms' in config:
        tm_ms = config.pop('timeout_ms')

        config['timeout'] = "%0.3fs" % (float(tm_ms) / 1000.0)

    return {
        'name': 'envoy.rate_limit',
        'config': config
    }


@v2filter.when("ir.cors")
def v2filter_cors(cors: IRCORS):
    del cors    # silence unused-variable warning

    return { 'name': 'envoy.cors' }


@v2filter.when("ir.router")
def v2filter_router(router: IRFilter):
    od: Dict[str, Any] = { 'name': 'envoy.router' }

    if router.ir.tracing:
        od['config'] = { 'start_child_span': True }

    return od


class V2Listener(dict):
    def __init__(self, config: 'V2Config', listener: IRListener) -> None:
        super().__init__()

        # Default some things to the way they should be for the redirect listener
        self.name = "redirect_listener"
        self.access_log: Optional[List[dict]] = None
        self.require_tls: Optional[str] = 'ALL'
        self.use_proxy_proto: Optional[bool] = None

        self.http_filters: List[dict] = []
        self.listener_filters: List[dict] = []
        self.filter_chains: List[dict] = []

        self.upgrade_configs: Optional[List[dict]] = None

        self.routes: List[dict] = [ {
            'match': {
                'prefix': '/',
            },
            'redirect': {
                'https_redirect': True,
                'path_redirect': '/'
            }
        } ]

        if listener.redirect_listener:
            self.http_filters = [{'name': 'envoy.router'}]
        else:
            # Use the actual listener name
            self.name = listener.name

            # Use a sane access log spec
            self.access_log = [ {
                'name': 'envoy.file_access_log',
                'config': {
                    'path': '/dev/fd/1',
                    'format': 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"\n'
                }
            } ]

            # # If we have a server context here, use it.
            # envoy_ctx = V2TLSContext()
            # for name, ctx in config.ir.envoy_tls.items():
            #     config.ir.logger.info("envoy_ctx adding %s" % ctx.as_json())
            #     envoy_ctx.add_context(ctx)
            #
            # config.ir.logger.info("envoy_ctx final %s" % envoy_ctx)

            # Assemble filters
            for f in config.ir.filters:
                v2f: dict = v2filter(f)

                if v2f:
                    self.http_filters.append(v2f)

            # Grab routes from the config (we do this as a shallow copy).
            self.routes = [ dict(r) for r in config.routes ]

            # Don't require TLS.
            self.require_tls = None

            # Use the actual get_proxy_proto setting
            self.use_proxy_proto = listener.get('use_proxy_proto')

            # Save upgrade configs.
            for group in config.ir.ordered_groups():
                if group.get('use_websocket'):
                    self.upgrade_configs = [{ 'upgrade_type': 'websocket' }]
                    break

            # Let self.handle_sni do the heavy lifting for SNI.
            self.handle_sni(config)

        # If the filter chain is empty here, we had no contexts. Add a single empty element to
        # to filter chain to make the logic below a bit simpler.
        if not self.filter_chains:
            self.filter_chains.append({
                'routes': self.routes
            })

        # OK. Build our base HTTP config...
        base_http_config: Dict[str, Any] = {
            'stat_prefix': 'ingress_http',
            'access_log': self.access_log,
            'http_filters': self.http_filters,
        }

        if self.upgrade_configs:
            base_http_config['upgrade_configs'] = self.upgrade_configs

        if 'use_remote_address' in config.ir.ambassador_module:
            base_http_config["use_remote_address"] = config.ir.ambassador_module.use_remote_address

        if config.ir.tracing:
            base_http_config["generate_request_id"] = True

            base_http_config["tracing"] = {
                "operation_name": "egress"
            }

            req_hdrs = config.ir.tracing.get('tag_headers', [])

            if req_hdrs:
                base_http_config["tracing"]["request_headers_for_tags"] = req_hdrs

        # OK. For each entry in our filter chain, we need to set up the rest of the
        # config.

        for chain in self.filter_chains:
            vhost = {
                'name': 'backend',
                'domains': [ '*' ],
                'routes': chain.pop('routes')
            }

            if self.require_tls:
                vhost['require_tls'] = self.require_tls

            http_config = dict(base_http_config)    # Shallow copy is enough.

            http_config['route_config'] = {
                'virtual_hosts': [ vhost ]
            }

            chain['filters'] = [
                {
                    'name': 'envoy.http_connection_manager',
                    'config': http_config
                }
            ]

            if self.use_proxy_proto is not None:
                chain['use_proxy_proto'] = self.use_proxy_proto

        self.update({
            'name': self.name,
            'address': {
                'socket_address': {
                    'address': '0.0.0.0',
                    'port_value': listener.service_port,
                    'protocol': 'TCP'
                }
            },
            'filter_chains': self.filter_chains
        })

        if self.listener_filters:
            self['listener_filters'] = self.listener_filters

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.listeners = []

        for irlistener in config.ir.listeners:
            listener = config.save_element('listener', irlistener, V2Listener(config, irlistener))
            config.listeners.append(listener)

    def handle_sni(self, config: 'V2Config') -> None:
        """
        Manage filter chains, etc., for SNI.

        :param config: the V2Config within which we're working
        """

        # Is SNI active?
        global_sni = False

        # We'll assemble a list of active TLS contexts here. It may end up empty,
        # of course.
        envoy_contexts: List[Tuple[str, List[str], V2TLSContext]] = []

        for tls_context in config.ir.get_tls_contexts():
            if tls_context.get('hosts', None):
                config.ir.logger.debug("V2Listener: SNI operating on termination context '%s'" % tls_context.name)
                config.ir.logger.debug(tls_context.as_json())
                v2ctx = V2TLSContext(tls_context)
                config.ir.logger.debug(json.dumps(v2ctx, indent=4, sort_keys=True))
                envoy_contexts.append((tls_context.name, tls_context.hosts, v2ctx))
            else:
                config.ir.logger.debug("V2Listener: SNI skipping origination context '%s'" % tls_context.name)

        # OK. If we have multiple contexts here, SNI is likely a thing.
        if len(envoy_contexts) > 1:
            config.ir.logger.debug("V2Listener: enabling SNI, %d contexts" % len(envoy_contexts))
            config.ir.logger.debug(json.dumps(envoy_contexts, indent=4, sort_keys=True))

            global_sni = True

            self.listener_filters.append({
                'name': 'envoy.listener.tls_inspector',
                'config': {}
            })

        for name, hosts, ctx in envoy_contexts:
            config.ir.logger.info("V2Listener: SNI route check %s, %s, %s" % (name, hosts, json.dumps(ctx, indent=4, sort_keys=True)))

            chain = {
                'tls_context': ctx,
                'routes': list(self.routes)
            }

            if global_sni:
                chain['filter_chain_match'] = {
                    'server_names': hosts
                }

            for sni_route in config.sni_routes:
                # Check if filter chain and SNI route have matching hosts
                config.ir.logger.info("V2Listener: SNI route check %s, route %s" % (name, json.dumps(sni_route, indent=4, sort_keys=True)))
                matched = sorted(sni_route['info']['hosts']) == sorted(hosts)

                # Check for certificate match too.
                for sni_key, ctx_key in [ ('cert_chain_file', 'certificate_chain'),
                                          ('private_key_file', 'private_key') ]:
                    sni_value = sni_route['info']['secret_info'][sni_key]
                    ctx_value = ctx['common_tls_context']['tls_certificates'][0][ctx_key]['filename']   # XXX ugh. Multiple certs?

                    if sni_value != ctx_value:
                        matched = False
                        break

                if matched:
                    chain['routes'].append(sni_route['route'])

            self.filter_chains.append(chain)
