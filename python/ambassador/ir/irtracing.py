from typing import Optional, TYPE_CHECKING

from .ircluster import IRCluster
from .irresource import IRResource
from ..config import Config
from ..utils import RichStatus

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRTracing(IRResource):
    cluster: Optional[IRCluster]
    service: str
    driver: str
    driver_config: dict
    tag_headers: list
    host_rewrite: Optional[str]
    sampling: dict

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str = "ir.tracing",
                 kind: str = "ir.tracing",
                 name: str = "tracing",
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

        # This "config" is a field on the aconf for the TracingService, not to be confused with the
        # envoyv2 untyped "config" field. We actually use a "typed_config" in the final Envoy
        # config, see envoy/v2/v2tracer.py.
        driver_config = config.get("config", {})
        if driver_config:
            if 'collector_endpoint_version' in driver_config:
                if not driver_config['collector_endpoint_version'] in ['HTTP_JSON_V1', 'HTTP_JSON', 'HTTP_PROTO']:
                    self.post_error(RichStatus.fromError("collector_endpoint_version must be one of 'HTTP_JSON_V1, HTTP_JSON, HTTP_PROTO'"))
                    return False

        # OK, we have a valid config.
        self.sourced_by(config)

        self.service = service
        self.driver = driver
        self.grpc = grpc
        self.cluster = None
        self.driver_config = driver_config
        self.tag_headers = config.get('tag_headers', [])
        self.sampling = config.get('sampling', {})

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
                grpc=self.grpc,
                stats_name=self.get("stats_name", None)
            )
        )

        cluster.referenced_by(self)
        self.cluster = cluster

    def finalize(self):
        self.ir.logger.debug("tracing cluster envoy name: %s" % self.cluster.envoy_name)
        self.driver_config['collector_cluster'] = self.cluster.envoy_name
