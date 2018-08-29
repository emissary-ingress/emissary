from typing import TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRTracing (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.tracing",
                 kind: str="IRTracing",
                 name: str="tracing",
                 **kwargs) -> None:

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            # why do we need a cluster?
            cluster="cluster_ext_tracing",
            timeout_ms=5000,
            hosts={}
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # Let's validate the incoming config
        is_valid = self.validate(aconf)
        self.logger.debug("is tracing config valid: {}".format(is_valid))
        if not is_valid:
            self.post_error(RichStatus.fromError("failed to validate TracingService"))
            return False

        # Now that validations are done, we can dive in

        # Not sure about host_rewrite. The original implementation has code for it, but no documentation.
        # host_rewrite = config.get("host_rewrite")

        self.ir.router_config['start_child_span'] = True

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        config = list(aconf.get_config('tracing_configs').values())[0]
        self.referenced_by(config)

        ir.tracing_config = {
            'cluster_name': self.name,
            'source': config.get('source'),
            'service': config.get('service'),
            'driver': config.get('driver'),
            'driver_config': config.get("config", {}),
            'host_rewrite': config.get("host_rewrite", None)
        }

        cluster = ir.add_cluster(
            IRCluster(
                ir=ir,
                aconf=aconf,
                location=self.location,
                service=ir.tracing_config['service'],
                name=self.cluster,
                host_rewrite=ir.tracing_config['host_rewrite'],
                source=ir.tracing_config['source']
            )
        )

        cluster.add_url(ir.tracing_config['service'])

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

    def validate(self, aconf: Config) -> bool:
        # Some of the validations might go away if JSON Schema is doing the validations, but need to check on that

        config_info = aconf.get_config('tracing_configs')
        if not config_info:
            return False

        self.logger.debug("tracing config: {}".format(config_info))

        configs = config_info.values()
        number_configs = len(configs)
        if number_configs is not 1:
            self.post_error(
                RichStatus.fromError("exactly one TracingService is supported, got {}".format(number_configs)))
            return False

        config = list(configs)[0]

        # This results in False right now
        # source = config.get('source')
        # if not source:
        #     self.post_error(RichStatus.fromError("no source found for TracingService"))
        #     return False

        service = config.get('service')
        if not service:
            self.post_error(RichStatus.fromError("service field is required in TracingService"))
            return False

        driver = config.get('driver')
        if not driver:
            self.post_error(RichStatus.fromError("driver field is required in TracingService"))
            return False

        return True

    def config_dict(self):
        config = {}

        return config
