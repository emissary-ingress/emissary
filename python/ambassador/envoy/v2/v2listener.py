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
import copy
from typing import Any, Dict, Iterable, List, NamedTuple, Optional, Sequence, Set, Tuple, TYPE_CHECKING
from typing import cast as typecast

from os import environ

import json

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
from ...ir.irtlscontext import IRTLSContext

from ...utils import ParsedService as Service

from .v2route import V2Route
from .v2tls import V2TLSContext

if TYPE_CHECKING:
    from . import V2Config

DictifiedV2Route = Dict[str,Any]

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
    return json.dumps(x, sort_keys=True, indent=4)


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


def invert_headermatchers(matchers: List[Dict[str,Any]]) -> List[Dict[str,Any]]:
    """Given a list of "envoy.api.v2.route.VirtualHost.HeaderMatcher"s,
    return a copy of it with "invert_match" field fliped on each
    member.

    """
    matchers = copy.deepcopy(matchers)
    for matcher in matchers:
        invert_match = matcher.pop("invert_match", False)
        if not invert_match:
            matcher["invert_match"] = True
    return matchers


def reduce_hostglobs(inglobs: Iterable[str]) -> Set[str]:
    """Given a list of hostglobs, reduce it to a minimal set that matches those hosts; the trivial example is that
    if the input list contains "*", then the result is just ["*"].

    This matches the semantics of `envoy.api.v2.route.VirtualHost.domains`.

    """
    outglobs: Set[str] = set()
    for a in inglobs:
        subsumed = False
        for b in inglobs:
            if a == b:
                continue

            if b == "*": # special wildcard
                subsumed = True
                break
            elif b.endswith("*"): # prefix match
                if a.startswith(b[:-1]):
                    subsumed = True
                    break
            elif b.startswith("*"): # suffix match
                if a.endswith(b[1:]):
                    subsumed = True
                    break
            else: # exact match
                pass

        if not subsumed:
            outglobs.add(a)
    return outglobs


def sorted_hostglobs(inglobs: Iterable[str]) -> Sequence[str]:
    """Given a list of hostglobs, sort the list according to `envoy.api.v2.route.VirtualHost.domains` precedence;
    highest precedence first.

    """
    def key(hostglob: str) -> Tuple:
        # higher numbers = higher precedence
        if hostglob == "*": # special wildcard
            return (0, len(hostglob))
        elif hostglob.endswith("*"): # prefix match
            return (1, len(hostglob))
        elif hostglob.startswith("*"): # suffix match
            return (2, len(hostglob))
        else: # exact match
            return (3, len(hostglob))
    return sorted(inglobs, key=key, reverse=True)


def hostglob_to_headermatchers(hostglob: str, other_hostglobs: Iterable[str]=[]) -> List[Dict[str,Any]]:
    """Given a hostglob to match, and a list of hostglobs to _not_ match, return a list of HeaderMatchers that
    replicate the behavior of `envoy.api.v2.route.VirtualHost.domains`.

    """

    matchers: List[Dict[str,Any]] = []
    exceptions: Set[str] = set()

    # Check each type of domain matcher from lowest precedence to highest
    if hostglob == "*": # special wildcard
        # Start by matching everything (don't add anything to 'matchers' yet).

        # OK, now we need to go through all of the other hostglobs and add exceptions for the ones that are
        # higher-precedence than us (hint: everything is higher-precedence than "*").
        for other in other_hostglobs:
            if other == hostglob:
                continue
            # Everything is higher-precedence than "*".
            exceptions.add(other)
    elif hostglob.endswith("*"): # prefix match
        matchers.append({
            "name": ":authority",
            "prefix_match": hostglob[:-1],
        })
        # OK, now we need to go through all of the other hostglobs and add exceptions for the ones that are
        # higher-precedence than us.
        for other in other_hostglobs:
            if other == hostglob:
                continue
            if other.startswith("*") and other != "*":
                # suffix matches are higher priority
                exceptions.add(other)
            elif other.startswith(hostglob[:-1]):
                # more specific matches (including exact matches) are higher priority
                exceptions.add(other)
    elif hostglob.startswith("*"): # suffix match
        matchers.append({
            "name": ":authority",
            "suffix_match": hostglob[1:],
        })
        # OK, now we need to go through all of the other hostglobs and add exceptions for the ones that are
        # higher-precedence than us.
        for other in other_hostglobs:
            if other == hostglob:
                continue
            if other.endswith(hostglob[1:]):
                # more specific matches (including exact matches) are higher priority
                exceptions.add(other)
    else: # exact match
        matchers.append({
            "name": ":authority",
            "exact_match": hostglob,
        })

    for exception in reduce_hostglobs(exceptions):
        matchers.extend(invert_headermatchers(hostglob_to_headermatchers(exception)))

    return matchers


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
        auth_info['config']['clear_route_cache'] = True

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
    """V2TCPListener is the Python type for a gRPC `envoy.api.v2.Listener`
    that happens to NOT contain an HttpConnectionManager in its filter
    chain; therefore serving `TCPMappings` rather than HTTP
    `Mappings`.

    """
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


class V2VirtualHost:
    def __init__(self, config: 'V2Config', listener: 'V2Listener',
                 name: str, hostname: str, ctx: Optional[IRTLSContext],
                 action: Optional[str]) -> None:
        super().__init__()

        self._config = config
        self._listener = listener
        self.name = name
        self._hostname = hostname
        self._ctx = ctx
        self._action = action
        self._hole_for_root = False
        self._insecure_actions: Dict[str,str] = {}  # hostname -> action
        self.routes: List[DictifiedV2Route] = []
        self._tls_context = V2TLSContext(ctx)

    def ensure_acme_route(self):
        """ensure_acme_route adjusts self.routes to ensure that there's a route
        for `/.well-known/acme-challenge/` such that Envoy won't 404
        it before calling to ext_authz.

        """
        for route in self.routes:
            if route["match"].get("prefix", None) == "/.well-known/acme-challenge/":
                # Nothing to do here if ACME prefix already exists
                return

        # The target cluster doesn't actually matter -- the auth service grabs the
        # challenge and does the right thing. But we do need a cluster that actually
        # exists, so use the sidecar cluster.
        if not self._config.ir.sidecar_cluster_name:
            # Uh whut? how is Edge Stack running exactly?
            raise Exception("Edge Stack claims to be running, but we have no sidecar cluster??")

        self._config.ir.logger.debug(f"V2VirtualHost {self.name}: finalize: punching a hole for ACME")

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

    # Build a cleaned-up version of this route without the '_sni' and '_precedence' elements...
    def generate_insecure_route(self, route: V2Route) -> dict:
        """
        Generates a cleaned up insecure route from a given route, ready to be put in Envoy config.
        :param route: V2Route from which insecure route will be created.
        :return: insecure route
        """
        insecure_route = copy.deepcopy(dict(route))
        insecure_route.pop('_sni', None)
        insecure_route.pop('_precedence', None)
        return insecure_route

    # ...then copy _that_ so we can make a secured version with an explicit XFP check.
    #
    # (Obviously the user may have put in an XFP check by hand here, in which case the
    # insecure_route isn't really insecure, but that's not actually up to us to mess with.)
    #
    # But wait, I hear you cry! Can't we use use require_tls: True in a VirtualHost?? Well,
    # no, not if we want to allow ACME challenges to flow through as cleartext at the same
    # time...
    def generate_secure_route(self, route: V2Route) -> dict:
        """
        Generates a cleaned up secure route from a given route, ready to be put in Envoy config.
        :param route: V2Route from which secure route will be created.
        :return: secure route
        """
        secure_route = self.generate_insecure_route(route)
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

        return secure_route

    # maybe_add_route inspects a V2Route and decides how and where it
    # fits in the supplied vhost (if at all).  Depending on the route
    # being insecure, secure, redirect, etc it configures and adds it
    # to the vhost.
    def maybe_add_route(self, c_route: V2Route):
        logger = self._config.ir.logger
        edge_stack_allowed = self._config.ir.edge_stack_allowed
        logger.debug(f"V2VirtualHost {self.name}: considering route={c_route} where edge_stack_allowed={edge_stack_allowed}")

        route_hosts = set(c_route.get('_sni', {}).get('hosts', []))
        route_precedence = c_route.get('_precedence', None)

        if route_hosts and (self._hostname != '*') and (self._hostname not in route_hosts):
            # Drop this because the host is mismatched.
            logger.debug(f"V2VirtualHost {self.name}: Dropping (rhosts {sorted(route_hosts)})")
            return

        if (edge_stack_allowed and
            (route_precedence == -1000000) and
            (c_route["match"].get("safe_regex", {}).get("regex", None) == "^/$")):
            # Force the actual route entry, instead of using the redirect_route, too.
            # (If the user overrides the fallback with their own route at precedence -1000000,
            # uh.... y'know what, on their own head be it.)
            logger.debug(
                f"V2VirtualHost {self.name}: force Route for fallback Mapping")
            self._hole_for_root = True

        logger.debug(f"V2VirtualHost {self.name}: Accepting")

        # Always generate a secure route (because we might want it on the insecure listener if we're trusing XFP)
        self.routes.append(self.generate_secure_route(c_route))
        # And also generate an insecure route if this is the insecure listener
        if self._action is None:
            self.routes.append(self.generate_insecure_route(c_route))

    def finalize(self) -> None:
        # Even though this is called V2VirtualHost, we track the filter_chain_match here,
        # because it makes more sense, because this is where we have the domain information.
        # The 1:1 correspondence that this implies between filters and domains may need to
        # change later, of course...
        self._config.ir.logger.debug(f"V2VirtualHost {self.name}: finalize: {jsonify(self.pretty())}")

        match: Dict[str,Any] = {}

        if self._ctx:
            match["transport_protocol"] = "tls"

            # Make sure we include a server name match if the hostname isn't "*".
            # ... and we only want server_names when this is a TLS listener, it only works with SNI over TLS.
            if self._hostname and (self._hostname != '*'):
                    match["server_names"] = [ self._hostname ]

        self.filter_chain_match = match

        # If we're on Edge Stack and we're not an intercept agent, then make sure that Envoy won't 404 ACME
        # challenges before they ever get to the AuthService.
        if self._config.ir.edge_stack_allowed and not self._config.ir.agent_active:
            # Punch ACME hole in routes
            self.ensure_acme_route()

        if self._action is None:
            if "*" not in self._insecure_actions:
                self._insecure_actions["*"] = "Redirect"
            self._config.ir.logger.debug(f"V2VirtualHost {self.name}: _insecure_actions={repr(self._insecure_actions)}")

            # Holes to poke in the Redirect and Reject actions...
            holes = [{"name": ":path", "prefix_match": "/.well-known/acme-challenge/"}]
            if self._hole_for_root:
                holes.append({"name": ":path", "exact_match": "/"})

            # For every Host, insert a route to perform its insecure.action.
            #
            # But first, to do that we need to sort them by precedence
            # and figure out which ones are redundant.
            hostglob_groups: List[List[str]] = []
            prev_hostaction = ""
            for hostglob in sorted_hostglobs(self._insecure_actions.keys()):
                hostaction = self._insecure_actions[hostglob]
                if hostaction != prev_hostaction:
                    hostglob_groups.append([])
                hostglob_groups[-1].append(hostglob)
                prev_hostaction = hostaction

            # OK, now insert a route to perform each insecure.action.
            insecure_action_routes: List[DictifiedV2Route] = []
            exceptions: List[str] = []
            for hostglob_group in hostglob_groups:
                hostaction = self._insecure_actions[hostglob_group[0]]
                other_hostglobs = [e for e in exceptions if self._insecure_actions[e] != hostaction]
                for hostglob in sorted(reduce_hostglobs(hostglob_group)):
                    route: Optional[Dict[str,Any]] = {
                        "Redirect": {
                            "match": {
                                "case_sensitive": True,
                                "prefix": "/",
                            },
                            "redirect": {
                                "https_redirect": True,
                            },
                        },
                        "Reject": {
                            "match": {
                                "prefix": "/",
                            },
                            "direct_response": {
                                "status": 404,
                            },
                        },
                        "Route": None,  # fall through to the normal Route list
                    }[hostaction]

                    if not route:
                        exceptions.append(hostglob)
                        continue

                    route["match"].setdefault("headers", [])

                    # Poke necessary holes in the action.
                    route["match"]["headers"].extend(invert_headermatchers(holes))

                    # Don't do the insecure action if XFP says we're actually secure (trusting that Envoy has already
                    # validated XFP).
                    route["match"]["headers"].append({
                        "exact_match": "http",
                        "name": "x-forwarded-proto",
                    })

                    # Use the :authority header to decide whether this route applies.
                    route["match"]["headers"].extend(hostglob_to_headermatchers(hostglob, other_hostglobs))

                    insecure_action_routes.append(route)

            # Now insert those routes in to the main route list
            self.routes = insecure_action_routes + self.routes

        for route in self.routes:
            self._config.ir.logger.debug(f"V2VirtualHost {self.name}: finalize: Route {prettyroute(route)}")


    def pretty(self) -> str:
        ctx_name = "-none-"

        if self._tls_context:
            ctx_name = self._tls_context.pretty()

        return f"<VHost {self._hostname} ctx={ctx_name} secure_action={self._action} len(routes)={len(self.routes)}>"

    def verbose_dict(self) -> dict:
        return {
            "name": self.name,
            "_hostname": self._hostname,
            "_action": self._action,
            "_insecure_actions": self._insecure_actions,
            "tls_context": self._tls_context,
            "routes": self.routes,
        }

    @property
    def domains(self) -> List[str]:
        if ((not self.name.endswith("_fstar")) and
            (self._ctx is not None) and
            (self._ctx.hosts is not None) and
            (len(self._ctx.hosts) > 0)):
            return self._ctx.hosts
        else:
            return [self._hostname]


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
        already_existed = (port in self)

        v2listener = self[port]

        if not already_existed:
            v2listener.use_proxy_proto = use_proxy_proto
        elif v2listener.use_proxy_proto != use_proxy_proto:
            raise Exception("listener for port %d has use_proxy_proto %s, requester wants upp %s" %
                            (v2listener.service_port, v2listener.use_proxy_proto, use_proxy_proto))

        return v2listener

class VHostKey(NamedTuple):
    secure: bool
    hostname: str

class V2Listener(dict):
    """V2Listener is the Python type for a gRPC `envoy.api.v2.Listener`
    that happens to contain an HttpConnectionManager in its filter
    chain; therefore serving HTTP `Mappings` rather than
    `TCPMappings`.

    """
    def __init__(self, config: 'V2Config', service_port: int) -> None:
        super().__init__()

        self.config = config
        self.service_port = service_port
        self.name = f"ambassador-listener-{self.service_port}"
        self.use_proxy_proto = False
        self.access_log: List[dict] = []
        self.upgrade_configs: Optional[List[dict]] = None
        self.vhosts: Dict[VHostKey, V2VirtualHost] = {}
        self.first_vhost: Dict[bool, V2VirtualHost] = {}
        self.http_filters: List[dict] = []
        self.listener_filters: List[dict] = []
        self.traffic_direction: str = "UNSPECIFIED"

        self.config.ir.logger.debug(f"V2Listener {self.name}: created")

        # Assemble filters
        for f in self.config.ir.filters:
            v2f: dict = v2filter(f, self.config)

            if v2f:
                self.http_filters.append(v2f)

        # Get Access Log Rules
        for al in self.config.ir.log_services.values():
            access_log_obj: Dict[str,Any] = { "common_config": al.get_common_config() }
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
                'name': 'envoy.file_access_log',
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

            self.config.ir.logger.debug(f"V2Listener {self.name}: using log_format {repr(log_format)}")
            self.access_log.append({
                'name': 'envoy.file_access_log',
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
            self.base_http_config["common_http_protocol_options"] = { 'idle_timeout': "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0) }

        if 'enable_http10' in self.config.ir.ambassador_module:
            self.base_http_config["http_protocol_options"] = { 'accept_http_10': self.config.ir.ambassador_module.enable_http10 }

        if 'preserve_external_request_id' in self.config.ir.ambassador_module:
            self.base_http_config["preserve_external_request_id"] = self.config.ir.ambassador_module.preserve_external_request_id

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

        proper_case = self.config.ir.ambassador_module['proper_case']

        if proper_case:
            proper_case_header: Dict[str,Dict[str,dict]] = {'header_key_format': {'proper_case_words': {}}}
            if 'http_protocol_options' in self.base_http_config:
                self.base_http_config["http_protocol_options"].update(proper_case_header)
            else:
                self.base_http_config["http_protocol_options"] = proper_case_header

    def add_irlistener(self, listener: IRListener) -> None:
        if listener.service_port != self.service_port:
            # This is a problem.
            raise Exception("V2Listener %s: trying to add listener %s on %s:%d??" %
                            (self.name, listener.name, listener.hostname, listener.service_port))

    # The action is Optional, this is not a typo.
    def make_vhost(self, name: str, hostname: str, context: Optional[IRTLSContext],
                   action: Optional[str]) -> V2VirtualHost:
        self.config.ir.logger.debug(f"V2Listener {self.name}: adding VHost {name} for host={hostname}, secure_action={action}")

        secure = action != None
        key = VHostKey(secure=secure, hostname=hostname)
        vhost = self.vhosts.get(key)

        if vhost:
            if ((name != vhost.name) or
                (hostname != vhost._hostname) or
                (context != vhost._ctx) or
                (action != vhost._action)):
                raise Exception(f"V2Listener {self.name}: trying to make vhost(name={name}, hostname={hostname}, context={context}, action={action}) but conflicting "
                                f"vhost(name={vhost.name}, hostname={vhost._hostname}, context={vhost._ctx}, action={vhost._action}) already exists")
            else:
                return vhost

        vhost = V2VirtualHost(config=self.config, listener=self,
                              name=name, hostname=hostname, ctx=context,
                              action=action)
        self.vhosts[key] = vhost

        self.first_vhost.setdefault(secure, vhost)

        return vhost


    def finalize(self) -> None:
        self.config.ir.logger.debug(f"V2Listener {self.name}:  finalize {self.pretty()}")

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

        for key, vhost in self.vhosts.items():
            vhostname = key.hostname
            # Finalize this VirtualHost...
            vhost.finalize()

            # ...then build up the Envoy structures around it.
            filter_chain: Dict[str, Any] = {
                "filter_chain_match": vhost.filter_chain_match,
            }

            if vhost._tls_context:
                filter_chain["tls_context"] = vhost._tls_context
                need_tcp_inspector = True

            http_config = dict(self.base_http_config)
            http_config["route_config"] = {
                "virtual_hosts": []
            }

            http_config["route_config"]["virtual_hosts"].append({
                "name": f"{self.name}-{vhost.name}",
                "domains": vhost.domains,
                "routes": vhost.routes,
            })

            filter_chain["filters"] = [
                {
                    "name": "envoy.http_connection_manager",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager",
                        **http_config
                    }
                }
            ]

            self.filter_chains.append(filter_chain)

        if self.use_proxy_proto:
            self.listener_filters.append({
                'name': 'envoy.listener.proxy_protocol',
                'config': {}
            })

        if need_tcp_inspector:
            self.listener_filters.append({
                'name': 'envoy.listener.tls_inspector',
                'config': {}
            })

    def as_dict(self) -> dict:
        return {
            "name": self.name,
            "address": self.address,
            "filter_chains": self.filter_chains,
            "listener_filters": self.listener_filters,
            "traffic_direction": self.traffic_direction
        }

    def pretty(self) -> dict:
        return { "name": self.name,
                 "port": self.service_port,
                 "use_proxy_proto": self.use_proxy_proto,
                 "vhosts": { k.hostname: v.pretty() for k, v in self.vhosts.items() } }

    @classmethod
    def log_listeners(cls, logger, listeners_by_port) -> None:
        pretty = { k: v.pretty() for k, v in listeners_by_port.items() }

        logger.debug(f"V2Listener.log_listeners: {json.dumps(pretty, sort_keys=True, indent=4)}")

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        """Inspect `config.ir` and `config.routes` in order to populate
        populate `config.listeners`.

        """
        config.listeners = []
        logger = config.ir.logger

        # Step 1.  Handle the listeners that handle HTTP

        # Step 1.1.  Instantiate all V2Listeners and their child V2VirtualHosts

        # OK, so we need to construct one or more V2Listeners, based on our IRListeners.
        # The highest-level thing that defines an Envoy listener is a port, so start
        # with that.
        #
        # This line itself does nothing; the Collection lazily creates V2Listeners on-demand with
        # `V2Listener(config, port_number)`.  As for accessing it, you can think of it as a
        # `Dict[int, V2Listener]`.
        listeners_by_port = V2ListenerCollection(config)

        # Also, in Edge Stack, the magic extremely-low-precedence "/" Mapping is always routed,
        # rather than being redirected. If a user doesn't want this behavior, they can override
        # the Mapping.
        first_irlistener_by_port: Dict[int, IRListener] = {}

        for irlistener in config.ir.listeners:
            if irlistener.service_port not in first_irlistener_by_port:
                first_irlistener_by_port[irlistener.service_port] = irlistener

            logger.debug(f"V2Listener.generate: working on {irlistener.pretty()}")

            # secure action #######################################################################################

            # Grab a new V2Listener for this IRListener...
            listener = listeners_by_port.get(irlistener.service_port, irlistener.use_proxy_proto)
            listener.add_irlistener(irlistener)

            # What VirtualHost hostname are we trying to work with here?
            vhostname = irlistener.hostname or "*"

            # Well, we only want a secure vhost if this irlistener has a TLS context! If it has no context, the
            # listener won't set transport_protocol to tls anyway.
            if irlistener.get('context') is not None:
                listener.make_vhost(name=vhostname,
                                    hostname=vhostname,
                                    context=irlistener.context,
                                    action=irlistener.secure_action)

            # insecure action #####################################################################################

            # An IRListener will always have an insecure action, either "Route, "Redirect", or "Reject".
            assert irlistener.insecure_action is not None

            # There are going to be times when an insecure action will NOT have a port associated with it. If there
            # is a port specified, then we get a listener for that port (and create a listener in the process if
            # need be), or we use the current listener we are working with.
            if (irlistener.insecure_addl_port is not None) and (irlistener.insecure_addl_port > 0):
                # Make sure we have a listener on the right port for this.
                listener = listeners_by_port.get(irlistener.insecure_addl_port, irlistener.use_proxy_proto)

                if irlistener.insecure_addl_port not in first_irlistener_by_port:
                    first_irlistener_by_port[irlistener.insecure_addl_port] = irlistener

            # Now, we are talking about insecure stuff here - this does not require multiple vhosts.
            # Multiple vhosts roughly (very, very roughly) translates to multiple fitler chains, and we don't
            # need that. We don't need separate filter chains with separate "server_names" for "insecure" hosts
            # because server_names is applicable only to SNI for TLS.
            #
            # What we need here instead is ONE insecure vhost (for each port) which has Routes that have
            # HeaderMatchers for these insecure domains.  Which header? ":authority" i.e. the "Host" header.
            #
            # Why, though? Even if we don't need it, it adds a decent chunk of complexity, so why not just
            # create multiple insecure vhosts?  Flynn says that envoy.api.v2.route.VirtualHost.domains only
            # works if you're using TLS, but secretly I suspect that he's mixing it up with
            # envoy.api.v2.listener.FilterChainMatch.server_names.  And checking that doesn't seem to be worth
            # it right now, and would be a rather larger overhaul than I want to include in 1.7.3 anyway.
            vhost = listener.make_vhost(name=f"_insecure-{listener.service_port}",
                                        hostname="*",
                                        context=None,
                                        action=None)
            vhost._insecure_actions[vhostname] = irlistener.insecure_action

            logger.debug(f"V2Listener {listener.name}: final vhosts: {[k.hostname for k in listener.vhosts.keys()]}")

        logger.debug(f"V2Listener.generate: after IRListeners")
        cls.log_listeners(logger, listeners_by_port)

        if config.ir.edge_stack_allowed and (not config.ir.agent_active) and (8080 not in listeners_by_port):
            # If we're running Edge Stack, and we're not an intercept agent, make sure we have
            # a listener on port 8080, so that we have a place to stand for ACME.

            # Check for a listener on the main service port to see if the proxy proto
            # is enabled.
            main_listener = first_irlistener_by_port.get(config.ir.ambassador_module.service_port, None)
            use_proxy_proto = main_listener.use_proxy_proto if main_listener else False

            # Force a listener on 8080 with a VHost for '*' that rejects everything. The ACME
            # hole-puncher will override the reject for ACME, and nothing else will get through.
            logger.debug(f"V2Listener.generate: listeners_by_port has no 8080, forcing Edge Stack listener on 8080")
            listener = listeners_by_port.get(8080, use_proxy_proto)

            # Remember, it is not a bug to have action=None. There is no secure action
            # for this vhost.
            vhost = listener.make_vhost(name="_forced-8080",
                                        hostname="*",
                                        context=None,
                                        action=None)
            vhost._insecure_actions["*"] = "Reject"

        # Make sure that each listener has a '*' vhost for every transport protocol that it listens on.
        for port, listener in listeners_by_port.items():
            logger.warning(f"V2Listener.generate ({port}) {listener.name}: LUKESHU: listener.vhosts.keys(): {listener.vhosts.keys()}")
            for secure in [True, False]:
                if not VHostKey(secure=secure, hostname='*') in listener.vhosts:
                    if secure in listener.first_vhost:
                        # Force the first VHost to '*'. I know, this is a little weird, but it's arguably
                        # the least surprising thing to do in most situations.
                        first_vhost = listener.first_vhost[secure]
                        first_vhost._hostname = '*'
                        first_vhost.name += "_fstar"

        # Step 1.2.  Populate the V2VirtualHosts with Routes

        # OK. We have all the listeners. Time to walk the routes (note that they are already ordered).
        for route in config.routes:
            logger.debug(f"V2Listener.generate: route {prettyroute(route)}...")

            # We need to walk all listeners and all vhosts, and match up the routes with the vhosts.
            for port, listener in listeners_by_port.items():

                for vhostkey, vhost in listener.vhosts.items():
                    logger.debug(f"V2Listener {listener.name}: generating route for vhost {vhost.name}")
                    vhost.maybe_add_route(route)

        # Step 1.3.  Finalize the V2Listeners and V2VirtualHosts

        # OK. Finalize the world.
        for port, listener in listeners_by_port.items():
            listener.finalize()

        logger.debug("V2Listener.generate: after finalize")
        cls.log_listeners(logger, listeners_by_port)

        for k, v in listeners_by_port.items():
            config.listeners.append(v.as_dict())

        # logger.info(f"==== ENVOY LISTENERS ====: {json.dumps(config.listeners, sort_keys=True, indent=4)}")

        # Step 2.  Handle the non-HTTP listeners

        # We need listeners for the TCPMappingGroups too.
        tcplisteners: Dict[str, V2TCPListener] = {}

        for irgroup in config.ir.ordered_groups():
            if not isinstance(irgroup, IRTCPMappingGroup):
                continue

            # OK, good to go. Do we already have a TCP listener binding where this one does?
            group_key = irgroup.bind_to()
            tcplistener = tcplisteners.get(group_key, None)

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
