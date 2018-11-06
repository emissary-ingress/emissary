from typing import Optional, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus
from ..resource import Resource

from .irfilter import IRFilter
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRAuth (IRFilter):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.auth",
                 kind: str="IRAuth",
                 name: str="envoy.ext_authz",
                 **kwargs) -> None:
        # print("IRAuth __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            cluster="cluster_ext_auth",
            timeout_ms=5000,
            path_prefix=None,
            allowed_request_headers=[],
            hosts={},
            **kwargs)

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
                # self.ir.logger.debug("SET CLUSTER %s" % cluster.as_json())

        if cluster_good:
            # self.ir.logger.debug("GOOD CLUSTER %s" % self.cluster.as_json())
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

        # DRY ..
        headers = module.get('allowed_resquest_headers', None)
        if headers:
            allowed_resquest_headers = self.get('allowed_resquest_headers', [])

            for hdr in headers:
                if hdr not in allowed_resquest_headers:
                    allowed_resquest_headers.append(hdr)

            self['allowed_resquest_headers'] = allowed_resquest_headers


        # DRY ..
        auth_headers = module.get('allowed_authorization_headers', None)
        if auth_headers:
            allowed_authorization_headers = self.get('allowed_authorization_headers', [])

            for hdr in auth_headers:
                if hdr not in allowed_authorization_headers:
                    allowed_authorization_headers.append(hdr)

            self['allowed_authorization_headers'] = allowed_authorization_headers
        

        auth_service = module.get("auth_service", None)
        weight = 100    # Can't support arbitrary weights right now.

        if auth_service:
            self.hosts[auth_service] = ( weight, module.get('tls', None), module.location )

    def config_dict(self):
        config = {
            "cluster": self.cluster.name
        }

        for key in [ 'allowed_resquest_headers', 'path_prefix', 'timeout_ms', 'weight' ]:
            if self.get(key, None):
                config[key] = self[key]
        
        # Sets request headers whitelist.
        if self.get('allowed_resquest_headers', []):
            config['allowed_resquest_headers'] = self.allowed_resquest_headers
        else:
            config['allowed_resquest_headers'] = []


        # Sets authorization headers whitelist.
        if self.get('allowed_authorization_headers', []):
            config['allowed_authorization_headers'] = self.allowed_authorization_headers
        else:
            config['allowed_authorization_headers'] = []
            
        # Sets allowed headers normally used in the context of authorization and authentication. Since Envoy uses 
        # set data structure for these lists, we don't care if keys get duplicated by the user's input.
        config['allowed_resquest_headers'].append("authorization")
        config['allowed_resquest_headers'].append("proxy-authorization")
        config['allowed_resquest_headers'].append("user-agent")
        config['allowed_resquest_headers'].append("x-forwarded-for")
        config['allowed_resquest_headers'].append("x-forwarded-host")
        config['allowed_resquest_headers'].append("x-forwarded-proto")
        config['allowed_resquest_headers'].append("cookie")

        config['allowed_authorization_headers'].append("www-authenticate")
        config['allowed_authorization_headers'].append("proxy-Authenticate")
        config['allowed_authorization_headers'].append("location")

        return config
