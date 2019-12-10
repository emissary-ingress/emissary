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
from ...ir.irgzip import IRGzip
from ...ir.irfilter import IRFilter
from ...ir.irratelimit import IRRateLimit
from ...ir.ircors import IRCORS
from ...ir.ircluster import IRCluster
from ...ir.irtcpmappinggroup import IRTCPMappingGroup

from ambassador.utils import ParsedService as Service

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

def header_pattern_key(x: Dict[str, str]) -> List[Tuple[str, str]]:
    return sorted([ (k, v) for k, v in x.items() ])

@multi
def v2filter(irfilter: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    if irfilter.kind == 'IRAuth':
        if irfilter.api_version == 'getambassador.io/v0':
            return 'IRAuth_v0'
        elif (irfilter.api_version == 'getambassador.io/v1') or (irfilter.api_version == 'getambassador.io/v2'):
            return 'IRAuth_v1-2'
        else:
            irfilter.post_error('AuthService version %s unknown, treating as v2' % irfilter.api_version)
            return 'IRAuth_v1-2'
    else:
        return irfilter.kind

@v2filter.when("IRBuffer")
def v2filter_buffer(buffer: IRBuffer, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.buffer',
        'config': {
            "max_request_bytes": buffer.max_request_bytes
        }
    }

@v2filter.when("IRGzip")
def v2filter_gzip(gzip: IRGzip, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.gzip',
        'config': {
            'memory_level': gzip.memory_level,
            'content_length': gzip.content_length,
            'compression_level': gzip.compression_level,
            'compression_strategy': gzip.compression_strategy,
            'window_bits': gzip.window_bits,
            'content_type': gzip.content_type,
            'disable_on_etag_header': gzip.disable_on_etag_header,
            'remove_accept_encoding_header': gzip.remove_accept_encoding_header,
        }
    }

@v2filter.when("ir.grpc_http1_bridge")
def v2filter_grpc_http1_bridge(irfilter: IRFilter, v2config: 'V2Config'):
    del irfilter  # silence unused-variable warning
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.grpc_http1_bridge',
        'config': {},
    }

@v2filter.when("ir.grpc_web")
def v2filter_grpc_web(irfilter: IRFilter, v2config: 'V2Config'):
    del irfilter  # silence unused-variable warning
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.grpc_web',
        'config': {},
    }

def auth_cluster_uri(auth: IRAuth, cluster: IRCluster) -> str:
    cluster_context = cluster.get('tls_context')
    scheme = 'https' if cluster_context else 'http'

    prefix = auth.get("path_prefix") or ""

    if prefix.startswith("/"):
        prefix = prefix[1:]

    server_uri = "%s://%s" % (scheme, prefix)

    auth.ir.logger.info("%s: server_uri %s" % (auth.name, server_uri))

    return server_uri

@v2filter.when("IRAuth_v0")
def v2filter_authv0(auth: IRAuth, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    assert auth.cluster
    cluster = typecast(IRCluster, auth.cluster)

    assert auth.api_version == "getambassador.io/v0"

    # This preserves almost exactly the same logic prior to ambassador/v1 implementation.
    request_headers = dict(ExtAuthRequestHeaders)

    for hdr in auth.allowed_headers:
        request_headers[hdr] = True

    # Always allow the default set, above. This may be a slight behavior change from the
    # v0 config, but it seems to aid usability.

    hdrs = set(auth.allowed_headers or [])      # turn list into a set
    hdrs.update(AllowedAuthorizationHeaders)    # merge in a frozenset

    allowed_authorization_headers = []

    for key in sorted(hdrs):
        allowed_authorization_headers.append({"exact": key})

    allowed_request_headers = []

    for key in sorted(request_headers.keys()):
        allowed_request_headers.append({"exact": key})

    return {
        'name': 'envoy.ext_authz',
        'config': {
            'http_service': {
                'server_uri': {
                    'uri': auth_cluster_uri(auth, cluster),
                    'cluster': cluster.name,
                    'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                },
                'path_prefix': auth.path_prefix,
                'authorization_request': {
                    'allowed_headers': {
                        'patterns': sorted(allowed_request_headers, key=header_pattern_key)
                    }
                },
                'authorization_response' : {
                    'allowed_upstream_headers': {
                        'patterns': sorted(allowed_authorization_headers, key=header_pattern_key)
                    },
                    'allowed_client_headers': {
                        'patterns': sorted(allowed_authorization_headers, key=header_pattern_key)
                    }
                }
            }
        }
    }


@v2filter.when("IRAuth_v1-2")
def v2filter_authv1(auth: IRAuth, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    assert auth.cluster
    cluster = typecast(IRCluster, auth.cluster)

    if (auth.api_version != "getambassador.io/v1") and (auth.api_version != "getambassador.io/v2"):
        auth.ir.logger.warning("IRAuth_v1 working on %s, mismatched at %s" % (auth.name, auth.api_version))

    assert auth.proto

    raw_body_info: Optional[Dict[str, int]] = auth.get('include_body')

    if not raw_body_info and auth.get('allow_request_body', False):
        raw_body_info = {
            'max_bytes': 4096,
            'allow_partial': True
        }

    body_info: Optional[Dict[str, int]] = None

    if raw_body_info:
        body_info = {}

        if 'max_bytes' in raw_body_info:
            body_info['max_request_bytes'] = raw_body_info['max_bytes']

        if 'allow_partial' in raw_body_info:
            body_info['allow_partial_message'] = raw_body_info['allow_partial']

    auth_info: Dict[str, Any] = {}

    if auth.proto == "http":
        allowed_authorization_headers = []
        headers_to_add = []

        for key in list(set(auth.allowed_authorization_headers).union(AllowedAuthorizationHeaders)):
            allowed_authorization_headers.append({"exact": key})

        allowed_request_headers = []

        for key in list(set(auth.allowed_request_headers).union(AllowedRequestHeaders)):
            allowed_request_headers.append({"exact": key})

        if auth.get('add_linkerd_headers', False):
            svc = Service(auth.ir.logger, auth_cluster_uri(auth, cluster))
            headers_to_add.append({
                'key' : 'l5d-dst-override',
                'value': svc.hostname_port
            })

        auth_info = {
            'name': 'envoy.ext_authz',
            'config': {
                'http_service': {
                    'server_uri': {
                        'uri': auth_cluster_uri(auth, cluster),
                        'cluster': cluster.name,
                        'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                    },
                    'path_prefix': auth.path_prefix,
                    'authorization_request': {
                        'allowed_headers': {
                            'patterns': sorted(allowed_request_headers, key=header_pattern_key)
                        },
                        'headers_to_add' : headers_to_add
                    },
                    'authorization_response' : {
                        'allowed_upstream_headers': {
                            'patterns': sorted(allowed_authorization_headers, key=header_pattern_key)
                        },
                        'allowed_client_headers': {
                            'patterns': sorted(allowed_authorization_headers, key=header_pattern_key)
                        }
                    }
                },
            }
        }

    if auth.proto == "grpc":
        auth_info = {
            'name': 'envoy.ext_authz',
            'config': {
                'grpc_service': {
                    'envoy_grpc': {
                        'cluster_name': cluster.name
                    },
                    'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                },
                'use_alpha': True
            }
        }

    if auth_info:
        if body_info:
            auth_info['config']['with_request_body'] = body_info

        if 'failure_mode_allow' in auth:
            auth_info['config']["failure_mode_allow"] = auth.failure_mode_allow

        if 'status_on_error' in auth:
            status_on_error: Optional[Dict[str, int]] = auth.get('status_on_error')
            auth_info['config']["status_on_error"] = status_on_error

        return auth_info

    # If here, something's gone horribly wrong.
    auth.post_error("Protocol '%s' is not supported, auth not enabled" % auth.proto)
    return None


@v2filter.when("IRRateLimit")
def v2filter_ratelimit(ratelimit: IRRateLimit, v2config: 'V2Config'):
    config = dict(ratelimit.config)

    if 'timeout_ms' in config:
        tm_ms = config.pop('timeout_ms')

        config['timeout'] = "%0.3fs" % (float(tm_ms) / 1000.0)

    # If here, we must have a ratelimit service configured.
    assert v2config.ratelimit
    config['rate_limit_service'] = dict(v2config.ratelimit)

    return {
        'name': 'envoy.rate_limit',
        'config': config,
    }


@v2filter.when("ir.cors")
def v2filter_cors(cors: IRCORS, v2config: 'V2Config'):
    del cors    # silence unused-variable warning
    del v2config  # silence unused-variable warning

    return { 'name': 'envoy.cors' }


@v2filter.when("ir.router")
def v2filter_router(router: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    od: Dict[str, Any] = { 'name': 'envoy.router' }

    if router.ir.tracing:
        od['config'] = { 'start_child_span': True }

    return od


@v2filter.when("ir.lua_scripts")
def v2filter_lua(irfilter: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.lua',
        'config': irfilter.config_dict(),
    }


class V2TCPListener(dict):
    def __init__(self, config: 'V2Config', group: IRTCPMappingGroup) -> None:
        super().__init__()

        # Use the actual listener name & port number
        self.bind_address = group.get('address') or '0.0.0.0'
        self.name = "listener-%s-%s" % (self.bind_address, group.port)

        self.tls_context: Optional[V2TLSContext] = None

        # Set the basics like our name and listening address.
        self.update({
            'name': self.name,
            'address': {
                'socket_address': {
                    'address': self.bind_address,
                    'port_value': group.port,
                    'protocol': 'TCP'
                }
            },
            'filter_chains': []
        })

        # Next: is SNI a thing?
        if group.get('tls_context', None):
            # Yup. We need the TLS inspector here...
            self['listener_filters'] = [ {
                'name': 'envoy.listener.tls_inspector',
                'config': {}
            } ]

            # ...and we need to save the TLS context we'll be using.
            self.tls_context = V2TLSContext(group.tls_context)

    def add_group(self, config: 'V2Config', group: IRTCPMappingGroup) -> None:
        # First up, which clusters do we need to talk to?
        clusters = [{
            'name': mapping.cluster.name,
            'weight': mapping.weight
        } for mapping in group.mappings]

        # From that, we can sort out a basic tcp_proxy filter config.
        tcp_filter = {
            'name': 'envoy.tcp_proxy',
            'config': {
                'stat_prefix': 'ingress_tcp_%d' % group.port,
                'weighted_clusters': {
                    'clusters': clusters
                }
            }
        }

        # OK. Basic filter chain entry next.
        chain_entry: Dict[str, Any] = {
            'filters': [
                tcp_filter
            ]
        }

        # Then, if SNI is a thing, update the chain entry with the appropriate chain match.
        if self.tls_context:
            # Apply the context to the chain...
            chain_entry['tls_context'] = self.tls_context

            # Do we have a host match?
            host_wanted = group.get('host') or '*'

            if host_wanted != '*':
                # Yup. Hook it in.
                chain_entry['filter_chain_match'] = {
                    'server_names': [ host_wanted ]
                }

        # OK, once that's done, stick this into our filter chains.
        self['filter_chains'].append(chain_entry)


class V2Listener(dict):
    def __init__(self, config: 'V2Config', listener: IRListener) -> None:
        super().__init__()

        # Default some things to the way they should be for the redirect listener
        self.name = "redirect_listener"
        self.access_log: List[dict] = []
        self.require_tls: Optional[str] = 'EXTERNAL_ONLY'
        self.use_proxy_proto = listener.get('use_proxy_proto')

        self.http_filters: List[dict] = []
        self.listener_filters: List[dict] = []
        self.filter_chains: List[dict] = []
        self.need_tls_inspector = False

        self.upgrade_configs: Optional[List[dict]] = None

        self.routes: List[dict] = [ {
                'match': {
                    'prefix': '/',
                },
                'redirect': {
                    'https_redirect': True
                }
            } ]

        # Get Access Log Rules
        for al in config.ir.log_services.values():
            access_log_obj = { "common_config": al.get_common_config() }
            req_headers = []
            resp_headers = []
            trailer_headers = []

            for additional_header in al.get_additional_headers():
                if additional_header.get('during_request', True):
                    req_headers.append(additional_header.get('header_name'))
                if additional_header.get('during_response', True):
                    resp_headers.append(additional_header.get('header_name'))
                if additional_header.get('during_trailer', True):
                    trailer_headers.append(additional_header.get('header_name'))

            if al.driver == 'http':
                access_log_obj['additional_request_headers_to_log'] = req_headers
                access_log_obj['additional_response_headers_to_log'] = resp_headers
                access_log_obj['additional_response_trailers_to_log'] = trailer_headers
                self.access_log.append({"name": "envoy.http_grpc_access_log", "config": access_log_obj})
            else:
                # inherently TCP right now
                # tcp loggers do not support additional headers
                self.access_log.append({"name": "envoy.tcp_grpc_access_log", "config": access_log_obj})


        if listener.redirect_listener:
            self.http_filters = [{'name': 'envoy.router'}]
        else:
            # Use the actual listener name & port number
            self.name = "ambassador-listener-%s" % listener.service_port

            # Use sane access log spec in JSON
            if(config.ir.ambassador_module.envoy_log_type.lower() == "json") :
                json_format = {
                    'start_time': '%START_TIME%',
                    'method': '%REQ(:METHOD)%',
                    'path': '%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%',
                    'protocol': '%PROTOCOL%',
                    'response_code': '%RESPONSE_CODE%',
                    'response_flags': '%RESPONSE_FLAGS%',
                    'bytes_received': '%BYTES_RECEIVED%',
                    'bytes_sent': '%BYTES_SENT%',
                    'duration': '%DURATION%',
                    'upstream_service_time': '%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%',
                    'x_forwarded_for': '%REQ(X-FORWARDED-FOR)%',
                    'user_agent': '%REQ(USER-AGENT)%',
                    'request_id': '%REQ(X-REQUEST-ID)%',
                    'authority': '%REQ(:AUTHORITY)%',
                    'upstream_host': '%UPSTREAM_HOST%',
                    'upstream_cluster': '%UPSTREAM_CLUSTER%',
                    'upstream_local_address': '%UPSTREAM_LOCAL_ADDRESS%',
                    'downstream_local_address': '%DOWNSTREAM_LOCAL_ADDRESS%',
                    'downstream_remote_address': '%DOWNSTREAM_REMOTE_ADDRESS%',
                    'requested_server_name': '%REQUESTED_SERVER_NAME%',
                    'istio_policy_status': '%DYNAMIC_METADATA(istio.mixer:status)%',
                    'upstream_transport_failure_reason': '%UPSTREAM_TRANSPORT_FAILURE_REASON%'
                }

                tracing_config = config.ir.tracing
                if tracing_config and tracing_config.driver == 'envoy.tracers.datadog':
                    json_format['dd.trace_id'] = '%REQ(X-DATADOG-TRACE-ID)%'
                    json_format['dd.span_id'] = '%REQ(X-DATADOG-PARENT-ID)%'

                self.access_log.append({
                    'name': 'envoy.file_access_log',
                    'config': {
                        'path': config.ir.ambassador_module.envoy_log_path,
                        'json_format': json_format
                    }
                })
            else:
                # Use a sane access log spec
                log_format = config.ir.ambassador_module.get('envoy_log_format', None)

                if not log_format:
                    log_format = 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"'

                config.ir.logger.info("V2Listener: Using log_format '%s'" % log_format)
                self.access_log.append({
                    'name': 'envoy.file_access_log',
                    'config': {
                        'path': config.ir.ambassador_module.envoy_log_path,
                        'format': log_format + '\n'
                    }
                })

            # Assemble filters
            for f in config.ir.filters:
                v2f: dict = v2filter(f, config)

                if v2f:
                    self.http_filters.append(v2f)

            # Grab routes from the config (we do this as a shallow copy).
            self.routes = [ dict(r) for r in config.routes ]

            # Don't require TLS.
            if not listener.require_tls:
                self.require_tls = None

            # Save upgrade configs.
            for group in config.ir.ordered_groups():
                if group.get('use_websocket'):
                    self.upgrade_configs = [{ 'upgrade_type': 'websocket' }]
                    break

            # Let self.handle_sni do the heavy lifting for SNI.
            self.handle_sni(config)

        # We need to add a cleartext listener if any of the following are true:
        #
        # 1. We don't have any termination contexts.
        # 2. We have a Host that explicitly says "acme-provider: none".
        # 3. We are allowing the fallback UI route.

        need_cleartext = False

        if not self.filter_chains:
            config.ir.logger.info("V2L: no filter chains, need cleartext")
            need_cleartext = True

        if config.ir.wizard_allowed:
            config.ir.logger.info("V2L: wizard allowed, need cleartext")
            need_cleartext = True

        host_dict = config.ir.aconf.get_config("hosts") or {}
    
        for host in host_dict.values():
            if host.get('acme-provider', 'zzz').lower() == 'none':
                config.ir.logger.info(f"V2L: host {host.hostname} has ACME none, need cleartext")
                need_cleartext = True
                break

        if need_cleartext:
            # By definition, this chain has no TLS contexts.
            cleartext_chain = {
                'routes': self.routes,
                '_ctx_name': '-cleartext-',
                '_ctx_hosts': ['*']
            }

            if self.need_tls_inspector:
                cleartext_chain['filter_chain_match'] = {}

            self.filter_chains.append(cleartext_chain)

            self.dump_chains(config)

        # Set up the TLS inspector if we need it.
        if self.need_tls_inspector:
            config.ir.logger.info("V2L: enabling TLS inspector")

            self.listener_filters.append({
                'name': 'envoy.listener.tls_inspector',
                'config': {}
            })
        else:
            config.ir.logger.info("V2L: leaving TLS inspector disabled")

        # Clean up our filter chains...
        for chain in self.filter_chains:
            chain.pop('_ctx_name', None)
            chain.pop('_ctx_hosts', None)

        # OK. Build our base HTTP config...
        base_http_config: Dict[str, Any] = {
            'stat_prefix': 'ingress_http',
            'access_log': self.access_log,
            'http_filters': self.http_filters,
            'normalize_path': True
        }

        if self.upgrade_configs:
            base_http_config['upgrade_configs'] = self.upgrade_configs

        if 'use_remote_address' in config.ir.ambassador_module:
            base_http_config["use_remote_address"] = config.ir.ambassador_module.use_remote_address

        if 'xff_num_trusted_hops' in config.ir.ambassador_module:
            base_http_config["xff_num_trusted_hops"] = config.ir.ambassador_module.xff_num_trusted_hops

        if 'server_name' in config.ir.ambassador_module:
            base_http_config["server_name"] = config.ir.ambassador_module.server_name

        if 'enable_http10' in config.ir.ambassador_module:
            base_http_config["http_protocol_options"] = { 'accept_http_10': config.ir.ambassador_module.enable_http10 }

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

    def handle_sni(self, config: 'V2Config') -> None:
        """
        Manage filter chains, etc., for SNI.

        :param config: the V2Config within which we're working
        """

        # Is SNI active?
        global_sni = False

        # We'll assemble a list of active TLS contexts here. It may end up empty,
        # of course.
        envoy_contexts: List[Tuple[str, Optional[List[str]], V2TLSContext]] = []

        for tls_context in config.ir.get_tls_contexts():
            if tls_context.get('hosts', None):
                config.ir.logger.debug("V2Listener: SNI taking termination context '%s'" % tls_context.name)
                config.ir.logger.debug(tls_context.as_json())
                v2ctx = V2TLSContext(tls_context)
                # config.ir.logger.debug(json.dumps(v2ctx, indent=4, sort_keys=True))
                envoy_contexts.append((tls_context.name, tls_context.hosts, v2ctx))
            else:
                config.ir.logger.debug("V2Listener: SNI skipping origination context '%s'" % tls_context.name)

        # OK. If we have multiple contexts here, SNI is likely a thing.
        if len(envoy_contexts) > 1:
            config.ir.logger.debug("V2Listener: enabling SNI, %d contexts" % len(envoy_contexts))
            config.ir.logger.debug("            [ %s ]" % ", ".join([ x[0] for x in envoy_contexts ]))

            global_sni = True
        else:
            config.ir.logger.debug("V2Listener: disabling SNI, %d contexts" % len(envoy_contexts))

        for name, hosts, ctx in envoy_contexts:
            if not ctx:
                continue

            config.ir.logger.info(f"V2Listener: SNI (1) route check {name} - {hosts}")

            routes = list(self.routes)

            chain: Dict[str, Any] = {
                'tls_context': ctx,
                '_ctx_name': name,
                '_ctx_hosts': hosts
            }

            # We have a TLS context, so we should make sure they're speaking TLS!
            self.need_tls_inspector = True
            filter_chain_match: Dict[str, Any] = {
                'transport_protocol': 'tls'
            }

            if global_sni and (hosts != [ '*' ]):
                filter_chain_match['server_names'] = hosts

            chain['filter_chain_match'] = filter_chain_match

            for sni_route in config.sni_routes:
                # Check if filter chain and SNI route have matching hosts
                context_hosts = sorted(hosts or [])
                route_hosts = sorted(sni_route['info']['hosts'])
                matched = (route_hosts == context_hosts)

                # Check for certificate match too.
                for sni_key, ctx_key in [ ('cert_chain_file', 'certificate_chain'),
                                          ('private_key_file', 'private_key') ]:
                    sni_value = sni_route['info']['secret_info'][sni_key]
                    # XXX ugh. Multiple certs?
                    ctx_value = ctx['common_tls_context']['tls_certificates'][0][ctx_key]['filename']

                    if sni_value != ctx_value:
                        matched = False
                        break

                config.ir.logger.info("V2Listener:   SNI (2) route check %s %s route for %s" %
                                      (name, "TAKE" if matched else "SKIP", route_hosts))

                if matched:
                    routes.append(sni_route['route'])

            chain['routes'] = routes
            self.filter_chains.append(chain)

        self.dump_chains(config)

    def dump_chains(self, config):
        dumpinfo = []
        for chain in self.filter_chains:
            di = {
                'filter_chain_match': chain.get('filter_chain_match') or {},
                'route_count': len(chain['routes']),
                'ctx_name': chain['_ctx_name'],
                'ctx_hosts': chain['_ctx_hosts']
            }

            dumpinfo.append(di)
        config.ir.logger.info("V2Listener: SNI filter chains\n%s" %
                              json.dumps(dumpinfo, indent=4, sort_keys=True))

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.listeners = []

        for irlistener in config.ir.listeners:
            listener = config.save_element('listener', irlistener, V2Listener(config, irlistener))
            config.listeners.append(listener)

        # We need listeners for the TCPMappingGroups too.
        tcplisteners: Dict[str, V2TCPListener] = {}

        for irgroup in config.ir.ordered_groups():
            if not isinstance(irgroup, IRTCPMappingGroup):
                continue

            # OK, good to go. Do we already have a TCP listener binding where this one does?
            group_key = irgroup.bind_to()
            listener = tcplisteners.get(group_key, None)

            config.ir.logger.info("V2TCPListener: group at %s found %s listener" %
                                  (group_key, "extant" if listener else "no"))

            if not listener:
                # Nope. Make a new one and save it.
                listener = config.save_element('listener', irgroup, V2TCPListener(config, irgroup))
                config.listeners.append(listener)
                tcplisteners[group_key] = listener

            # Whether we just created this listener or not, add this irgroup to it.
            listener.add_group(config, irgroup)
