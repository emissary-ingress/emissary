# Copyright 2021 Datawire. All rights reserved.
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

import logging

from multi import multi
from ...ir.irauth import IRAuth
from ...ir.irerrorresponse import IRErrorResponse
from ...ir.irbuffer import IRBuffer
from ...ir.irgzip import IRGzip
from ...ir.irfilter import IRFilter
from ...ir.irratelimit import IRRateLimit
from ...ir.ircors import IRCORS
from ...ir.ircluster import IRCluster

from ...utils import parse_bool
from ...utils import ParsedService as Service

if TYPE_CHECKING:
    from . import V3Config

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
def V3HTTPFilter(irfilter: IRFilter, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

    if irfilter.kind == 'IRAuth':
        if irfilter.api_version == 'getambassador.io/v0':
            return 'IRAuth_v0'
        elif irfilter.api_version in ['getambassador.io/v1', 'getambassador.io/v2']:
            return 'IRAuth_v1-2'
        else:
            irfilter.post_error('AuthService version %s unknown, treating as v2' % irfilter.api_version)
            return 'IRAuth_v1-2'
    else:
        return irfilter.kind

@V3HTTPFilter.when("IRBuffer")
def V3HTTPFilter_buffer(buffer: IRBuffer, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.buffer',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.extensions.filters.http.buffer.v3.Buffer',
            "max_request_bytes": buffer.max_request_bytes
        }
    }

@V3HTTPFilter.when("IRGzip")
def V3HTTPFilter_gzip(gzip: IRGzip, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.gzip',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.extensions.filters.http.gzip.v3.Gzip',
            'memory_level': gzip.memory_level,
            'compression_level': gzip.compression_level,
            'compression_strategy': gzip.compression_strategy,
            'window_bits': gzip.window_bits,
            'compressor': {
                'content_type': gzip.content_type,
                'content_length': gzip.content_length,
                'disable_on_etag_header': gzip.disable_on_etag_header,
                'remove_accept_encoding_header': gzip.remove_accept_encoding_header,
            },
        }
    }

@V3HTTPFilter.when("ir.grpc_http1_bridge")
def V3HTTPFilter_grpc_http1_bridge(irfilter: IRFilter, v3config: 'V3Config'):
    del irfilter  # silence unused-variable warning
    del v3config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.grpc_http1_bridge'
    }

@V3HTTPFilter.when("ir.grpc_web")
def V3HTTPFilter_grpc_web(irfilter: IRFilter, v3config: 'V3Config'):
    del irfilter  # silence unused-variable warning
    del v3config  # silence unused-variable warning

    return {
        'name': 'envoy.filters.http.grpc_web'
    }

@V3HTTPFilter.when("ir.grpc_stats")
def V3HTTPFilter_grpc_stats(irfilter: IRFilter, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning
    config = typecast(Dict[str, Any], irfilter.config_dict())

    return {
        'name': 'envoy.filters.http.grpc_stats',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.extensions.filters.http.grpc_stats.v3.FilterConfig',
            **config,
        }
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

@V3HTTPFilter.when("IRAuth_v0")
def V3HTTPFilter_authv0(auth: IRAuth, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

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
        'name': 'envoy.filters.http.ext_authz',
        'typed_config': {
            '@type': 'type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz',
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


@V3HTTPFilter.when("IRAuth_v1-2")
def V3HTTPFilter_authv1(auth: IRAuth, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

    assert auth.cluster
    cluster = typecast(IRCluster, auth.cluster)

    if auth.api_version not in ['getambassador.io/v1', 'getambassador.io/v2']:
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
            'name': 'envoy.filters.http.ext_authz',
            'typed_config': {
                '@type': 'type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz',
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
                '@type': 'type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz',
                'grpc_service': {
                    'envoy_grpc': {
                        'cluster_name': cluster.envoy_name
                    },
                    'timeout': "%0.3fs" % (float(auth.timeout_ms) / 1000.0)
                },
                'transport_api_version': protocol_version.replace("alpha", "").upper(),
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
@V3HTTPFilter.when("IRErrorResponse")
def V3HTTPFilter_error_response(error_response: IRErrorResponse, v3config: 'V3Config'):
    # Error response configuration can come from the Ambassador module, on a
    # a Mapping, or both. We need to use the response_map filter if either one
    # of these sources defines error responses. First, check if any route
    # has per-filter config for error responses. If so, we know a Mapping has
    # defined error responses.
    route_has_error_responses = False
    for route in v3config.routes:
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


@V3HTTPFilter.when("IRRateLimit")
def V3HTTPFilter_ratelimit(ratelimit: IRRateLimit, v3config: 'V3Config'):
    config = dict(ratelimit.config)

    if 'timeout_ms' in config:
        tm_ms = config.pop('timeout_ms')

        config['timeout'] = "%0.3fs" % (float(tm_ms) / 1000.0)

    # If here, we must have a ratelimit service configured.
    assert v3config.ratelimit
    config['rate_limit_service'] = dict(v3config.ratelimit)
    config['@type'] = 'type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit'

    return {
        'name': 'envoy.filters.http.ratelimit',
        'typed_config': config,
    }


@V3HTTPFilter.when("IRIPAllowDeny")
def V3HTTPFilter_ipallowdeny(irfilter: IRFilter, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

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
            "@type": "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBAC",
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


@V3HTTPFilter.when("ir.cors")
def V3HTTPFilter_cors(cors: IRCORS, v3config: 'V3Config'):
    del cors    # silence unused-variable warning
    del v3config  # silence unused-variable warning

    return { 'name': 'envoy.filters.http.cors' }


@V3HTTPFilter.when("ir.router")
def V3HTTPFilter_router(router: IRFilter, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

    od: Dict[str, Any] = { 'name': 'envoy.filters.http.router' }

    # Use this config base if we actually need to set config fields below. We don't set
    # this on `od` by default because it would be an error to end up returning a typed
    # config that has no real config fields, only a type.
    typed_config_base = {
        '@type': 'type.googleapis.com/envoy.extensions.filters.http.router.v3.Router',
    }

    if router.ir.tracing:
        typed_config = od.setdefault('typed_config', typed_config_base)
        typed_config['start_child_span'] = True

    if parse_bool(router.ir.ambassador_module.get('suppress_envoy_headers', 'false')):
        typed_config = od.setdefault('typed_config', typed_config_base)
        typed_config['suppress_envoy_headers'] = True

    return od


@V3HTTPFilter.when("ir.lua_scripts")
def V3HTTPFilter_lua(irfilter: IRFilter, v3config: 'V3Config'):
    del v3config  # silence unused-variable warning

    config_dict = irfilter.config_dict()
    config: Dict[str, Any]
    config = {
        'name': 'envoy.filters.http.lua'
    }

    if config_dict:
        config['typed_config'] = config_dict
        config['typed_config']['@type'] = 'type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua'

    return config
