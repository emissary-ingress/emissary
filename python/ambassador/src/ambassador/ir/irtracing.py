from typing import TYPE_CHECKING, Optional

from ..config import Config
from ..utils import RichStatus
from .ircluster import IRCluster
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRTracing(IRResource):
    cluster: Optional[IRCluster]
    service: str
    driver: str
    driver_config: dict
    # TODO: tag_headers is deprecated and should be removed once migrated to CRD v3
    tag_headers: list
    custom_tags: list
    host_rewrite: Optional[str]
    sampling: dict

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.tracing",
        kind: str = "ir.tracing",
        name: str = "tracing",
        namespace: Optional[str] = None,
        **kwargs,
    ) -> None:
        del kwargs  # silence unused-variable warning

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, namespace=namespace
        )
        self.cluster = None

    def setup(self, ir: "IR", aconf: Config) -> bool:
        # Some of the validations might go away if JSON Schema is doing the validations, but need to check on that

        config_info = aconf.get_config("tracing_configs")

        if not config_info:
            ir.logger.debug("IRTracing: no tracing config, bailing")
            # No tracing info. Be done.
            return False

        configs = config_info.values()
        number_configs = len(configs)
        if number_configs != 1:
            self.post_error(
                RichStatus.fromError(
                    "exactly one TracingService is supported, got {}".format(
                        number_configs
                    ),
                    module=aconf,
                )
            )
            return False

        config = list(configs)[0]

        service = config.get("service")
        if not service:
            self.post_error(
                RichStatus.fromError("service field is required in TracingService")
            )
            return False

        driver = config.get("driver")
        if not driver:
            self.post_error(
                RichStatus.fromError("driver field is required in TracingService")
            )
            return False

        self.namespace = config.get("namespace", self.namespace)

        grpc = False

        if driver == "lightstep":
            self.post_error(
                RichStatus.fromError(
                    "as of v3.4+ the 'lightstep' driver is no longer supported in the TracingService, please see docs for migration options"
                )
            )
            return False
        if driver == "opentelemetry":
            ir.logger.warning(
                "The OpenTelemetry tracing driver is work-in-progress. Functionality is incomplete and it is not intended for production use. This extension has an unknown security posture and should only be used in deployments where both the downstream and upstream are trusted."
            )
            grpc = True

        if driver == "datadog":
            driver = "envoy.tracers.datadog"

        # This "config" is a field on the aconf for the TracingService, not to be confused with the
        # envoyv2 untyped "config" field. We actually use a "typed_config" in the final Envoy
        # config, see envoy/v2/v2tracer.py.
        driver_config = config.get("config", {})

        if driver == "zipkin":
            # fill zipkin defaults
            if not driver_config.get("collector_endpoint"):
                driver_config["collector_endpoint"] = "/api/v2/spans"
            if not driver_config.get("collector_endpoint_version"):
                driver_config["collector_endpoint_version"] = "HTTP_JSON"
            if "trace_id_128bit" not in driver_config:
                # Make 128-bit traceid the default
                driver_config["trace_id_128bit"] = True
            # validate
            if driver_config["collector_endpoint_version"] not in [
                "HTTP_JSON",
                "HTTP_PROTO",
            ]:
                self.post_error(
                    RichStatus.fromError(
                        "collector_endpoint_version must be one of HTTP_JSON, HTTP_PROTO'"
                    )
                )
                return False

        # OK, we have a valid config.
        self.sourced_by(config)

        self.service = service
        self.driver = driver
        self.grpc = grpc
        self.cluster = None
        self.driver_config = driver_config
        self.tag_headers = config.get("tag_headers", [])
        self.custom_tags = config.get("custom_tags", [])
        self.sampling = config.get("sampling", {})

        self.stats_name = config.get("stats_name", None)

        # XXX host_rewrite actually isn't in the schema right now.
        self.host_rewrite = config.get("host_rewrite", None)

        # Remember that the config references us.
        self.referenced_by(config)

        return True

    def add_mappings(self, ir: "IR", aconf: Config):
        cluster = ir.add_cluster(
            IRCluster(
                ir=ir,
                aconf=aconf,
                parent_ir_resource=self,
                location=self.location,
                service=self.service,
                host_rewrite=self.get("host_rewrite", None),
                marker="tracing",
                grpc=self.grpc,
                stats_name=self.get("stats_name", None),
            )
        )

        cluster.referenced_by(self)
        self.cluster = cluster

    def finalize(self):
        assert self.cluster
        self.ir.logger.debug("tracing cluster envoy name: %s" % self.cluster.envoy_name)
        # Opentelemetry is the only one that does not use collector_cluster
        if self.driver == "opentelemetry":
            self.driver_config["grpc_service"] = {
                "envoy_grpc": {"cluster_name": self.cluster.envoy_name}
            }
        else:
            self.driver_config["collector_cluster"] = self.cluster.envoy_name
