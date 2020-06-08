from typing import Optional, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRTracing (IRResource):
    cluster: Optional[IRCluster]
    service: str
    driver: str
    driver_config: dict
    tag_headers: list
    host_rewrite: Optional[str]

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.tracing",
                 kind: str="ir.tracing",
                 name: str="tracing",
                 namespace: Optional[str] = None,
                 **kwargs) -> None:
        del kwargs  # silence unused-variable warning

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, namespace=namespace
        )
        self.cluster = None

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # Some of the validations might go away if JSON Schema is doing the validations, but need to check on that

        config_info = aconf.get_config('tracing_configs')

        if not config_info:
            ir.logger.debug("IRTracing: no tracing config, bailing")
            # No tracing info. Be done.
            return False

        configs = config_info.values()
        number_configs = len(configs)
        if number_configs != 1:
            self.post_error(
                RichStatus.fromError("exactly one TracingService is supported, got {}".format(number_configs),
                                     module=aconf))
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

        self.namespace = config.get("namespace", self.namespace)

        grpc = False
        if driver == "lightstep":
            grpc = True

        if driver == "datadog":
            driver = "envoy.tracers.datadog"

        # OK, we have a valid config.
        self.sourced_by(config)

        self.service = service
        self.driver = driver
        self.grpc = grpc
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
                parent_ir_resource=self,
                location=self.location,
                service=self.service,
                host_rewrite=self.get('host_rewrite', None),
                marker='tracing',
                grpc=self.grpc
            )
        )

        cluster.referenced_by(self)
        self.cluster = cluster

    def finalize(self):
        self.ir.logger.debug("tracing cluster name: %s" % self.cluster.name)
        self.driver_config['collector_cluster'] = self.cluster.name
