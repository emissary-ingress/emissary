from typing import Optional, TYPE_CHECKING
from typing import cast as typecast

from ..config import Config
from ..utils import RichStatus
from ..resource import Resource

from .irfilter import IRFilter
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRAuth (IRFilter):
    cluster: Optional[IRCluster]

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.auth",
                 kind: str="IRAuth",
                 name: str="extauth",
                 type: Optional[str] = "decoder",
                 **kwargs) -> None:


        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            cluster=None,
            timeout_ms=5000,
            connect_timeout_ms=3000,
            path_prefix=None,
            api_version=None,
            allowed_headers=[],
            allowed_request_headers=[],
            allowed_authorization_headers=[],
            hosts={},
            type=type,
            **kwargs)

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        module_info = aconf.get_module("authentication")

        if module_info:
            self._load_auth(module_info)

        config_info = aconf.get_config("auth_configs")

        if config_info:
            for config in config_info.values():
                self._load_auth(config)

        if not self.hosts:
            self.logger.debug("IRAuth: found no hosts! going inactive")
            return False

        self.logger.debug("IRAuth: found some hosts! going active")

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        cluster_hosts = self.get('hosts', { '127.0.0.1:5000': ( 100, None, '-internal-' ) })

        self.cluster = None
        cluster_good = False

        for service, params in cluster_hosts.items():
            weight, grpc, ctx_name, location = params

            self.logger.debug("IRAuth: svc %s, weight %s, grpc %s, ctx_name %s, location %s" %
                              (service, weight, grpc, ctx_name, location))

            cluster = IRCluster(
                ir=ir, aconf=aconf, location=location,
                service=service,
                host_rewrite=self.get('host_rewrite', False),
                ctx_name=ctx_name,
                grpc=grpc,
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
            ir.add_cluster(typecast(IRCluster, self.cluster))
            self.referenced_by(typecast(IRCluster, self.cluster))

    def _load_auth(self, module: Resource):
        if self.location == '--internal--':
            self.sourced_by(module)

        for key in [ 'path_prefix', 'timeout_ms', 'cluster', 'allow_request_body', 'proto' ]:
            value = module.get(key, None)

            if value:
                previous = self.get(key, None)

                if previous and (previous != value):
                    # Don't use self.post_error() here, since we need to explicitly override the
                    # resource. And don't use self.ir.post_error, since our module isn't an IRResource.
                    self.ir.aconf.post_error(
                        "AuthService cannot support multiple %s values; using %s" % (key, previous),
                        resource=module
                    )
                else:
                    self[key] = value

            self.referenced_by(module)

        self["allow_request_body"] = module.get("allow_request_body", False)
        self["include_body"] = module.get("include_body", None)
        self["api_version"] = module.get("apiVersion", None)
        self["proto"] = module.get("proto", "http")
        self["timeout_ms"] = module.get("timeout_ms", 5000)
        self["connect_timeout_ms"] = module.get("connect_timeout_ms", 3000)
        self.__to_header_list('allowed_headers', module)
        self.__to_header_list('allowed_request_headers', module)
        self.__to_header_list('allowed_authorization_headers', module)

        # Required fields check.
        if self["api_version"] == None:
            self.post_error(RichStatus.fromError("AuthService config requires apiVersion field"))

        if self["api_version"] == "ambassador/v1" and self["proto"] == None:
            self.post_error(RichStatus.fromError("AuthService v1 config requires proto field."))

        if self.get("include_body") and self.get("allow_request_body"):
            self.post_error('AuthService ignoring allow_request_body since include_body is present')
            del(self['allow_request_body'])

        auth_service = module.get("auth_service", None)
        weight = 100    # Can't support arbitrary weights right now.

        if auth_service:
            is_grpc = True if self["proto"] == "grpc" else False
            self.hosts[auth_service] = ( weight, is_grpc, module.get('tls', None), module.location)

    # This method is only used by v1listener.
    def config_dict(self):
        config = {
            "cluster": self.cluster.name
        }

        for key in [ 'allowed_headers', 'path_prefix', 'timeout_ms', 'weight', 'connect_timeout_ms' ]:
            if self.get(key, None):
                config[key] = self[key]

        if self.get('allowed_headers', []):
            config['allowed_headers'] = self.allowed_headers

        return config

    def __to_header_list(self, list_name, module):
        headers = module.get(list_name, None)

        if headers:
            allowed_headers = self.get(list_name, [])

            for hdr in headers:
                if hdr not in allowed_headers:
                    allowed_headers.append(hdr.lower())

            self[list_name] = allowed_headers
