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
from typing import Any, Dict, List, Optional, Tuple, Union, TYPE_CHECKING
from typing import cast as typecast

from os import environ

import json
import logging

from multi import multi
from ...ir.irlistener import IRListener
from ...ir.irauth import IRAuth
from ...ir.irerrorresponse import IRErrorResponse
from ...ir.irbuffer import IRBuffer
from ...ir.irgzip import IRGzip
from ...ir.irfilter import IRFilter
from ...ir.irratelimit import IRRateLimit
from ...ir.ircors import IRCORS
from ...ir.ircluster import IRCluster
from ...ir.irtcpmappinggroup import IRTCPMappingGroup
from ...ir.irtlscontext import IRTLSContext

from ...utils import dump_json, parse_bool
from ...utils import ParsedService as Service

from .v2route import V2Route
from .v2tls import V2TLSContext

if TYPE_CHECKING:
    from . import V2Config # pragma: no cover

DictifiedV2Route = Dict[str, Any]

EnvoyCTXInfo = Tuple[str, Optional[List[str]], V2TLSContext]

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


def jsonify(x) -> str:
    return dump_json(x, pretty=True)


def prettyroute(route: DictifiedV2Route) -> str:
    match = route["match"]

    key = "PFX"
    value = match.get("prefix", None)

    if not value:
        key = "SRX"
        value = match.get("safe_regex", {}).get("regex", None)

    if not value:
        key = "URX"
        value = match.get("unsafe_regex", None)

    if not value:
        key = "???"
        value = "-none-"

    match_str = f"{key} {value}"

    headers = match.get("headers", {})
    xfp = None
    host = None

    for header in headers:
        name = header.get("name", None).lower()
        exact = header.get("exact_match", None)

        if not name or not exact:
            continue

        if name == "x-forwarded-proto":
            xfp = bool(exact == "https")
        elif name == ":authority":
            host = exact

    match_str += f" {'IN' if not xfp else ''}SECURE"

    if host:
        match_str += f" HOST {host}"

    target_str = "-none-"

    if route.get("route"):
        target_str = f"ROUTE {route['route']['cluster']}"
    elif route.get("redirect"):
        target_str = f"REDIRECT"

    return f"<V2Route {match_str} -> {target_str}>"


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
        'name': 'envoy.filters.http.buffer',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.config.filter.http.buffer.v2.Buffer',
            "max_request_bytes": buffer.max_request_bytes
        }
    }

@v2filter.when("IRGzip")
def v2filter_gzip(gzip: IRGzip, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.gzip',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.config.filter.http.gzip.v2.Gzip',
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
        'name': 'envoy.filters.http.grpc_http1_bridge'
    }

@v2filter.when("ir.grpc_web")
def v2filter_grpc_web(irfilter: IRFilter, v2config: 'V2Config'):
    del irfilter  # silence unused-variable warning
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.grpc_web'
    }

@v2filter.when("ir.grpc_stats")
def v2filter_grpc_stats(irfilter: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.grpc_stats',
        'config': irfilter.config_dict(),
    }

def auth_cluster_uri(auth: IRAuth, cluster: IRCluster) -> str:
    cluster_context = cluster.get('tls_context')
    scheme = 'https' if cluster_context else 'http'

    prefix = auth.get("path_prefix") or ""

    if prefix.startswith("/"):
        prefix = prefix[1:]

    server_uri = "%s://%s" % (scheme, prefix)

    if auth.ir.logger.isEnabledFor(logging.DEBUG):
        auth.ir.logger.debug("%s: server_uri %s" % (auth.name, server_uri))

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
        allowed_authorization_headers.append({"exact": key, "ignore_case": True})

    allowed_request_headers = []

    for key in sorted(request_headers.keys()):
        allowed_request_headers.append({"exact": key, "ignore_case": True})

    return {
        'name': 'envoy.filters.http.ext_authz',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthz',
            'http_service': {
                'server_uri': {
                    'uri': auth_cluster_uri(auth, cluster),
                    'cluster': cluster.envoy_name,
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

        for k, v in auth.get('add_auth_headers', {}).items():
            headers_to_add.append({
                'key': k,
                'value': v,
            })

        for key in list(set(auth.allowed_authorization_headers).union(AllowedAuthorizationHeaders)):
            allowed_authorization_headers.append({"exact": key, "ignore_case": True})

        allowed_request_headers = []

        for key in list(set(auth.allowed_request_headers).union(AllowedRequestHeaders)):
            allowed_request_headers.append({"exact": key, "ignore_case": True})

        if auth.get('add_linkerd_headers', False):
            svc = Service(auth.ir.logger, auth_cluster_uri(auth, cluster))
            headers_to_add.append({
                'key' : 'l5d-dst-override',
                'value': svc.hostname_port
            })

        auth_info = {
            'name': 'envoy.filters.http.ext_authz',
            'typed_config': {
                '@type': 'type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthz',
                'http_service': {
                    'server_uri': {
                        'uri': auth_cluster_uri(auth, cluster),
                        'cluster': cluster.envoy_name,
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
        protocol_version = auth.get('protocol_version', 'v2')
        auth_info = {
            'name': 'envoy.filters.http.ext_authz',
            'typed_config': {
                '@type': 'type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthz',
                'grpc_service': {
                    'envoy_grpc': {
                        'cluster_name': cluster.envoy_name
                    },
                    'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                }
            }
        }

    if auth_info:
        auth_info['typed_config']['clear_route_cache'] = True

        if body_info:
            auth_info['typed_config']['with_request_body'] = body_info

        if 'failure_mode_allow' in auth:
            auth_info['typed_config']["failure_mode_allow"] = auth.failure_mode_allow

        if 'status_on_error' in auth:
            status_on_error: Optional[Dict[str, int]] = auth.get('status_on_error')
            auth_info['typed_config']["status_on_error"] = status_on_error

        return auth_info

    # If here, something's gone horribly wrong.
    auth.post_error("Protocol '%s' is not supported, auth not enabled" % auth.proto)
    return None


# Careful: this function returns None to indicate that no Envoy response_map
# filter needs to be instantiated, because either no Module nor Mapping
# has error_response_overrides, or the ones that exist are not valid.
#
# By not instantiating the filter in those cases, we prevent adding a useless
# filter onto the chain.
@v2filter.when("IRErrorResponse")
def v2filter_error_response(error_response: IRErrorResponse, v2config: 'V2Config'):
    # Error response configuration can come from the Ambassador module, on a
    # a Mapping, or both. We need to use the response_map filter if either one
    # of these sources defines error responses. First, check if any route
    # has per-filter config for error responses. If so, we know a Mapping has
    # defined error responses.
    route_has_error_responses = False
    for route in v2config.routes:
        typed_per_filter_config = route.get('typed_per_filter_config', {})
        if 'envoy.filters.http.response_map' in typed_per_filter_config:
            route_has_error_responses = True
            break

    filter_config: Dict[str, Any] = {
        # The IRErrorResponse filter builds on the 'envoy.filters.http.response_map' filter.
        'name': 'envoy.filters.http.response_map'
    }

    module_config = error_response.config()
    if module_config:
        # Mappers are required, otherwise this the response map has nothing to do. We really
        # shouldn't have a config with nothing in it, but we defend against this case anyway.
        if 'mappers' not in module_config or len(module_config['mappers']) == 0:
            error_response.post_error('ErrorResponse Module config has no mappers, cannot configure.')
            return None

        # If there's module config for error responses, create config for that here.
        # If not, there must be some Mapping config for it, so we'll just return
        # a filter with no global config and let the Mapping's per-route config
        # take action instead.
        filter_config['typed_config'] = {
            '@type': 'type.googleapis.com/envoy.extensions.filters.http.response_map.v3.ResponseMap',
            # The response map filter supports an array of mappers for matching as well
            # as default actions to take if there are no overrides on a mapper. We do
            # not take advantage of any default actions, and instead ensure that all of
            # the mappers we generate contain some action (eg: body_format_override).
            'mappers': module_config['mappers']
        }
        return filter_config
    elif route_has_error_responses:
        # Return the filter config as-is without global configuration. The mapping config
        # has its own per-route config and simply needs this filter to exist.
        return filter_config

    # There is no module config nor mapping config that requires the response map filter,
    # so we omit it. By returning None, the caller will omit this filter from the
    # filter chain entirely, which is not the usual way of handling filter config,
    # but it's valid.
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
    config['@type'] = 'type.googleapis.com/envoy.config.filter.http.rate_limit.v2.RateLimit'

    return {
        'name': 'envoy.filters.http.ratelimit',
        'typed_config': config,
    }


@v2filter.when("IRIPAllowDeny")
def v2filter_ipallowdeny(irfilter: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    # Go ahead and convert the irfilter to its dictionary form; it's
    # just simpler to do that once up front.

    fdict = irfilter.as_dict()

    # How many principals do we have?
    num_principals = len(fdict["principals"])
    assert num_principals > 0

    # Ew.
    SinglePrincipal = Dict[str, Dict[str, str]]
    MultiplePrincipals = Dict[str, Dict[str, List[SinglePrincipal]]]

    principals: Union[SinglePrincipal, MultiplePrincipals]

    if num_principals == 1:
        # Just one principal, so we can stuff it directly into the
        # Envoy-config principals "list".
        principals = fdict["principals"][0]
    else:
        # Multiple principals, so we have to set up an or_ids set.
        principals = {
            "or_ids": {
                "ids": fdict["principals"]
            }
        }

    return {
        "name": "envoy.filters.http.rbac",
        "typed_config": {
            "@type": "type.googleapis.com/envoy.config.filter.http.rbac.v2.RBAC",
            "rules": {
                "action": irfilter.action.upper(),
                "policies": {
                    f"ambassador-ip-{irfilter.action.lower()}": {
                        "permissions": [
                            {
                                "any": True
                            }
                        ],
                        "principals": [ principals ]
                    }
                }
            }
        }
    }


@v2filter.when("ir.cors")
def v2filter_cors(cors: IRCORS, v2config: 'V2Config'):
    del cors    # silence unused-variable warning
    del v2config  # silence unused-variable warning

    return { 'name': 'envoy.filters.http.cors' }


@v2filter.when("ir.router")
def v2filter_router(router: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    od: Dict[str, Any] = { 'name': 'envoy.filters.http.router' }

    # Use this config base if we actually need to set config fields below. We don't set
    # this on `od` by default because it would be an error to end up returning a typed
    # config that has no real config fields, only a type.
    typed_config_base = {
        '@type': 'type.googleapis.com/envoy.config.filter.http.router.v2.Router'
    }

    if router.ir.tracing:
        typed_config = od.setdefault('typed_config', typed_config_base)
        typed_config['start_child_span'] = True

    if parse_bool(router.ir.ambassador_module.get('suppress_envoy_headers', 'false')):
        typed_config = od.setdefault('typed_config', typed_config_base)
        typed_config['suppress_envoy_headers'] = True

    return od


@v2filter.when("ir.lua_scripts")
def v2filter_lua(irfilter: IRFilter, v2config: 'V2Config'):
    del v2config  # silence unused-variable warning

    config_dict = irfilter.config_dict()
    config: Dict[str, Any]
    config = {
        'name': 'envoy.filters.http.lua'
    }

    if config_dict:
        config['typed_config'] = config_dict
        config['typed_config']['@type'] = 'type.googleapis.com/envoy.config.filter.http.lua.v2.Lua'

    return config


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
                'name': 'envoy.filters.listener.tls_inspector'
            } ]

            # ...and we need to save the TLS context we'll be using.
            self.tls_context = V2TLSContext(group.tls_context)

    def add_group(self, config: 'V2Config', group: IRTCPMappingGroup) -> None:
        # First up, which clusters do we need to talk to?
        clusters = [{
            'name': mapping.cluster.envoy_name,
            'weight': mapping.weight
        } for mapping in group.mappings]

        # From that, we can sort out a basic tcp_proxy filter config.
        tcp_filter = {
            'name': 'envoy.filters.network.tcp_proxy',
            'typed_config': {
                '@type': 'type.googleapis.com/envoy.config.filter.network.tcp_proxy.v2.TcpProxy',
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


class V2VirtualHost:
    def __init__(self, config: 'V2Config', listener: 'V2Listener',
                 name: str, hostname: str, ctx: Optional[IRTLSContext],
                 secure: bool, action: Optional[str], insecure_action: Optional[str]) -> None:
        super().__init__()

        self._config = config
        self._listener = listener
        self._name = name
        self._hostname = hostname
        self._ctx = ctx
        self._secure = secure
        self._action = action
        self._insecure_action = insecure_action
        self._needs_redirect = False

        self.tls_context = V2TLSContext(ctx)
        self.routes: List[DictifiedV2Route] = []

    def needs_redirect(self) -> None:
        self._needs_redirect = True

    def finalize(self) -> None:
        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        log_debug = self._config.ir.logger.isEnabledFor(logging.DEBUG)

        # Even though this is called V2VirtualHost, we track the filter_chain_match here,
        # because it makes more sense, because this is where we have the domain information.
        # The 1:1 correspondence that this implies between filters and domains may need to
        # change later, of course...
        if log_debug:
            self._config.ir.logger.debug(f"V2VirtualHost finalize {jsonify(self.pretty())}")

        match: Dict[str, Any] = {}

        if self._ctx:
            match["transport_protocol"] = "tls"

        # Make sure we include a server name match if the hostname isn't "*".
        if self._hostname and (self._hostname != '*'):
                match["server_names"] = [ self._hostname ]

        self.filter_chain_match = match

        # If we're on Edge Stack and we're not an intercept agent, punch a hole for ACME
        # challenges, for every listener.
        if self._config.ir.edge_stack_allowed and not self._config.ir.agent_active:
            found_acme = False

            for route in self.routes:
                if route["match"].get("prefix", None) == "/.well-known/acme-challenge/":
                    found_acme = True
                    break

            if not found_acme:
                # The target cluster doesn't actually matter -- the auth service grabs the
                # challenge and does the right thing. But we do need a cluster that actually
                # exists, so use the sidecar cluster.

                if not self._config.ir.sidecar_cluster_name:
                    # Uh whut? how is Edge Stack running exactly?
                    raise Exception("Edge Stack claims to be running, but we have no sidecar cluster??")

                if log_debug:
                    self._config.ir.logger.debug(f"V2VirtualHost finalize punching a hole for ACME")

                self.routes.insert(0, {
                    "match": {
                        "case_sensitive": True,
                        "prefix": "/.well-known/acme-challenge/"
                    },
                    "route": {
                        "cluster": self._config.ir.sidecar_cluster_name,
                        "prefix_rewrite": "/.well-known/acme-challenge/",
                        "timeout": "3.000s"
                    }
                })

        if log_debug:
            for route in self.routes:
                self._config.ir.logger.debug(f"VHost Route {prettyroute(route)}")

    def pretty(self) -> str:
        ctx_name = "-none-"

        if self.tls_context:
            ctx_name = self.tls_context.pretty()

        route_count = len(self.routes)
        route_plural = "" if (route_count == 1) else "s"

        return "<VHost %s ctx %s redir %s a %s ia %s %d route%s>" % \
               (self._hostname, ctx_name, self._needs_redirect, self._action, self._insecure_action,
                route_count, route_plural)

    def verbose_dict(self) -> dict:
        return {
            "_name": self._name,
            "_hostname": self._hostname,
            "_secure": self._secure,
            "_action": self._action,
            "_insecure_action": self._insecure_action,
            "_needs_redirect": self._needs_redirect,
            "tls_context": self.tls_context,
            "routes": self.routes,
        }


class V2ListenerCollection:
    def __init__(self, config: 'V2Config') -> None:
        self.listeners: Dict[int, 'V2Listener'] = {}
        self.config = config

    def __getitem__(self, port: int) -> 'V2Listener':
        listener = self.listeners.get(port, None)

        if listener is None:
            listener = V2Listener(self.config, port)
            self.listeners[port] = listener

        return listener

    def __contains__(self, port: int) -> bool:
        return port in self.listeners

    def items(self):
        return self.listeners.items()

    def get(self, port: int, use_proxy_proto: bool) -> 'V2Listener':
        set_upp = (not port in self)

        v2listener = self[port]

        if set_upp:
            v2listener.use_proxy_proto = use_proxy_proto
        elif v2listener.use_proxy_proto != use_proxy_proto:
            raise Exception("listener for port %d has use_proxy_proto %s, requester wants upp %s" %
                            (v2listener.service_port, v2listener.use_proxy_proto, use_proxy_proto))

        return v2listener


class V2Listener(dict):
    def __init__(self, config: 'V2Config', service_port: int) -> None:
        super().__init__()

        self.config = config
        self.service_port = service_port
        self.name = f"ambassador-listener-{self.service_port}"
        self.use_proxy_proto = False
        self.access_log: List[dict] = []
        self.upgrade_configs: Optional[List[dict]] = None
        self.vhosts: Dict[str, V2VirtualHost] = {}
        self.first_vhost: Optional[V2VirtualHost] = None
        self.http_filters: List[dict] = []
        self.listener_filters: List[dict] = []
        self.traffic_direction: str = "UNSPECIFIED"

        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        log_debug = self.config.ir.logger.isEnabledFor(logging.DEBUG)
        if log_debug:
            self.config.ir.logger.debug(f"V2Listener {self.name} created")

        # Assemble filters
        for f in self.config.ir.filters:
            v2f: dict = v2filter(f, self.config)

            # v2filter can return None to indicate that the filter config
            # should be omitted from the final envoy config. This is the
            # uncommon case, but it can happen if a filter waits utnil the
            # v2config is generated before deciding if it needs to be
            # instantiated. See IRErrorResponse for an example.
            if v2f:
                self.http_filters.append(v2f)

        # Get Access Log Rules
        for al in self.config.ir.log_services.values():
            access_log_obj: Dict[str, Any] = { "common_config": al.get_common_config() }
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
                access_log_obj['@type'] = 'type.googleapis.com/envoy.config.accesslog.v2.HttpGrpcAccessLogConfig'
                self.access_log.append({
                    "name": "envoy.access_loggers.http_grpc",
                    "typed_config": access_log_obj
                })
            else:
                # inherently TCP right now
                # tcp loggers do not support additional headers
                access_log_obj['@type'] = 'type.googleapis.com/envoy.config.accesslog.v2.TcpGrpcAccessLogConfig'
                self.access_log.append({
                    "name": "envoy.access_loggers.tcp_grpc",
                    "typed_config": access_log_obj
                })

        # Use sane access log spec in JSON
        if self.config.ir.ambassador_module.envoy_log_type.lower() == "json":
            log_format = self.config.ir.ambassador_module.get('envoy_log_format', None)
            if log_format is None:
                log_format = {
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

                tracing_config = self.config.ir.tracing
                if tracing_config and tracing_config.driver == 'envoy.tracers.datadog':
                    log_format['dd.trace_id'] = '%REQ(X-DATADOG-TRACE-ID)%'
                    log_format['dd.span_id'] = '%REQ(X-DATADOG-PARENT-ID)%'

            self.access_log.append({
                'name': 'envoy.access_loggers.file',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.config.accesslog.v2.FileAccessLog',
                    'path': self.config.ir.ambassador_module.envoy_log_path,
                    'json_format': log_format
                }
            })
        else:
            # Use a sane access log spec
            log_format = self.config.ir.ambassador_module.get('envoy_log_format', None)

            if not log_format:
                log_format = 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"'

            if log_debug:
                self.config.ir.logger.debug("V2Listener: Using log_format '%s'" % log_format)
            self.access_log.append({
                'name': 'envoy.access_loggers.file',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.config.accesslog.v2.FileAccessLog',
                    'path': self.config.ir.ambassador_module.envoy_log_path,
                    'format': log_format + '\n'
                }
            })

        # Start by building our base HTTP config...
        self.base_http_config: Dict[str, Any] = {
            'stat_prefix': 'ingress_http',
            'access_log': self.access_log,
            'http_filters': self.http_filters,
            'normalize_path': True
        }

        if self.upgrade_configs:
            self.base_http_config['upgrade_configs'] = self.upgrade_configs

        if 'use_remote_address' in self.config.ir.ambassador_module:
            self.base_http_config["use_remote_address"] = self.config.ir.ambassador_module.use_remote_address

        if 'xff_num_trusted_hops' in self.config.ir.ambassador_module:
            self.base_http_config["xff_num_trusted_hops"] = self.config.ir.ambassador_module.xff_num_trusted_hops

        if 'server_name' in self.config.ir.ambassador_module:
            self.base_http_config["server_name"] = self.config.ir.ambassador_module.server_name

        listener_idle_timeout_ms = self.config.ir.ambassador_module.get('listener_idle_timeout_ms', None)
        if listener_idle_timeout_ms:
            if 'common_http_protocol_options' in self.base_http_config:
                self.base_http_config["common_http_protocol_options"]["idle_timeout"] = "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0)
            else:
                self.base_http_config["common_http_protocol_options"] = { 'idle_timeout': "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0) }

        if 'headers_with_underscores_action' in self.config.ir.ambassador_module:
            if 'common_http_protocol_options' in self.base_http_config:
                self.base_http_config["common_http_protocol_options"]["headers_with_underscores_action"] = self.config.ir.ambassador_module.headers_with_underscores_action
            else:
                self.base_http_config["common_http_protocol_options"] = { 'headers_with_underscores_action': self.config.ir.ambassador_module.headers_with_underscores_action }

        max_request_headers_kb = self.config.ir.ambassador_module.get('max_request_headers_kb', None)
        if max_request_headers_kb:
            self.base_http_config["max_request_headers_kb"] = max_request_headers_kb

        if 'enable_http10' in self.config.ir.ambassador_module:
            http_options = self.base_http_config.setdefault("http_protocol_options", {})
            http_options['accept_http_10'] = self.config.ir.ambassador_module.enable_http10

        if 'preserve_external_request_id' in self.config.ir.ambassador_module:
            self.base_http_config["preserve_external_request_id"] = self.config.ir.ambassador_module.preserve_external_request_id

        if 'forward_client_cert_details' in self.config.ir.ambassador_module:
            self.base_http_config["forward_client_cert_details"] = self.config.ir.ambassador_module.forward_client_cert_details

        if 'set_current_client_cert_details' in self.config.ir.ambassador_module:
            self.base_http_config["set_current_client_cert_details"] = self.config.ir.ambassador_module.set_current_client_cert_details

        if self.config.ir.tracing:
            self.base_http_config["generate_request_id"] = True

            self.base_http_config["tracing"] = {}
            self.traffic_direction = "OUTBOUND"

            req_hdrs = self.config.ir.tracing.get('tag_headers', [])

            if req_hdrs:
                self.base_http_config["tracing"]["request_headers_for_tags"] = req_hdrs

            sampling = self.config.ir.tracing.get('sampling', {})
            if sampling:
                client_sampling = sampling.get('client', None)
                if client_sampling is not None:
                    self.base_http_config["tracing"]["client_sampling"] = {
                        "value": client_sampling
                    }

                random_sampling = sampling.get('random', None)
                if random_sampling is not None:
                    self.base_http_config["tracing"]["random_sampling"] = {
                        "value": random_sampling
                    }

                overall_sampling = sampling.get('overall', None)
                if overall_sampling is not None:
                    self.base_http_config["tracing"]["overall_sampling"] = {
                        "value": overall_sampling
                    }


        proper_case: bool = self.config.ir.ambassador_module['proper_case']

        # Get the list of downstream headers whose casing should be overriden
        # from the Ambassador module. We configure the upstream side of this
        # in v2cluster.py
        header_case_overrides = self.config.ir.ambassador_module.get('header_case_overrides', None)
        if header_case_overrides:
            if proper_case:
                self.config.ir.post_error(
                    "Only one of 'proper_case' or 'header_case_overrides' fields may be set on " +\
                    "the Ambassador module. Honoring proper_case and ignoring " +\
                    "header_case_overrides.")
                header_case_overrides = None
            if not isinstance(header_case_overrides, list):
                # The header_case_overrides field must be an array.
                self.config.ir.post_error("Ambassador module config 'header_case_overrides' must be an array")
                header_case_overrides = None
            elif len(header_case_overrides) == 0:
                # Allow an empty list to mean "do nothing".
                header_case_overrides = None

        if header_case_overrides:
            # We have this config validation here because the Ambassador module is
            # still an untyped config. That is, we aren't yet using a CRD or a
            # python schema to constrain the configuration that can be present.
            rules = []
            for hdr in header_case_overrides:
                if not isinstance(hdr, str):
                    self.config.ir.post_error("Skipping non-string header in 'header_case_overrides': {hdr}")
                    continue
                rules.append(hdr)

            if len(rules) == 0:
                self.config.ir.post_error(f"Could not parse any valid string headers in 'header_case_overrides': {header_case_overrides}")
            else:
                # Create custom header rules that map the lowercase version of every element in
                # `header_case_overrides` to the the respective original casing.
                #
                # For example the input array [ X-HELLO-There, X-COOL ] would create rules:
                # { 'x-hello-there': 'X-HELLO-There', 'x-cool': 'X-COOL' }. In envoy, this effectively
                # overrides the response header case by remapping the lowercased version (the default
                # casing in envoy) back to the casing provided in the config.
                custom_header_rules: Dict[str, Dict[str, dict]] = {
                    'custom': {
                        'rules': {
                            header.lower() : header for header in rules
                        }
                    }
                }
                http_options = self.base_http_config.setdefault("http_protocol_options", {})
                http_options["header_key_format"] = custom_header_rules

        if proper_case:
            proper_case_header: Dict[str, Dict[str, dict]] = {'header_key_format': {'proper_case_words': {}}}
            if 'http_protocol_options' in self.base_http_config:
                self.base_http_config["http_protocol_options"].update(proper_case_header)
            else:
                self.base_http_config["http_protocol_options"] = proper_case_header

    def add_irlistener(self, listener: IRListener) -> None:
        if listener.service_port != self.service_port:
            # This is a problem.
            raise Exception("V2Listener %s: trying to add listener %s on %s:%d??" %
                            (self.name, listener.name, listener.hostname, listener.service_port))

        # OK, make sure we don't somehow have a VHost collision.
        if listener.hostname in self.vhosts:
            raise Exception("V2Listener %s: listener %s on %s:%d already has a vhost??" %
                            (self.name, listener.name, listener.hostname, listener.service_port))

    # Weirdly, the action is optional but the insecure_action is not. This is not a typo.
    def make_vhost(self, name: str, hostname: str, context: Optional[IRTLSContext], secure: bool,
                   action: Optional[str], insecure_action: str) -> None:
        if self.config.ir.logger.isEnabledFor(logging.DEBUG):
            self.config.ir.logger.debug("V2Listener %s: adding VHost %s for host %s, secure %s, insecure %s)" %
                                       (self.name, name, hostname, action, insecure_action))

        vhost = self.vhosts.get(hostname)

        if vhost:
            if ((hostname != vhost._hostname) or
                (context != vhost._ctx) or
                (secure != vhost._secure) or
                (action != vhost._action) or
                (insecure_action != vhost._insecure_action)):
                raise Exception("V2Listener %s: trying to make vhost %s for %s but one already exists" %
                                (self.name, name, hostname))
            else:
                return

        vhost = V2VirtualHost(config=self.config, listener=self,
                              name=name, hostname=hostname, ctx=context,
                              secure=secure, action=action, insecure_action=insecure_action)
        self.vhosts[hostname] = vhost

        if not self.first_vhost:
            self.first_vhost = vhost

    def finalize(self) -> None:
        if self.config.ir.logger.isEnabledFor(logging.DEBUG):
            self.config.ir.logger.debug(f"V2Listener finalize {self.pretty()}")

        # Check if AMBASSADOR_ENVOY_BIND_ADDRESS is set, and if so, bind Envoy to that external address.
        if "AMBASSADOR_ENVOY_BIND_ADDRESS" in environ:
            envoy_bind_address = environ.get("AMBASSADOR_ENVOY_BIND_ADDRESS")
        else:
            envoy_bind_address = "0.0.0.0"

        # OK. Assemble the high-level stuff for Envoy.
        self.address = {
            "socket_address": {
                "address": envoy_bind_address,
                "port_value": self.service_port,
                "protocol": "TCP"
            }
        }

        self.filter_chains: List[dict] = []
        need_tcp_inspector = False

        for vhostname, vhost in self.vhosts.items():
            # Finalize this VirtualHost...
            vhost.finalize()

            if vhost._hostname == "*":
                domains = [vhost._hostname]
            else:
                if vhost._ctx is not None and vhost._ctx.hosts is not None and len(vhost._ctx.hosts) > 0:
                    domains = vhost._ctx.hosts
                else:
                    domains = [vhost._hostname]

            # ...then build up the Envoy structures around it.
            filter_chain: Dict[str, Any] = {
                "filter_chain_match": vhost.filter_chain_match,
            }

            if vhost.tls_context:
                filter_chain["tls_context"] = vhost.tls_context
                need_tcp_inspector = True

            http_config = dict(self.base_http_config)
            http_config["route_config"] = {
                "virtual_hosts": [
                    {
                        "name": f"{self.name}-{vhost._name}",
                        "domains": domains,
                        "routes": vhost.routes
                    }
                ]
            }

            if parse_bool(self.config.ir.ambassador_module.get("strip_matching_host_port", "false")):
                http_config["strip_matching_host_port"] = True

            if parse_bool(self.config.ir.ambassador_module.get("merge_slashes", "false")):
                http_config["merge_slashes"] = True

            if parse_bool(self.config.ir.ambassador_module.get("reject_requests_with_escaped_slashes", "false")):
                http_config["path_with_escaped_slashes_action"] = "REJECT_REQUEST"

            filter_chain["filters"] = [
                {
                    "name": "envoy.filters.network.http_connection_manager",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                        **http_config
                    }
                }
            ]

            self.filter_chains.append(filter_chain)

        if self.use_proxy_proto:
            self.listener_filters.append({
                'name': 'envoy.filters.listener.proxy_protocol'
            })

        if need_tcp_inspector:
            self.listener_filters.append({
                'name': 'envoy.filters.listener.tls_inspector'
            })

    def as_dict(self) -> dict:
        listener = {
            "name": self.name,
            "address": self.address,
            "filter_chains": self.filter_chains,
            "listener_filters": self.listener_filters,
            "traffic_direction": self.traffic_direction
        }
        # We only want to add the buffer limit setting to the listener if specified in the module.
        # Otherwise, we want to leave it unset and allow Envoys Default 1MiB setting.
        if 'buffer_limit_bytes' in self.config.ir.ambassador_module and self.config.ir.ambassador_module.buffer_limit_bytes != None:
            listener["per_connection_buffer_limit_bytes"] = self.config.ir.ambassador_module.buffer_limit_bytes
        return listener

    def pretty(self) -> dict:
        return { "name": self.name,
                 "port": self.service_port,
                 "use_proxy_proto": self.use_proxy_proto,
                 "vhosts": { k: v.pretty() for k, v in self.vhosts.items() } }

    @classmethod
    def dump_listeners(cls, logger, listeners_by_port) -> None:
        pretty = { k: v.pretty() for k, v in listeners_by_port.items() }

        logger.debug(f"V2Listeners: {dump_json(pretty, pretty=True)}")

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.listeners = []
        logger = config.ir.logger

        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        log_debug = logger.isEnabledFor(logging.DEBUG)

        # OK, so we need to construct one or more V2Listeners, based on our IRListeners.
        # The highest-level thing that defines an Envoy listener is a port, so start
        # with that.

        listeners_by_port = V2ListenerCollection(config)

        # Also, in Edge Stack, the magic extremely-low-precedence / Mapping is always routed,
        # rather than being redirected. If a user doesn't want this behavior, they can override
        # the Mapping.

        first_irlistener_by_port: Dict[int, IRListener] = {}

        for irlistener in config.ir.listeners:
            if irlistener.service_port not in first_irlistener_by_port:
                first_irlistener_by_port[irlistener.service_port] = irlistener

            if log_debug:
                logger.debug(f"V2Listeners: working on {irlistener.pretty()}")

            # Grab a new V2Listener for this IRListener...
            listener = listeners_by_port.get(irlistener.service_port, irlistener.use_proxy_proto)
            listener.add_irlistener(irlistener)

            # What VirtualHost hostname are we trying to work with here?
            vhostname = irlistener.hostname or "*"

            listener.make_vhost(name=vhostname,
                                hostname=vhostname,
                                context=irlistener.context,
                                secure=True,
                                action=irlistener.secure_action,
                                insecure_action=irlistener.insecure_action)

            if (irlistener.insecure_addl_port is not None) and (irlistener.insecure_addl_port > 0):
                # Make sure we have a listener on the right port for this.
                listener = listeners_by_port.get(irlistener.insecure_addl_port, irlistener.use_proxy_proto)

                if irlistener.insecure_addl_port not in first_irlistener_by_port:
                    first_irlistener_by_port[irlistener.insecure_addl_port] = irlistener

                # Do we already have a VHost for this hostname?
                if vhostname not in listener.vhosts:
                    # Nope, add one. Also, no, it is not a bug to have action=None.
                    # There is no secure action for this vhost.
                    listener.make_vhost(name=vhostname,
                                        hostname=vhostname,
                                        context=None,
                                        secure=False,
                                        action=None,
                                        insecure_action=irlistener.insecure_action)

        if log_debug:
            logger.debug(f"V2Listeners: after IRListeners")
            cls.dump_listeners(logger, listeners_by_port)

        # Make sure that each listener has a '*' vhost.
        for port, listener in listeners_by_port.items():
            if not '*' in listener.vhosts:
                # Force the first VHost to '*'. I know, this is a little weird, but it's arguably
                # the least surprising thing to do in most situations.
                assert listener.first_vhost
                first_vhost = listener.first_vhost
                first_vhost._hostname = '*'
                first_vhost._name = f"{first_vhost._name}-forced-star"

        if config.ir.edge_stack_allowed and not config.ir.agent_active:
            # If we're running Edge Stack, and we're not an intercept agent, make sure we have
            # a listener on port 8080, so that we have a place to stand for ACME.

            if 8080 not in listeners_by_port:
                # Check for a listener on the main service port to see if the proxy proto
                # is enabled.
                main_listener = first_irlistener_by_port.get(config.ir.ambassador_module.service_port, None)
                use_proxy_proto = main_listener.use_proxy_proto if main_listener else False

                # Force a listener on 8080 with a VHost for '*' that rejects everything. The ACME
                # hole-puncher will override the reject for ACME, and nothing else will get through.
                if log_debug:
                    logger.debug(f"V2Listeners: listeners_by_port has no 8080, forcing Edge Stack listener on 8080")
                listener = listeners_by_port.get(8080, use_proxy_proto)

                # Remember, it is not a bug to have action=None. There is no secure action
                # for this vhost.
                listener.make_vhost(name="forced-8080",
                                    hostname="*",
                                    context=None,
                                    secure=False,
                                    action=None,
                                    insecure_action='Reject')

        prune_unreachable_routes = config.ir.ambassador_module['prune_unreachable_routes']

        # OK. We have all the listeners. Time to walk the routes (note that they are already ordered).
        for c_route in config.routes:
            # Remember which hosts this can apply to
            route_hosts = c_route.host_constraints(prune_unreachable_routes)

            # Remember, also, if a precedence was set.
            route_precedence = c_route.get('_precedence', None)

            if log_debug:
                logger.debug(f"V2Listeners: route {prettyroute(c_route)}...")

            # Build a cleaned-up version of this route without the '_sni' and '_precedence' elements...
            insecure_route: DictifiedV2Route = dict(c_route)
            insecure_route.pop('_sni', None)
            insecure_route.pop('_precedence', None)

            # ...then copy _that_ so we can make a secured version with an explicit XFP check.
            #
            # (Obviously the user may have put in an XFP check by hand here, in which case the
            # insecure_route isn't really insecure, but that's not actually up to us to mess with.)
            #
            # But wait, I hear you cry! Can't we use use require_tls: True in a VirtualHost?? Well,
            # no, not if we want to allow ACME challenges to flow through as cleartext at the same
            # time...
            secure_route = dict(insecure_route)

            found_xfp = False
            for header in secure_route["match"].get("headers", []):
                if header.get("name", "").lower() == "x-forwarded-proto":
                    found_xfp = True
                    break

            if not found_xfp:
                # Ew.
                match_copy = dict(secure_route["match"])
                secure_route["match"] = match_copy

                headers_copy = list(match_copy.get("headers") or [])
                match_copy["headers"] = headers_copy

                headers_copy.append({
                    "name": "x-forwarded-proto",
                    "exact_match": "https"
                })

            # Also gen up a redirecting route.
            redirect_route = dict(insecure_route)
            redirect_route.pop("route", None)
            redirect_route["redirect"] = {
                "https_redirect": True
            }

            # We now have a secure route and an insecure route, so we need to walk all listeners
            # and all vhosts, and match up the routes with the vhosts.

            for port, listener in listeners_by_port.items():
                for vhostkey, vhost in listener.vhosts.items():
                    # For each vhost, we need to look at things for the secure world as well
                    # as the insecure world, depending on what the action is exactly (and note
                    # that we can have an action of None if we're looking at a vhost created
                    # by an insecure_addl_port).

                    candidates: List[Tuple[bool, DictifiedV2Route, str]] = []
                    vhostname = vhost._hostname

                    if vhost._action is not None:
                        candidates.append(( True, secure_route, vhost._action ))

                    if vhost._insecure_action == "Redirect":
                        candidates.append(( False, redirect_route, "Redirect" ))
                    elif vhost._insecure_action is not None:
                        candidates.append((False, insecure_route, vhost._insecure_action))

                    for secure, route, action in candidates:
                        variant = "secure" if secure else "insecure"

                        if route["match"].get("prefix", None) == "/.well-known/acme-challenge/":
                            # We need to be sure to route ACME challenges, no matter what else is going
                            # on (this is the infamous ACME hole-puncher mentioned everywhere).
                            if log_debug:
                                logger.debug(f"V2Listeners: {listener.name} {vhostname} force Route for ACME challenge")
                            action = "Route"

                            # We have to force the correct route entry, too, just in case. (Note that right now,
                            # the user can't create a Mapping that forces redirection. When they can do this
                            # per-Mapping, well, really, we can't force them to not redirect if they explicitly
                            # ask for it, and that'll be OK.)

                            if secure:
                                route = secure_route
                            else:
                                route = insecure_route
                        elif ('*' not in route_hosts) and (vhostname != '*') and (vhostname not in route_hosts):
                            # Drop this because the host is mismatched.
                            if log_debug:
                                logger.debug(
                                    f"V2Listeners: {listener.name} {vhostname} {variant}: force Reject (rhosts {sorted(route_hosts)}, vhost {vhostname})")
                            action = "Reject"
                        elif (config.ir.edge_stack_allowed and
                              (route_precedence == -1000000) and
                              (route["match"].get("safe_regex", {}).get("regex", None) == "^/$")):
                            if log_debug:
                                logger.debug(
                                    f"V2Listeners: {listener.name} {vhostname} {variant}: force Route for fallback Mapping")
                            action = "Route"

                            # Force the actual route entry, instead of using the redirect_route, too.
                            # (If the user overrides the fallback with their own route at precedence -1000000,
                            # uh.... y'know what, on their own head be it.)
                            route = insecure_route

                        if action != 'Reject':
                            if log_debug:
                                logger.debug(
                                    f"V2Listeners: {listener.name} {vhostname} {variant}: Accept as {action}")
                            vhost.routes.append(route)
                        else:
                            if log_debug:
                                logger.debug(
                                    f"V2Listeners: {listener.name} {vhostname} {variant}: Drop")

                        # Also, remember if we're redirecting so that the VHost finalizer can DTRT
                        # for ACME.
                        if action == 'Redirect':
                            vhost.needs_redirect()

        # OK. Finalize the world.
        for port, listener in listeners_by_port.items():
            listener.finalize()

        if log_debug:
            logger.debug("V2Listeners: after finalize")
            cls.dump_listeners(logger, listeners_by_port)

        for k, v in listeners_by_port.items():
            config.listeners.append(v.as_dict())

        # logger.info(f"==== ENVOY LISTENERS ====: {dump_json(config.listeners, pretty=True)}")

        # We need listeners for the TCPMappingGroups too.
        tcplisteners: Dict[str, V2TCPListener] = {}

        for irgroup in config.ir.ordered_groups():
            if not isinstance(irgroup, IRTCPMappingGroup):
                continue

            # OK, good to go. Do we already have a TCP listener binding where this one does?
            group_key = irgroup.bind_to()
            tcplistener = tcplisteners.get(group_key, None)

            if log_debug:
                config.ir.logger.debug("V2TCPListener: group at %s found %s listener" %
                                       (group_key, "extant" if tcplistener else "no"))

            if not tcplistener:
                # Nope. Make a new one and save it.
                tcplistener = config.save_element('listener', irgroup, V2TCPListener(config, irgroup))
                assert tcplistener
                config.listeners.append(tcplistener)
                tcplisteners[group_key] = tcplistener

            # Whether we just created this listener or not, add this irgroup to it.
            tcplistener.add_group(config, irgroup)
