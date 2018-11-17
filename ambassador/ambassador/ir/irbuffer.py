from typing import Optional, TYPE_CHECKING
from typing import cast as typecast

from ..config import Config
from ..utils import RichStatus
from ..resource import Resource

from .irfilter import IRFilter
from .ircluster import IRCluster

if TYPE_CHECKING:
    from .ir import IR

class IRBuffer (IRFilter):
    
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.buffer",
                 kind: str="IRBuffer",
                 **kwargs) -> None: 

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, **kwargs)

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        config_info = aconf.get_config("buffer_configs")

        if config_info:
            for config in config_info.values():
                self._load_buffer(config)
        else:
            return False

        return True

    def _load_buffer(self, module: Resource):
        if self.location == '--internal--':
            self.sourced_by(module)

        for key in [ 'max_request_bytes', 'max_request_time' ]:
            value = module.get(key, None)

            if value:
                previous = self.get(key, None)

                if previous and (previous != value):
                    errstr = (
                        "Buffer filter cannot support multiple %s values; using %s" %
                        (key, previous)
                    )

                    self.post_error(RichStatus.fromError(errstr, resource=module))
                else:
                    self[key] = value

            self.referenced_by(module)

        max_req_bytes = module.get("max_request_bytes", None)
        if max_req_bytes is not None:
            self["max_request_bytes"] = max_req_bytes
        else:
            self.post_error(RichStatus.fromError("missing required field: max_request_bytes", resource=module))
        
        max_req_time = module.get("max_request_time", None)
        if max_req_time is not None:
            self["max_request_time"] = max_req_time
        else:
            self.post_error(RichStatus.fromError("missing required field: max_request_time", resource=module))
