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
                 name: str="extauth",
                 type: Optional[str] = "decoder",
                 **kwargs) -> None:
        # print("IRAuth __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            cluster="cluster_ext_auth",
            timeout_ms=5000,
            path_prefix=None,
            allowed_headers=[],
            hosts={},
            type=type,
            **kwargs)

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        module_info = aconf.get_module("authentication")

        if module_info:
            self._load_auth(module_info)

        config_info = aconf.get_config("auth_configs")

        if config_info:
            # self.logger.debug("auth_configs: %s" % auth_configs)
            for config in config_info.values():
                self._load_auth(config)

        if not self.hosts:
            self.logger.info("IRAuth: found no hosts! going inactive")
            return False

        self.logger.info("IRAuth: found some hosts! going active")

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        cluster_name = self.cluster
        cluster_hosts = self.get('hosts', { '127.0.0.1:5000': ( 100, None ) })

        cluster = None

        # print("AUTH ADD_MAPPINGS: %s" % self.as_json())

        for service, params in cluster_hosts.items():
            weight, ctx_name = params

            if not cluster:
                cluster_args = {
                    'name': cluster_name,
                    'service': service,
                    'host_rewrite': self.get('host_rewrite', False),
                    'ctx_name': ctx_name,
                    # 'grpc': self.get('grpc', False)
                }

                cluster = ir.add_cluster(IRCluster(ir=ir, aconf=aconf, location=self.location, **cluster_args))
                cluster.referenced_by(self)
            else:
                cluster.add_url(service)
                cluster.referenced_by(self)

        self.cluster = cluster
        self.referenced_by(cluster)

        #     urls = []
        #     protocols = {}
        #
        #     for svc in sorted(cluster_hosts.keys()):
        #         _, tls_context = cluster_hosts[svc]
        #
        #         (svc, url, originate_tls, otls_name) = self.service_tls_check(svc, tls_context, host_rewrite)
        #
        #         if originate_tls:
        #             protocols['https'] = True
        #         else:
        #             protocols['http'] = True
        #
        #         if otls_name:
        #             filter_config['cluster'] = cluster_name + "_" + otls_name
        #             cluster_name = filter_config['cluster']
        #
        #         urls.append(url)
        #
        #     if len(protocols.keys()) != 1:
        #         raise Exception("auth config cannot try to use both HTTP and HTTPS")
        #
        #     self.add_intermediate_cluster(first_source, cluster_name,
        #                                   'extauth', urls,
        #                                   type="strict_dns", lb_type="round_robin",
        #                                   originate_tls=originate_tls, host_rewrite=host_rewrite)
        #
        # name = "internal_%s_probe_mapping" % name

    def _load_auth(self, module: Resource):
        if self.location == '--internal--':
            self.sourced_by(module)

        for key in [ 'path_prefix', 'timeout_ms', 'cluster' ]:
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

        headers = module.get('allowed_headers', None)

        if headers:
            allowed_headers = self.get('allowed_headers', [])

            for hdr in headers:
                if hdr not in allowed_headers:
                    allowed_headers.append(hdr)

            self['allowed_headers'] = allowed_headers

        auth_service = module.get("auth_service", None)
        # weight = module.get("weight", 100)
        weight = 100    # Can't support arbitrary weights right now.

        if auth_service:
            self.hosts[auth_service] = ( weight, module.get('tls', None) )

    def config_dict(self):
        config = {
            "cluster": self.cluster.name
        }

        for key in [ 'allowed_headers', 'path_prefix', 'timeout_ms', 'weight' ]:
            if self.get(key, None):
                config[key] = self[key]

        if self.get('allowed_headers', []):
            config['allowed_headers'] = self.allowed_headers

        return config
