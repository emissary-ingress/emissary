from ..config import Config
from ..utils import RichStatus
from ..resource import Resource

from .resource import IRResource


class AuthResource (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.auth",
                 kind: str="IRAuth",
                 name: str="ir.auth",
                 **kwargs) -> None:
        print("IRAuth __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            cluster="cluster_ext_auth",
            timeout_ms=5000,
            path_prefix=None,
            allowed_headers=[ ],
            weight=100,
            hosts={},
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

        if self.hosts:
            # If we got here, either we have no errors or it's not a big enough
            # deal to stop work.
            self.logger.info("IRAuth: found some hosts! going active")
            return True
        else:
            self.logger.info("IRAuth: found no hosts! going inactive")
            return False

    def _load_auth(self, module: Resource):
        for key in ['path_prefix', 'timeout_ms', 'cluster']:
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


    # cluster_name = auth_resource.cluster
    # host_rewrite = auth_resource.get('host_rewrite', False)
    #
    # if cluster_name not in self.clusters:
    #     if not cluster_hosts:
    #         cluster_hosts = { '127.0.0.1:5000': ( 100, None ) }
    #
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

        # # Then append the rate-limit filter, because we might rate-limit based on auth headers
        # ratelimit_configs = config.get_config('ratelimit_configs')
        # (ratelimit_filter, ratelimit_grpc_service) = self.module_config_ratelimit(ratelimit_configs)
        # if ratelimit_filter and ratelimit_grpc_service:
        #     self.config['filters'].append(ratelimit_filter)
        #     self.config['grpc_services'].append(ratelimit_grpc_service)
