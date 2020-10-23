from typing import Optional, TYPE_CHECKING
from typing import cast as typecast

from ..config import Config
from ..utils import RichStatus
from ..resource import Resource

from .irfilter import IRFilter

if TYPE_CHECKING:
    from .ir import IR

# IRErrorResponse implements custom error response bodies using Envoy's HTTP response_map filter.
#
# Error responses are configured as an array of rules on the Ambassador module. Rules can be
# bypassed on a Mapping using `bypass_error_response_overrides`. In a future implementation,
# rules will be supported at both the Module level and at the Mapping level, allowing a flexible
# configuration where certain default behaviors apply and Mappings can specify have their own.
#
# The Ambassador module config isn't subject to strict typing at higher layers, so this IR has
# to pay special attention to the types and format of the incoming config.
class IRErrorResponse (IRFilter):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.error_response",
                 kind: str="IRErrorResponse",
                 name: str="error_response",
                 type: Optional[str] = "decoder",
                 **kwargs) -> None:
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

    # Configure global behavior using the `error_response_overrides` module config.
    def setup(self, ir: 'IR', aconf: Config) -> bool:
        error_response_config = ir.ambassador_module.get('error_response_overrides', [])

        # Do nothing (and post no errors) if there's no error_response_overrides configured.
        if not error_response_config:
            return False

        # If we have some configuration to deal with, try to load it, and post an errors
        # we find along the way. Internally, _load_error_response_config will skip any
        # error response rules that are invalid, preserving other rules. This prevents
        # one bad rule from eliminating the others. In practice this isn't as useful as
        # it sounds because module config is only loaded once on startup, but ideally
        # we'll move away from that limitation.
        self.config = self._load_error_response_config(error_response_config)
        if not self.config:
            return False

        ir.logger.debug("IRErrorResponse: loaded config %s" % repr(self.config))
        self.referenced_by(ir.ambassador_module)
        return True

    def _load_error_response_config(self, error_response_config):
        # The error_response_overrides field must be an array
        if not isinstance(error_response_config, list):
            self.post_error(f"IRErrorResponse: error_response_overrides: field must be an array")
            return False

        # The error_response_overrides field must contain at least one entry
        if len(error_response_config) == 0:
            self.post_error(f"IRErrorResponse: error_response_overrides: no mappers, nothing to do")
            return False

        all_mappers = []
        for error_response in error_response_config:
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

                ir_on_status_code = int(ir_on_status_code)
                if ir_on_status_code < 400 or ir_on_status_code >= 600:
                    raise ValueError("field must be an integer >= 400 and < 600")

                ir_on_status_code = str(ir_on_status_code)
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
                        f"IRErrorResponse: body: field must be an object, found %s" % ir_body)
                continue

            # We currently only support filtering using an equality match on status codes.
            # The underlying response_map filter in Envoy supports a larger set of filters,
            # however, and adding support for them should be relatively straight-forward.
            mapper = {
                "filter": {
                    "status_code_filter": {
                        "comparison": {
                            "op": "EQ",
                            "value": {
                                "default_value": ir_on_status_code,
                                # Envoy _requires_ that the status code comparison value
                                # has an associated "runtime_key". This is used as a key
                                # in the runtime config system for changing config values
                                # without restarting Envoy.

                                # We definitely do not want this value to ever change
                                # inside of Envoy at runtime, so the best we can do is name
                                # this key something arbitrary and hopefully unused.
                                "runtime_key": "_donotsetthiskey"
                            }
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

            # Only one of text_format, json_format, or text_format_source may be set.
            # Post an error if we found more than one these fields set.
            formats_set = 0
            for f in [ ir_text_format_source, ir_text_format, ir_json_format ]:
                if f is not None:
                    formats_set += 1
            if formats_set > 1:
                self.post_error(
                        "IRErrorResponse: only one of \"text_format\", \"json_format\", "
                        +"or \"text_format_source\" may be set, found %d of these fields set." %
                        formats_set)
                continue

            body_format_override = dict()

            if ir_text_format_source is not None:
                # Verify that the text_format_source field is an object with a string filename.
                if not isinstance(ir_text_format_source, dict) or \
                        not isinstance(ir_text_format_source.get('filename', None), str):
                    self.post_error(
                            f"IRErrorResponse: text_format_source field must be an object with a single filename field, found \"{ir_text_format_source}\"")
                    continue

                body_format_override["text_format_source"] = ir_text_format_source
            elif ir_text_format is not None:
                # Verify that the text_format field is a string
                try:
                    body_format_override["text_format"] = str(ir_text_format)
                except ValueError as e:
                    self.post_error(f"IRErrorResponse: text_format: %s" % e)
            elif ir_json_format is not None:
                # Verify that the json_format field is an object
                if not isinstance(ir_json_format, dict):
                    self.post_error(f"IRErrorResponse: json_format field must be an object, found \"{ir_json_format}\"")
                    continue

                body_format_override["json_format"] = ir_json_format
            else:
                self.post_error(
                        f"IRErrorResponse: could not find a valid format field in body \"{ir_body}\"")
                continue

            if ir_content_type is not None:
                # Content type is optional, but it must be a string if set.
                if not isinstance(ir_content_type, str):
                    self.post_error(f"IRErrorResponse: content_type: field must be a string")
                    continue

                body_format_override["content_type"] = ir_content_type

            # The mapper config now has a `filter` (the rule) and a `body_format_override` (the action)
            mapper["body_format_override"] = body_format_override
            all_mappers.append(mapper)

        # If nothing could be parsed successfully, post an error.
        if len(all_mappers) < 1:
            self.post_error(f"IRErrorResponse: no valid error response mappers could be parsed")
            return False

        # We only use the mappers field of the response map config.
        config = {
            'mappers': all_mappers
        }
        return config
