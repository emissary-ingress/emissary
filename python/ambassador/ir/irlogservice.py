from typing import Optional, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR


class IRLogService(IRResource):
    cluster: Optional[IRCluster]
    service: str
    driver: str
    driver_config: dict
    flush_interval_byte_size: int
    flush_interval_time: int
    grpc: bool

    def __init__(self, ir: 'IR', config,
                 rkey: str = "ir.logservice",
                 kind: str = "ir.logservice",
                 name: str = "logservice",
                 namespace: Optional[str] = None,
                 **kwargs) -> None:
        del kwargs  # silence unused-variable warning

        super().__init__(
            ir=ir, aconf=config, rkey=rkey, kind=kind, name=name, namespace=namespace
        )

    def setup(self, ir: 'IR', config) -> bool:
        self.service = config.get('service')
        if not self.service:
            self.post_error("service must be present for a remote log service!")
            return False

        self.namespace = config.get("namespace", self.namespace)
        self.cluster = None
        self.grpc = config.get('grpc', False)
        self.driver = config.get('driver')
        # These defaults come from Envoy:
        # https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/accesslog/v2/als.proto#envoy-api-msg-config-accesslog-v2-commongrpcaccesslogconfig
        self.flush_interval_byte_size = config.get('flush_interval_byte_size', 16384)
        self.flush_interval_time = config.get('flush_interval_time', 1)

        self.driver_config = config.get('driver_config')
        if 'additional_log_headers' in self.driver_config:
            if self.driver != 'http' and self.driver_config['additional_log_headers']:
                self.post_error("additional_log_headers are not supported in tcp mode")
                return False

            for header_obj in self.get_additional_headers():
                if header_obj.get('header_name', '') == '':
                    self.post_error("Please provide a header name for every additional log header!")
                    return False

        self.sourced_by(config)
        self.referenced_by(config)

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        self.cluster = ir.add_cluster(
            IRCluster(
                ir=ir,
                aconf=aconf,
                parent_ir_resource=self,
                location=self.location,
                service=self.service,
                host_rewrite=self.get('host_rewrite', None),
                marker='logging',
                grpc=self.grpc
            )
        )

        self.cluster.referenced_by(self)

    def get_common_config(self) -> dict:
        # get_common_config isn't allowed to be called before add_mappings 
        # is called (by ir.walk_saved_resources). So we can assert that 
        # self.cluster isn't None here, both to make mypy happier and out
        # of paranoia.
        assert(self.cluster)

        return {
            "log_name": self.name,
            "grpc_service": {
                "envoy_grpc": {
                    "cluster_name": self.cluster.name
                }
            },
            "buffer_flush_interval": "%ds" % self.flush_interval_time,
            "buffer_size_bytes": self.flush_interval_byte_size,
        }

    def get_additional_headers(self) -> list:
        if 'additional_log_headers' in self.driver_config:
            return self.driver_config.get('additional_log_headers', [])
        else:
            return []


class IRLogServiceFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        services = aconf.get_config('log_services')
        if services is not None:
            for config in services.values():
                srv = IRLogService(ir, config)
                extant_srv = ir.log_services.get(srv.name, None)

                if extant_srv:
                    ir.post_error("Duplicate LogService %s; keeping definition from %s" % (srv.name, extant_srv.location))
                else:
                    ir.log_services[srv.name] = srv
                    ir.save_resource(srv)
