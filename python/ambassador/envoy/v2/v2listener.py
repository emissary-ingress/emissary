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
from typing import Any, Dict, List, NamedTuple, Optional, Tuple, TYPE_CHECKING
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
                 action: Optional[str], insecure_action: Optional[str]) -> None:
        super().__init__()

        self._config = config
        self._listener = listener
        self.name = name
        self._hostname = hostname
        self._ctx = ctx
        self._action = action
        self._insecure_action = insecure_action
        # vhost._domains gets populated after self.__init__() but before self.finalize()
        self._domains: Dict[str, List[DictifiedV2Route]] = {}

        self._tls_context = V2TLSContext(ctx)
        self.routes: List[DictifiedV2Route] = []

    def punch_acme_in_routes(self, route_list: List[DictifiedV2Route]):
        for route in route_list:
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

        route_list.insert(0, {
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

    def generate_redirect_route(self, route: V2Route) -> dict:
        """
        Generates a cleaned up redirect route ready to be put in Envoy config.
        :param route: V2Route from which redirect route will be created.
        :return: redirect route
        """
        redirect_route = self.generate_insecure_route(route)
        redirect_route.pop("route", None)
        redirect_route["redirect"] = {
            "https_redirect": True
        }
        return redirect_route

    # maybe_add_route inspects a V2Route and decides how and where it
    # fits in the supplied vhost (if at all).  Depending on the route
    # being insecure, secure, redirect, etc it configures and adds it
    # to the vhost.
    def maybe_add_route(self, c_route: V2Route, action: str):

        logger = self._config.ir.logger
        edge_stack_allowed = self._config.ir.edge_stack_allowed
        final_route: Optional[DictifiedV2Route] = None

        logger.debug(f"V2VirtualHost {self.name}: considering route={c_route} action={action} where edge_stack_allowed={edge_stack_allowed}")

        # If this an SNI route, remember the host[s] to which it pertains.
        route_hosts = set(c_route.get('_sni', {}).get('hosts', []))

        # Remember, also, if a precedence was set.
        route_precedence = c_route.get('_precedence', None)

        # Now, we have basic information of the incoming route, vhost and
        # listener. First, let's see if we need to change the "action" for
        # this route and do some sanity checks.
        if c_route["match"].get("prefix", None) == "/.well-known/acme-challenge/":
            # We need to be sure to route ACME challenges, no matter what else is going
            # on (this is the infamous ACME hole-puncher mentioned everywhere).
            logger.debug(f"V2VirtualHost {self.name}: force Route for ACME challenge")
            action = "Route"

            # We have to force the correct route entry, too, just in case. (Note that right now,
            # the user can't create a Mapping that forces redirection. When they can do this
            # per-Mapping, well, really, we can't force them to not redirect if they explicitly
            # ask for it, and that'll be OK.)

        elif route_hosts and (self._hostname != '*') and (self._hostname not in route_hosts):
            # Drop this because the host is mismatched.
            logger.debug(
                f"V2VirtualHost {self.name}: force Reject (rhosts {sorted(route_hosts)})")
            action = "Reject"

        elif (edge_stack_allowed and
              (route_precedence == -1000000) and
              (c_route["match"].get("safe_regex", {}).get("regex", None) == "^/$")):
            logger.debug(
                f"V2VirtualHost {self.name}: force Route for fallback Mapping")
            action = "Route"

            # Force the actual route entry, instead of using the redirect_route, too.
            # (If the user overrides the fallback with their own route at precedence -1000000,
            # uh.... y'know what, on their own head be it.)

        # Now that all we have configured the right action for this route,
        # let's generate cleaned up routes for secure/insecure routes based
        # on the action.
        if self._action is not None:
            final_route = self.generate_secure_route(c_route)
        else:
            if action == "Redirect":
                logger.debug(f"V2VirtualHost {self.name}: generating redirect route for {dict(c_route)}")
                final_route = self.generate_redirect_route(c_route)

            elif action is not None:
                logger.debug(f"V2VirtualHost {self.name}: generating insecure route for {dict(c_route)}")
                final_route = self.generate_insecure_route(c_route)

            else:
                # Wait, what? This is an insecure route but with no action? Can't be right!
                # Anyway, final_route remains None in this case.
                logger.debug(f"V2VirtualHost {self.name}: no route generated for insecure route "
                             f"{dict(c_route)} because no insecure action is specified")

        if action != 'Reject':
            logger.debug(
                f"V2VirtualHost {self.name}: Accept as {action}")

            # Populate the domains for insecure routes
            if self._action is None:
                for domain in self._domains:
                    logger.debug(f"V2VirtualHost {self.name}: adding route={dict(final_route)} to domain={domain}")
                    self._domains[domain].append(final_route)

            self.routes.append(final_route)
        else:
            logger.debug(f"V2VirtualHost {self.name}: Drop")

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

        # If we're on Edge Stack and we're not an intercept agent, punch a hole for ACME
        # challenges, for every listener.
        if self._config.ir.edge_stack_allowed and not self._config.ir.agent_active:
            # Punch ACME hole in routes
            self.punch_acme_in_routes(self.routes)

            # Punch ACME hole in domains
            for domain in self._domains:
                self.punch_acme_in_routes(self._domains[domain])

        for route in self.routes:
            self._config.ir.logger.debug(f"V2VirtualHost {self.name}: finalize: Route {prettyroute(route)}")

    def pretty(self) -> str:
        ctx_name = "-none-"

        if self._tls_context:
            ctx_name = self._tls_context.pretty()

        return f"<VHost {self._hostname} ctx={ctx_name} secure_action={self._action} insecure_action={self._insecure_action} len(routes)={len(self.routes)}>"

    def verbose_dict(self) -> dict:
        return {
            "name": self.name,
            "_hostname": self._hostname,
            "_action": self._action,
            "_insecure_action": self._insecure_action,
            "tls_context": self._tls_context,
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

class VHostKey(NamedTuple):
    secure: bool
    hostname: str

class V2Listener(dict):
    def __init__(self, config: 'V2Config', service_port: int) -> None:
        super().__init__()

        self.config = config
        self.service_port = service_port
        self.name = f"ambassador-listener-{self.service_port}"
        self.use_proxy_proto = False
        self.access_log: List[dict] = []
        self.upgrade_configs: Optional[List[dict]] = None
        self.vhosts: Dict[VHostKey, V2VirtualHost] = {}
        self.first_vhost: Optional[V2VirtualHost] = None
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

    # Weirdly, the action is optional but the insecure_action is not. This is not a typo.
    def make_vhost(self, name: str, hostname: str, context: Optional[IRTLSContext],
                   action: Optional[str], insecure_action: str) -> V2VirtualHost:
        self.config.ir.logger.debug(f"V2Listener {self.name}: adding VHost {name} for host={hostname}, secure_action={action}, insecure_action={insecure_action}")

        key = VHostKey(secure=action != None, hostname=hostname)
        vhost = self.vhosts.get(key)

        if vhost:
            if ((name != vhost.name) or
                (hostname != vhost._hostname) or
                (context != vhost._ctx) or
                (action != vhost._action) or
                (insecure_action != vhost._insecure_action)):
                raise Exception(f"V2Listener {self.name}: trying to make vhost(name={name}, hostname={hostname}, context={context}, action={action}) but conflicting "
                                f"vhost(name={vhost.name}, hostname={vhost._hostname}, context={vhost._ctx}, action={vhost._action}) already exists")
            else:
                return vhost

        vhost = V2VirtualHost(config=self.config, listener=self,
                              name=name, hostname=hostname, ctx=context,
                              action=action, insecure_action=insecure_action)
        self.vhosts[key] = vhost

        if not self.first_vhost:
            self.first_vhost = vhost

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

            if vhost._hostname == "*":
                domains: List[str] = [vhost._hostname]
            else:
                if vhost._ctx is not None and vhost._ctx.hosts is not None and len(vhost._ctx.hosts) > 0:
                    domains = vhost._ctx.hosts
                else:
                    domains = [vhost._hostname]

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

            if len(vhost._domains) is 0:
                http_config["route_config"]["virtual_hosts"].append({
                    "name": f"{self.name}-{vhost.name}",
                    "domains": domains,
                    "routes": vhost.routes
                    })
            else:
                # vhost._hostname will *not* be used here because
                # vhost._domains is present
                for domain, routes in vhost._domains.items():
                    http_config["route_config"]["virtual_hosts"].append(
                        {
                            "name": f"{self.name}-{vhost.name}-{domain}",
                            "domains": [domain],
                            "routes": routes
                        }
                    )

                # HACK! HACK! HACK! This should go away whenever we turn the first_vhost behavior off.
                # This is essentially turning first domain's behavior into the wildcard behavior. All unknown hosts
                # get this behavior.
                http_config["route_config"]["virtual_hosts"][0]["domains"] = ["*"]

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
    def dump_listeners(cls, logger, listeners_by_port) -> None:
        pretty = { k: v.pretty() for k, v in listeners_by_port.items() }

        logger.debug(f"V2Listener.dump_listeners: {json.dumps(pretty, sort_keys=True, indent=4)}")

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.listeners = []
        logger = config.ir.logger

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

            logger.debug(f"V2Listener.generate: working on {irlistener.pretty()}")

            # Grab a new V2Listener for this IRListener...
            listener = listeners_by_port.get(irlistener.service_port, irlistener.use_proxy_proto)
            listener.add_irlistener(irlistener)

            # What VirtualHost hostname are we trying to work with here?
            vhostname = irlistener.hostname or "*"

            # Well, we only want a secure vhost if this irlistener has a context! If it has no context, the listener
            # won't set transport_protocol to tls anyway.
            if irlistener.get('context') is not None:
                listener.make_vhost(name=vhostname,
                                    hostname=vhostname,
                                    context=irlistener.context,
                                    action=irlistener.secure_action,
                                    insecure_action=irlistener.insecure_action)

            # An irlistener will always have an insecure action, either Route or Redirect
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
            # What we need here instead is ONE insecure vhost which header matches for these insecure domains.
            # Which header? ":authority" i.e. the "Host" header.
            #
            # We have one insecure vhost for every port. All insecure routes (route or redirect or xxx) will be
            # appended to this vhost. If this vhost exists, then make_vhost will get it - else it will create one.
            #
            # We're going to populate all the hostnames in vhost._domains which going to put all of these in
            # virtual_hosts.domains matches on the Host header of the incoming request. Even though it also takes
            # wildcard entries, exact matches are preferred over wildcard matches. This means that
            # www.foo.com > *.foo.com > *.com > *
            #
            # Keep in mind that this name= will *not* be used, it will be overridden by
            # `vhost._domains`. We are only using this for identification purposes.
            vhost = listener.make_vhost(name=f"_insecure-{listener.service_port}",
                                        hostname="*",
                                        context=None,
                                        action=None,
                                        insecure_action=irlistener.insecure_action)
            vhost._domains.setdefault(vhostname, [])

            logger.debug(f"V2Listener {listener.name}: final vhosts: {[k.hostname for k in listener.vhosts.keys()]}")

        logger.debug(f"V2Listener.generate: after IRListeners")
        cls.dump_listeners(logger, listeners_by_port)

        # Make sure that each listener has a '*' vhost.
        for port, listener in listeners_by_port.items():
            for secure in [True, False]:
                if not VHostKey(secure=secure, hostname='*') in listener.vhosts:
                    # Force the first VHost to '*'. I know, this is a little weird, but it's arguably
                    # the least surprising thing to do in most situations.
                    assert listener.first_vhost
                    first_vhost = listener.first_vhost
                    first_vhost._hostname = '*'
                    first_vhost.name += "_fstar"

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
                logger.debug(f"V2Listener.generate: listeners_by_port has no 8080, forcing Edge Stack listener on 8080")
                listener = listeners_by_port.get(8080, use_proxy_proto)

                # Remember, it is not a bug to have action=None. There is no secure action
                # for this vhost.
                vhost = listener.make_vhost(name="_forced-8080",
                                            hostname="*",
                                            context=None,
                                            action=None,
                                            insecure_action='Reject')

        # OK. We have all the listeners. Time to walk the routes (note that they are already ordered).
        for route in config.routes:
            logger.debug(f"V2Listener.generate: route {prettyroute(route)}...")

            # We need to walk all listeners and all vhosts, and match up the routes with the vhosts.
            for port, listener in listeners_by_port.items():

                for vhostkey, vhost in listener.vhosts.items():
                    if vhost._insecure_action is not None:
                        # insecure
                        logger.debug(f"V2Listener {listener.name}: generating insecure route for vhost {vhost._hostname}: action: {vhost._insecure_action}")
                        vhost.maybe_add_route(route, vhost._insecure_action)
                    if vhost._action is not None:
                        # secure
                        logger.debug(f"V2Listener {listener.name}: generating secure route for vhost {vhost._hostname}: action: {vhost._action}")
                        vhost.maybe_add_route(route, vhost._action)

        # OK. Finalize the world.
        for port, listener in listeners_by_port.items():
            listener.finalize()

        logger.debug("V2Listener.generate: after finalize")
        cls.dump_listeners(logger, listeners_by_port)

        for k, v in listeners_by_port.items():
            config.listeners.append(v.as_dict())

        # logger.info(f"==== ENVOY LISTENERS ====: {json.dumps(config.listeners, sort_keys=True, indent=4)}")

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
