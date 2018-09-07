from typing import TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irfilter import IRFilter
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRRateLimit (IRFilter):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.ratelimit",
                 kind: str="IRRateLimit",
                 name: str="rate_limit",    # This is a key for Envoy! You can't just change it.
                 **kwargs) -> None:
        # print("IRRateLimit __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, type='decoder'
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        config_info = aconf.get_config("ratelimit_configs")

        if not config_info:
            ir.logger.debug("IRRateLimit: no ratelimit config, bailing")
            # No tracing info. Be done.
            return False

        configs = config_info.values()
        number_configs = len(configs)
        if number_configs is not 1:
            self.post_error(
                RichStatus.fromError("only one RateLimitService is supported, got {}".format(number_configs)))
            return False

        config = list(configs)[0]

        service = config.get("service", None)

        if not service:
            self.post_error(RichStatus.fromError("service is required in RateLimitService",
                                                 module=config))
            return False

        # OK, we have a valid config.
        self.sourced_by(config)

        self.service = service
        self.name = "rate_limit"    # This is a key for Envoy, so we force it, just in case.

        # XXX host_rewrite actually isn't in the schema right now.
        self.host_rewrite = config.get('host_rewrite', None)

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

        self.referenced_by(config)

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        cluster = ir.add_cluster(
            IRCluster(
                ir=ir,
                aconf=aconf,
                location=self.location,
                service=self.service,
                grpc=True,
                host_rewrite=self.get('host_rewrite', None)
            )
        )

        cluster.referenced_by(self)

        grpc_service = ir.add_grpc_service("rate_limit_service", cluster)

        self.cluster = cluster
        self.config = {
            "domain": "ambassador",
            "request_type": "both",
            "timeout_ms": 20
        }
