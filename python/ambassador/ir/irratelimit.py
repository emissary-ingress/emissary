from typing import Optional, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irfilter import IRFilter
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRRateLimit (IRFilter):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.ratelimit",
                 kind: str="IRRateLimit",
                 name: str="rate_limit",    # This is a key for Envoy! You can't just change it.
                 namespace: Optional[str] = None,
                 **kwargs) -> None:
        # print("IRRateLimit __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, namespace=namespace, type='decoder'
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        config_info = aconf.get_config("ratelimit_configs")

        if not config_info:
            ir.logger.debug("IRRateLimit: no ratelimit config, bailing")
            # No ratelimit info. Be done.
            return False

        configs = config_info.values()
        number_configs = len(configs)
        if number_configs != 1:
            self.post_error("only one RateLimitService is supported, got {}".format(number_configs))
            return False

        config = list(configs)[0]

        service = config.get("service", None)

        if not service:
            self.post_error(RichStatus.fromError("service is required in RateLimitService",
                                                 module=config))
            return False

        ir.logger.debug("IRRateLimit: ratelimit using service %s" % service)

        # OK, we have a valid config.

        self.service = service
        self.ctx_name = config.get('tls', None)
        self.name = "rate_limit"    # Force this, just in case.
        self.namespace = config.get("namespace", self.namespace)
        self.domain = config.get('domain', ir.ambassador_module.default_label_domain)
        self.protocol_version = config.get("protocol_version", "v2")

        self.stats_name = config.get("stats_name", None)

        # XXX host_rewrite actually isn't in the schema right now.
        self.host_rewrite = config.get('host_rewrite', None)

        # Should we use the shiny new data_plane_proto? Default false right now.
        # XXX Needs to be configurable.
        self.data_plane_proto = False

        # Filter config.
        self.config = {
            "domain": self.domain,
            "timeout_ms": config.get('timeout_ms', 20),
            "request_type": "both"  # XXX configurability!
        }

        self.sourced_by(config)
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
                grpc=True,
                host_rewrite=self.get('host_rewrite', None),
                ctx_name=self.get('ctx_name', None),
                stats_name=self.get("stats_name", None)
            )
        )

        cluster.referenced_by(self)

        # Go ahead and define a GRPC service -- just recognize that we may not
        # use it.
        self.grpc_service = ir.add_grpc_service("rate_limit_service", cluster)

        self.cluster = cluster
