from typing import Optional, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus
from ..resource import Resource

from .irfilter import IRFilter
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


# Static allowed request headers normally used in the context of 
# authorization and authentication. 
AllowedRequestHeader = set([
    'Authorization',
    'Cookie',
    'From',
    'Host',
    'Proxy-Authorization',
    'User-Agent',
    'X-Forwarded-For',
    'X-Forwarded-Host',
    'X-Forwarded-Proto'
])

# Static allowed authorization headers normally used in the context 
# of authorization and authentication.
AllowedAuthorizationHeaders = set([
    'Location'
    'Proxy-Authenticate',
    'Set-Cookie',
    'WWW-Authenticate'
])

class IRAuth (IRFilter):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.auth",
                 kind: str="IRAuth",
                 name: str="envoy.ext_authz",
                 **kwargs) -> None:
        
        super().__init__(
            ir = ir, aconf = aconf, rkey = rkey, kind = kind, name = name,
            cluster = "cluster_ext_auth",
            timeout_ms = None,
            path_prefix = None,
            allowed_request_headers = AllowedRequestHeader,
            allowed_authorization_headers = AllowedAuthorizationHeaders,
            hosts = {}, **kwargs)

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        module_info = aconf.get_module("authentication")

        if module_info:
            self._load_auth(module_info)

        config_info = aconf.get_config("auth_configs")

        if config_info:
            self.logger.debug("auth_configs: %s" % config_info)
            for config in config_info.values():
                self._load_auth(config)

        if not self.hosts:
            self.logger.info("IRAuth: found no hosts! going inactive")
            return False

        self.logger.info("IRAuth: found some hosts! going active")

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        cluster_hosts = self.get('hosts', { '127.0.0.1:5000': ( 100, None ) })

        self.cluster = None

        self.ir.logger.debug("AUTH ADD_MAPPINGS: %s" % self.as_json())

        for service, params in cluster_hosts.items():
            weight, ctx_name, location = params

            cluster = IRCluster(
                ir=ir, 
                aconf=aconf, 
                location=location,
                service=service,
                host_rewrite=self.get('host_rewrite', False),
                ctx_name=ctx_name,
                marker='extauth'
            )

            cluster.referenced_by(self)

            cluster_good = True

            if self.cluster:
                if not self.cluster.merge(cluster):
                    self.post_error(RichStatus.fromError("auth canary %s can only change service!" % cluster.name))
                    cluster_good = False
            else:
                self.cluster = cluster

        if cluster_good:
            ir.add_cluster(self.cluster)
            self.referenced_by(self.cluster)

    def _load_auth(self, module: Resource):
        if self.location == '--internal--':
            self.sourced_by(module)

        for key in [ 'path_prefix', 'timeout_ms', 'cluster', 'auth_service' ]:
            value = module.get(key, None)

            if value:
                previous = self.get(key, None)

                if previous and (previous != value):
                    errstr = (
                        "AuthService cannot support multiple %s values; using %s" %
                        (key, previous)
                    )

                    self.post_error(RichStatus.fromError(errstr, resource=module))
                else:
                    self[key] = value

            self.referenced_by(module)
        
        self.to_header_list("allowed_request_headers", module)
        self.to_header_list("allowed_authorization_headers", module)

        self["timeout_ms"] = module.get("timeout_ms", "3s")

        auth_service = module.get("auth_service", None)
        weight = 100    # Can't support arbitrary weights right now.
        if auth_service:
            self.hosts[auth_service] = ( weight, module.get('tls', None), module.location )

    def config_dict(self):
        config = {
            "cluster": self.cluster.name
        }

        for key in [ 'allowed_resquest_headers', 'allowed_authorization_headers', 'path_prefix', 'timeout_ms', 'weight' ]:
            if self.get(key, None):
                config[key] = self[key]
        
        # Sets request headers whitelist.
        if self.get('allowed_resquest_headers', []):
            config['allowed_resquest_headers'] = self.allowed_resquest_headers

        # Sets authorization headers whitelist.
        if self.get('allowed_authorization_headers', []):
            config['allowed_authorization_headers'] = self.allowed_authorization_headers

        return config

    def to_header_list(self, list_name, module):
        joined_list = set(self.get(list_name, [])).union(module.get(list_name, []))
        self[list_name] = list(joined_list)