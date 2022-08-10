import logging
from typing import TYPE_CHECKING, Any, Dict, List, Optional, Union

from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRHealthChecks(IRResource):

    # The list of mappers that will make up the final health checking config
    _mappers: Optional[List[Dict[str, Union[str, int, Dict]]]]

    # The IR config, used as input from a `health_checks` field on a Mapping
    _ir_config: List[Dict[str, Union[str, int, Dict]]]

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        health_checks_config: List[Dict[str, Union[str, int, Dict]]],
        rkey: str = "ir.health_checks",
        kind: str = "IRHealthChecks",
        name: str = "health_checks",
        **kwargs,
    ) -> None:
        self._ir_config = health_checks_config
        self._mappers = None
        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

    # Return the final config, or None if there isn't any, either because
    # there was no input config, or none of the input config was valid.
    def config(self) -> Optional[Dict[str, Union[str, int, Dict]]]:
        if not self._mappers:
            return None
        return {"mappers": self._mappers}

    def setup(self, ir: "IR", aconf: Config) -> bool:
        self._setup(ir, aconf)
        return True

    def _setup(self, ir: "IR", aconf: Config):
        # Dont post any errors if there is empty config
        if not self._ir_config:
            return

        # The health checking config must be an array
        if not isinstance(self._ir_config, list):
            self.post_error(
                f"IRHealthChecks: health_checks: field must be an array, got {type(self._ir_config)}, Ignoring...",
                log_level=logging.ERROR,
            )
            return

        # Do nothing (and post no errors) if there's config, but it's empty.
        if len(self._ir_config) == 0:
            return

        # If we have some configuration to deal with, try to load it, and post any errors
        # that we find along the way. Internally, _load_config will skip any health checks
        # that are invalid, preserving other rules. This prevents one bad health check from eliminating
        # the others.
        self._mappers = self._generate_mappers()
        if self._mappers is not None:
            ir.logger.debug("IRHealthChecks: loaded mappers %s" % repr(self._mappers))

    def _generate_mappers(self) -> Optional[List[Dict[str, Union[str, int, Dict]]]]:
        all_mappers: List[Dict[str, Union[str, int, Dict]]] = []

        # Make sure each health check in the list has config for either a grpc health check or an http health check
        for hc in self._ir_config:

            grpc_health_check = hc.get("grpc_health_check", None)
            http_health_check = hc.get("http_health_check", None)

            if grpc_health_check is None and http_health_check is None:
                self.post_error(
                    f"IRHealthChecks: Either grpc_health_check or http_health_check must exist in the health check config. Ignoring health-check: {hc}",
                    log_level=logging.ERROR,
                )
                continue
            if grpc_health_check is not None and http_health_check is not None:
                self.post_error(
                    f"IRHealthChecks: Only one of grpc_health_check or http_health_check may exist in the health check config. Ignoring health-check: {hc}",
                    log_level=logging.ERROR,
                )
                continue

            timeout = hc.get("timeout", "3s")  # default 3.0s timeout
            interval = hc.get("interval", "5s")  # default 5.0s Interval
            healthy_threshold = hc.get("healthy_threshold", 1)
            unhealthy_threshold = hc.get("unhealthy_threshold", 2)

            mapper: Dict[str, Union[str, int, Dict]] = {
                "timeout": timeout,
                "interval": interval,
                "healthy_threshold": healthy_threshold,
                "unhealthy_threshold": unhealthy_threshold,
            }

            # Process a http health check
            if http_health_check is not None:
                if not isinstance(http_health_check, dict):
                    self.post_error(
                        f"IRHealthChecks: http_health_check: field must be an object, found {http_health_check}. Ignoring health-check {hc}",
                        log_level=logging.ERROR,
                    )
                    continue

                # Make sure we have a path
                path = http_health_check.get("path", None)
                if path is None:
                    self.post_error(
                        f"IRHealthChecks: http_health_check.path is a required field. Ignoring health-check: {hc}",
                        log_level=logging.ERROR,
                    )
                    continue
                http_mapper: Dict[str, Any] = {"path": path}

                # Process header add/remove operations
                request_headers_to_add = http_health_check.get("add_request_headers", None)
                if request_headers_to_add is not None:
                    if isinstance(request_headers_to_add, list):
                        self.post_error(
                            f"IRHealthChecks: add_request_headers must be a dict of header:value pairs. Ignoring field for health-check: {hc}",
                            log_level=logging.ERROR,
                        )
                    addHeaders = self.generate_headers_to_add(request_headers_to_add)
                    if len(addHeaders) > 0:
                        http_mapper["request_headers_to_add"] = addHeaders
                request_headers_to_remove = http_health_check.get("remove_request_headers", None)
                if request_headers_to_remove is not None:
                    if not isinstance(request_headers_to_remove, list):
                        self.post_error(
                            f"IRHealthChecks: remove_request_headers must be a list. Ignoring field for health-check: {hc}",
                            log_level=logging.ERROR,
                        )
                    else:
                        http_mapper["request_headers_to_remove"] = request_headers_to_remove

                host = http_health_check.get("hostname", None)
                if host is not None:
                    http_mapper["host"] = host

                # Process the expected statuses
                expected_statuses = http_health_check.get("expected_statuses", None)
                if expected_statuses is not None:
                    validStatuses = []
                    for statusRange in expected_statuses:
                        startCode = int(statusRange["start"])
                        endCode = int(statusRange["end"])
                        if startCode < 100 or startCode >= 600:
                            self.post_error(
                                f"IRHealthChecks: expected_statuses: {startCode} must be an integer >= 100 and < 600. Ignoring expected status for health-check {hc}",
                                log_level=logging.ERROR,
                            )
                            continue
                        if endCode < 100 or endCode >= 600:
                            self.post_error(
                                f"IRHealthChecks: expected_statuses: {endCode} must be an integer >= 100 and < 600. Ignoring expected status for health-check {hc}",
                                log_level=logging.ERROR,
                            )
                            continue
                        if startCode > endCode:
                            self.post_error(
                                f"IRHealthChecks: expected_statuses: status range start value {startCode} cannot be higher than the end {endCode} for range. Ignoring expected status for health-check {hc}",
                                log_level=logging.ERROR,
                            )
                            continue

                        # We add one to the end code because by default Envoy expects the start of the range to be
                        # inclusive, but the end of the range to be exclusive. Lets just make both inclusive for simplicity.
                        endCode += 1
                        newRange = {"start": startCode, "end": endCode}
                        validStatuses.append(newRange)
                    if len(validStatuses) > 0:
                        http_mapper["expected_statuses"] = validStatuses
                # Add the http health check to the config
                mapper["http_health_check"] = http_mapper

            # Process a gRPC health check
            if grpc_health_check is not None:
                if not isinstance(grpc_health_check, dict):
                    self.post_error(
                        f"IRHealthChecks: grpc_health_check: field must be an object, found {grpc_health_check}, Ignoring...",
                        log_level=logging.ERROR,
                    )
                    continue

                service_name = grpc_health_check.get("service_name", None)
                if service_name is None:
                    self.post_error(
                        f"IRHealthChecks: grpc_health_check: required field service_name field not set, ignoring health-check {hc}",
                        log_level=logging.ERROR,
                    )
                    continue
                else:
                    grpc_mapper: Dict[str, str] = {"service_name": service_name}

                authority = grpc_health_check.get("authority", None)
                if authority is not None:
                    grpc_mapper["authority"] = authority

                # Add the gRPC health check to the config
                mapper["grpc_health_check"] = grpc_mapper
            all_mappers.append(mapper)

        # If nothing could be parsed successfully, post an error.
        if len(all_mappers) == 0:
            self.post_error(
                f"IRHealthChecks: no valid health check could be parsed for config: {self._ir_config}",
                log_level=logging.ERROR,
            )
            return None
        return all_mappers

    @staticmethod
    def generate_headers_to_add(header_dict: dict) -> List[dict]:
        headers = []
        for k, v in header_dict.items():
            append = True
            if isinstance(v, dict):
                if "append" in v:
                    append = bool(v["append"])
                headers.append({"header": {"key": k, "value": v["value"]}, "append": append})
            else:
                headers.append(
                    {
                        "header": {"key": k, "value": v},
                        "append": append,  # Default append True, for backward compatability
                    }
                )
        return headers
