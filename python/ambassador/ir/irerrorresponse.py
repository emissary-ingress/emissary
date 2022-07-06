from typing import TYPE_CHECKING, Any, ClassVar, Dict, List, Optional
from typing import cast as typecast

from ..config import Config
from .irfilter import IRFilter

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover
    from .ir.irresource import IRResource  # pragma: no cover

import re

# github.com/datawire/apro/issues/2661
# Use a whitelist to validate that any command operators in error response body are supported by envoy
# TODO: remove this after support for escaping "%" lands in envoy
ALLOWED_ENVOY_FMT_TOKENS = [
    "START_TIME",
    "REQUEST_HEADERS_BYTES",
    "BYTES_RECEIVED",
    "PROTOCOL",
    "RESPONSE_CODE",
    "RESPONSE_CODE_DETAILS",
    "CONNECTION_TERMINATION_DETAILS",
    "RESPONSE_HEADERS_BYTES",
    "RESPONSE_TRAILERS_BYTES",
    "BYTES_SENT",
    "UPSTREAM_WIRE_BYTES_SENT",
    "UPSTREAM_WIRE_BYTES_RECEIVED",
    "UPSTREAM_HEADER_BYTES_SENT",
    "UPSTREAM_HEADER_BYTES_RECEIVED",
    "DOWNSTREAM_WIRE_BYTES_SENT",
    "DOWNSTREAM_WIRE_BYTES_RECEIVED",
    "DOWNSTREAM_HEADER_BYTES_SENT",
    "DOWNSTREAM_HEADER_BYTES_RECEIVED",
    "DURATION",
    "REQUEST_DURATION",
    "REQUEST_TX_DURATION",
    "RESPONSE_DURATION",
    "RESPONSE_TX_DURATION",
    "RESPONSE_FLAGS",
    "ROUTE_NAME",
    "UPSTREAM_HOST",
    "UPSTREAM_CLUSTER",
    "UPSTREAM_LOCAL_ADDRESS",
    "UPSTREAM_TRANSPORT_FAILURE_REASON",
    "DOWNSTREAM_REMOTE_ADDRESS",
    "DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT",
    "DOWNSTREAM_DIRECT_REMOTE_ADDRESS",
    "DOWNSTREAM_DIRECT_REMOTE_ADDRESS_WITHOUT_PORT",
    "DOWNSTREAM_LOCAL_ADDRESS",
    "DOWNSTREAM_LOCAL_ADDRESS_WITHOUT_PORT",
    "CONNECTION_ID",
    "GRPC_STATUS",
    "DOWNSTREAM_LOCAL_PORT",
    "REQ",
    "RESP",
    "TRAILER",
    "DYNAMIC_METADATA",
    "CLUSTER_METADATA",
    "FILTER_STATE",
    "REQUESTED_SERVER_NAME",
    "DOWNSTREAM_LOCAL_URI_SAN",
    "DOWNSTREAM_PEER_URI_SAN",
    "DOWNSTREAM_LOCAL_SUBJECT",
    "DOWNSTREAM_PEER_SUBJECT",
    "DOWNSTREAM_PEER_ISSUER",
    "DOWNSTREAM_TLS_SESSION_ID",
    "DOWNSTREAM_TLS_CIPHER",
    "DOWNSTREAM_TLS_VERSION",
    "DOWNSTREAM_PEER_FINGERPRINT_256",
    "DOWNSTREAM_PEER_FINGERPRINT_1",
    "DOWNSTREAM_PEER_SERIAL",
    "DOWNSTREAM_PEER_CERT",
    "DOWNSTREAM_PEER_CERT_V_START",
    "DOWNSTREAM_PEER_CERT_V_END",
    "HOSTNAME",
    "LOCAL_REPLY_BODY",
    "FILTER_CHAIN_NAME",
]
ENVOY_FMT_TOKEN_REGEX = (
    "\%([A-Za-z0-9_]+?)(\([A-Za-z0-9_.]+?((:|\?)[A-Za-z0-9_.]+?)+\))?(:[A-Za-z0-9_]+?)?\%"
)

# IRErrorResponse implements custom error response bodies using Envoy's HTTP response_map filter.
#
# Error responses are configured as an array of rules on the Ambassador module. Rules can be
# bypassed on a Mapping using `bypass_error_response_overrides`. In a future implementation,
# rules will be supported at both the Module level and at the Mapping level, allowing a flexible
# configuration where certain behaviors apply globally and Mappings can override them.
#
# The Ambassador module config isn't subject to strict typing at higher layers, so this IR has
# to pay special attention to the types and format of the incoming config.
class IRErrorResponse(IRFilter):

    # The list of mappers that will make up the final error response config
    _mappers: Optional[List[Dict[str, Any]]]

    # The IR config, used as input, typically from an `error_response_overrides` field
    # on a Resource (eg: the Ambassador module or a Mapping)
    _ir_config: List[Dict[str, Any]]

    # The object that references this IRErrorResource.
    # Use by diagnostics to report the exact source of configuration errors.
    _referenced_by_obj: Optional["IRResource"]

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        error_response_config: List[Dict[str, Any]],
        referenced_by_obj: Optional["IRResource"] = None,
        rkey: str = "ir.error_response",
        kind: str = "IRErrorResponse",
        name: str = "error_response",
        type: Optional[str] = "decoder",
        **kwargs,
    ) -> None:
        self._ir_config = error_response_config
        self._referenced_by_obj = referenced_by_obj
        self._mappers = None
        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

    # Return the final config, or None if there isn't any, either because
    # there was no input config, or none of the input config was valid.
    #
    # Callers shoulh always check for None to mean that this IRErrorResponse
    # has no config to generate, and so the underlying envoy.http.filter.response_map
    # (or per-route config) does not need to be configured.
    def config(self) -> Optional[Dict[str, Any]]:
        if not self._mappers:
            return None
        return {"mappers": self._mappers}

    # Runs setup and always returns true to indicate success. This is safe because
    # _setup is tolerant of missing or invalid config. At the end of setup, the caller
    # should retain this object and use `config()` get the final, good config, if any.
    def setup(self, ir: "IR", aconf: Config) -> bool:
        self._setup(ir, aconf)
        return True

    def _setup(self, ir: "IR", aconf: Config):
        # Do nothing (and post no errors) if there's no config.
        if not self._ir_config:
            return

        # The error_response_overrides config must be an array
        if not isinstance(self._ir_config, list):
            self.post_error(
                f"IRErrorResponse: error_response_overrides: field must be an array, got {type(self._ir_config)}"
            )
            return

        # Do nothing (and post no errors) if there's config, but it's empty.
        if len(self._ir_config) == 0:
            return

        # If we have some configuration to deal with, try to load it, and post any errors
        # that we find along the way. Internally, _load_config will skip any error response rules
        # that are invalid, preserving other rules. This prevents one bad rule from eliminating
        # the others. In practice this isn't as useful as it sounds because module config is only
        # loaded once on startup, but ideally we'll move away from that limitation.
        self._mappers = self._generate_mappers()
        if self._mappers is not None:
            ir.logger.debug("IRErrorResponse: loaded mappers %s" % repr(self._mappers))
            if self._referenced_by_obj is not None:
                self.referenced_by(self._referenced_by_obj)

    def _generate_mappers(self) -> Optional[List[Dict[str, Any]]]:
        all_mappers: List[Dict[str, Any]] = []
        for error_response in self._ir_config:
            # Try to parse `on_status_code` (a required field) as an integer
            # in the interval [400, 600). We don't support matching on 3XX
            # (or 1xx/2xx for that matter) codes yet. If there's appetite for
            # that in the future, it should be as easy as relaxing the rules
            # enforced here. The underlying response_map filter in Envoy supports
            # it natively.
            try:
                ir_on_status_code = error_response.get("on_status_code", None)
                if ir_on_status_code is None:
                    raise ValueError("field must exist")

                code = int(ir_on_status_code)
                if code < 400 or code >= 600:
                    raise ValueError("field must be an integer >= 400 and < 600")

                status_code_str: str = str(code)
            except ValueError as e:
                self.post_error(f"IRErrorResponse: on_status_code: %s" % e)
                continue

            # Try to parse `body` (a required field) as an object.
            ir_body = error_response.get("body", None)
            if ir_body is None:
                self.post_error(f"IRErrorResponse: body: field must exist")
                continue
            if not isinstance(ir_body, dict):
                self.post_error(
                    f"IRErrorResponse: body: field must be an object, found %s" % ir_body
                )
                continue

            # We currently only support filtering using an equality match on status codes.
            # The underlying response_map filter in Envoy supports a larger set of filters,
            # however, and adding support for them should be relatively straight-forward.
            mapper: Dict[str, Any] = {
                "filter": {
                    "status_code_filter": {
                        "comparison": {
                            "op": "EQ",
                            "value": {
                                "default_value": status_code_str,
                                # Envoy _requires_ that the status code comparison value
                                # has an associated "runtime_key". This is used as a key
                                # in the runtime config system for changing config values
                                # without restarting Envoy.
                                # We definitely do not want this value to ever change
                                # inside of Envoy at runtime, so the best we can do is name
                                # this key something arbitrary and hopefully unused.
                                "runtime_key": "_donotsetthiskey",
                            },
                        }
                    }
                }
            }

            # Content type is optional. It can be used to override the content type of the
            # error response body.
            ir_content_type = ir_body.get("content_type", None)

            ir_text_format_source = ir_body.get("text_format_source", None)
            ir_text_format = ir_body.get("text_format", None)
            ir_json_format = ir_body.get("json_format", None)

            # get the text used for error response body so we can check it for bad tokens
            # TODO: remove once envoy supports escaping "%"
            format_body = ""

            # Only one of text_format, json_format, or text_format_source may be set.
            # Post an error if we found more than one these fields set.
            formats_set: int = 0
            for f in [ir_text_format_source, ir_text_format, ir_json_format]:
                if f is not None:
                    formats_set += 1
            if formats_set > 1:
                self.post_error(
                    'IRErrorResponse: only one of "text_format", "json_format", '
                    + 'or "text_format_source" may be set, found %d of these fields set.'
                    % formats_set
                )
                continue

            body_format_override: Dict[str, Any] = {}

            if ir_text_format_source is not None:
                # Verify that the text_format_source field is an object with a string filename.
                if not isinstance(ir_text_format_source, dict) or not isinstance(
                    ir_text_format_source.get("filename", None), str
                ):
                    self.post_error(
                        f'IRErrorResponse: text_format_source field must be an object with a single filename field, found "{ir_text_format_source}"'
                    )
                    continue

                body_format_override["text_format_source"] = ir_text_format_source
                try:
                    fmt_file = open(ir_text_format_source["filename"], mode="r")
                    format_body = fmt_file.read()
                    fmt_file.close()
                except OSError:
                    self.post_error(
                        "IRErrorResponse: text_format_source field references a file that does not exist"
                    )
                    continue

            elif ir_text_format is not None:
                # Verify that the text_format field is a string
                try:
                    body_format_override["text_format"] = str(ir_text_format)
                    format_body = str(ir_text_format)
                except ValueError as e:
                    self.post_error(f"IRErrorResponse: text_format: %s" % e)
            elif ir_json_format is not None:
                # Verify that the json_format field is an object
                if not isinstance(ir_json_format, dict):
                    self.post_error(
                        f'IRErrorResponse: json_format field must be an object, found "{ir_json_format}"'
                    )
                    continue

                # Envoy requires string values for json_format. Validate that every field in the
                # json_format can be trivially converted to a string, error otherwise.
                #
                # The mapping CRD validates that json_format maps strings to strings, but our
                # module config doesn't have the same validation, so we do it here.
                error: str = ""
                sanitized: Dict[str, str] = {}
                try:
                    for k, v in ir_json_format.items():
                        k = str(k)
                        if isinstance(v, bool):
                            sanitized[k] = str(v).lower()
                            format_body += f"{k}: {str(v).upper()}, "
                        elif isinstance(v, (int, float, str)):
                            sanitized[k] = str(v)
                            format_body += f"{k}: {str(v)}, "
                        else:
                            error = f'IRErrorResponse: json_format only supports string values, and type "{type(v)}" for key "{k}" cannot be implicitly converted to string'
                            break
                except ValueError as e:
                    # This really shouldn't be possible, because the string casts we do above
                    # are "safely" done on types where casting is always valid (eg: bool, int).
                    error = f"IRErrorResponse: unexpected ValueError while sanitizing ir_json_format {ir_json_format}: {e}"

                if error:
                    self.post_error(error)
                    continue

                body_format_override["json_format"] = sanitized
            else:
                self.post_error(
                    f'IRErrorResponse: could not find a valid format field in body "{ir_body}"'
                )
                continue

            if ir_content_type is not None:
                # Content type is optional, but it must be a string if set.
                if not isinstance(ir_content_type, str):
                    self.post_error(f"IRErrorResponse: content_type: field must be a string")
                    continue

                body_format_override["content_type"] = ir_content_type

            # search the body for command tokens
            # TODO: remove this code when envoy supports escaping "%"
            token_finder = re.compile(ENVOY_FMT_TOKEN_REGEX)
            matches = token_finder.findall(format_body)

            bad_token = False
            for i in matches:
                # i[0] is first group in regex match which will contain the command operator name
                if not i[0] in ALLOWED_ENVOY_FMT_TOKENS:
                    self.post_error(f"IRErrorResponse: Invalid Envoy command token: {i[0]}")
                    bad_token = True

            if bad_token:
                continue

            # The mapper config now has a `filter` (the rule) and a `body_format_override` (the action)
            mapper["body_format_override"] = body_format_override
            all_mappers.append(mapper)

        # If nothing could be parsed successfully, post an error.
        if len(all_mappers) == 0:
            self.post_error(f"IRErrorResponse: no valid error response mappers could be parsed")
            return None

        return all_mappers
