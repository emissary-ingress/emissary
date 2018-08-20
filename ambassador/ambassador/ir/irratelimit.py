from typing import TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRRateLimit (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.ratelimit",
                 kind: str="IRRateLimit",
                 name: str="ir.ratelimit",
                 **kwargs) -> None:
        # print("IRRateLimit __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            cluster="cluster_ext_ratelimit",
            timeout_ms=5000,
            hosts={}
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        config_info = aconf.get_config("ratelimit_configs")

        if not config_info:
            return False

        assert(len(config_info) > 0)    # really rank paranoia on my part...

        self.logger.debug("ratelimit_configs: %s" % config_info)

        all_configs = config_info.values()
        config = all_configs[0]

        if len(all_configs) > 1:
            self.post_error(RichStatus.fromError("only one RateLimitService is supported",
                                                 module=config))
            return False

        rlservice = config.get("service", None)

        if not rlservice:
            self.post_error(RichStatus.fromError("service is required in RateLimitService",
                                                 module=config))
            return False

        self.referenced_by(config)

        # host_rewrite = config.get("host_rewrite", None)
        #
        # cluster_name = "cluster_ext_ratelimit"
        # filter_config = {
        #     "domain": "ambassador",
        #     "request_type": "both",
        #     "timeout_ms": 20
        # }
        # grpc_service = SourcedDict(
        #     name="rate_limit_service",
        #     cluster_name=cluster_name
        # )
        #
        # first_source = sources.pop(0)
        #
        # filter = SourcedDict(
        #     _source=first_source,
        #     type="decoder",
        #     name="rate_limit",
        #     config=filter_config
        # )
        #
        # if cluster_name not in self.clusters:
        #     # (svc, url, originate_tls, otls_name) = self.service_tls_check(cluster_hosts, None, host_rewrite)
        #     (_, url, _, _) = self.service_tls_check(cluster_hosts, None, host_rewrite)
        #     self.add_intermediate_cluster(first_source, cluster_name,
        #                                   'extratelimit', [url],
        #                                   type="strict_dns", lb_type="round_robin",
        #                                   grpc=True, host_rewrite=host_rewrite)

        return True
