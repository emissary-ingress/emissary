from typing import Optional, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRTracing (IRResource):
    cluster: Optional[IRCluster]

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.tracing",
                 kind: str="ir.tracing",
                 name: str="tracing",
                 **kwargs) -> None:

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name
        )
        self.cluster = None

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # Some of the validations might go away if JSON Schema is doing the validations, but need to check on that

        config_info = aconf.get_config('tracing_configs')

        if not config_info:
            # No tracing info. Be done.
            return False

        configs = config_info.values()
        number_configs = len(configs)
        if number_configs is not 1:
            self.post_error(
                RichStatus.fromError("exactly one TracingService is supported, got {}".format(number_configs)))
            return False

        config = list(configs)[0]

        service = config.get('service')
        if not service:
            self.post_error(RichStatus.fromError("service field is required in TracingService"))
            return False

        driver = config.get('driver')
        if not driver:
            self.post_error(RichStatus.fromError("driver field is required in TracingService"))
            return False

        # OK, we have a valid config.
        self.sourced_by(config)

        self.service = service
        self.driver = driver
        self.cluster = None
        self.driver_config = config.get("config", {})
        self.tag_headers = config.get('tag_headers', [])

        # XXX host_rewrite actually isn't in the schema right now.
        self.host_rewrite = config.get('host_rewrite', None)

        # Remember that the config references us.
        self.referenced_by(config)

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        cluster = ir.add_cluster(
            IRCluster(
                ir=ir,
                aconf=aconf,
                location=self.location,
                service=self.service,
                host_rewrite=self.get('host_rewrite', None)
            )
        )

        cluster.referenced_by(self)
        self.cluster = cluster

        self.driver_config['collector_cluster'] = cluster.name

        # if not ir.add_to_primary_listener(tracing=True):
        #     raise Exception("Failed to update primary listener with tracing config")

        # if tracing_config:
        #     for config in tracing_config.values():
        #         sources.append(config['_source'])
        #         cluster_hosts = config.get("service", None)
        #         driver = config.get("driver", None)
        #         driver_config = config.get("config", {})
        #         host_rewrite = config.get("host_rewrite", None)
        #  if not cluster_hosts or not sources:
        #     return
        #  cluster_name = "cluster_ext_tracing"
        #  first_source = sources.pop(0)
        #  if cluster_name not in self.envoy_clusters:
        #     (svc, url, originate_tls, otls_name) = self.service_tls_check(cluster_hosts, None, host_rewrite)
        #     self.add_intermediate_cluster(first_source, cluster_name,
        #                                   'exttracing', [url],
        #                                   type="strict_dns", lb_type="round_robin",
        #                                   host_rewrite=host_rewrite)
        #  driver_config['collector_cluster'] = cluster_name
        # tracing = SourcedDict(
        #     _source=first_source,
        #     driver=driver,
        #     config=driver_config,
        #     cluster_name=cluster_name
        # )
        # self.envoy_config['tracing'] = tracing
