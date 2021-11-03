import logging
import sys

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR
from ambassador.ir.irerrorresponse import IRErrorResponse
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler


def _status_code_filter_eq_obj(status_code):
    return {
        'status_code_filter': {
            'comparison': {
                'op': 'EQ',
                'value': {
                    'default_value': f'{status_code}',
                    'runtime_key': '_donotsetthiskey'
                }
            }
        }
    }


def _json_format_obj(json_format, content_type=None):
    return {
        'json_format': json_format
    }


def _text_format_obj(text, content_type=None):
    obj = {
            'text_format': f'{text}'
    }
    if content_type is not None:
        obj['content_type'] = content_type
    return obj


def _text_format_source_obj(filename, content_type=None):
    obj = {
            'text_format_source': {
                'filename': filename
            }
    }
    if content_type is not None:
        obj['content_type'] = content_type
    return obj


def _ambassador_module_config():
    return '''
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
'''


def _ambassador_module_onemapper(status_code, body_kind, body_value, content_type=None):
    mod = _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: "{status_code}"
    body:
'''
    if body_kind == 'text_format_source':
        mod = mod + f'''
      {body_kind}:
        filename: "{body_value}"
'''
    elif body_kind == 'json_format':
        mod = mod + f'''
      {body_kind}: {body_value}
'''
    else:
        mod = mod + f'''
      {body_kind}: "{body_value}"
'''
    if content_type is not None:
        mod = mod + f'''
      content_type: "{content_type}"
'''
    return mod


def _test_errorresponse(yaml, expectations, expect_fail=False):
    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    error_response = IRErrorResponse(ir, aconf,
                                     ir.ambassador_module.get('error_response_overrides', None),
                                     ir.ambassador_module)

    error_response.setup(ir, aconf)
    if aconf.errors:
        print("errors: %s" % repr(aconf.errors))

    ir_conf = error_response.config()
    if expect_fail:
        assert ir_conf is None
        return
    assert ir_conf

    # There should be no default body format override
    body_format = ir_conf.get('body_format', None)
    assert body_format is None

    mappers = ir_conf.get('mappers', None)
    assert mappersx
    assert len(mappers) == len(expectations), \
            f"unexpected len(mappers) {len(expectations)} != len(expectations) {len(expectations)}"

    for i in range(len(expectations)):
        expected_filter, expected_body_format_override = expectations[i]
        m = mappers[i]

        print("checking with expected_body_format_override %s and expected_filter %s" %
                (expected_body_format_override, expected_filter))
        print("checking m: ", m)
        actual_filter = m['filter']
        assert m['filter'] == expected_filter
        if expected_body_format_override:
            actual_body_format_override = m['body_format_override']
            assert actual_body_format_override == expected_body_format_override


def _test_errorresponse_onemapper(yaml, expected_filter, expected_body_format_override, fail=False):
    return _test_errorresponse(yaml, [ (expected_filter, expected_body_format_override) ], expect_fail=fail)


def _test_errorresponse_twomappers(yaml, expectation1, expectation2, fail=False):
    return _test_errorresponse(yaml, [ expectation1, expectation2 ], expect_fail=fail)


def _test_errorresponse_onemapper_onstatuscode_textformat(status_code, text_format):
    _test_errorresponse_onemapper(
        _ambassador_module_onemapper(status_code, 'text_format', text_format),
        _status_code_filter_eq_obj(status_code),
        _text_format_obj(text_format)
    )


def _test_errorresponse_onemapper_onstatuscode_textformat_contenttype(
        status_code, text_format, content_type):
    _test_errorresponse_onemapper(
        _ambassador_module_onemapper(
            status_code, 'text_format', text_format, content_type=content_type),
        _status_code_filter_eq_obj(status_code),
        _text_format_obj(text_format, content_type=content_type)
    )


def _test_errorresponse_onemapper_onstatuscode_textformat_datasource(
        status_code, text_format, source, content_type):
    open(source, "x").close()
    _test_errorresponse_onemapper(
        _ambassador_module_onemapper(status_code, 'text_format_source', source,
            content_type=content_type),
        _status_code_filter_eq_obj(status_code),
        _text_format_source_obj(source, content_type=content_type),
    )


def _sanitize_json(json_format):
    sanitized = {}
    for k, v in json_format.items():
        if isinstance(v, bool):
            sanitized[k] = str(v).lower()
        else:
            sanitized[k] = str(v)
    return sanitized


def _test_errorresponse_onemapper_onstatuscode_jsonformat(status_code, json_format):
    _test_errorresponse_onemapper(
        _ambassador_module_onemapper(status_code, 'json_format', json_format),
        _status_code_filter_eq_obj(status_code),
        # We expect the output json to be sanitized and contain the string representation
        # of every value. We provide a basic implementation of string sanitizatino in this
        # test, `sanitize_json`.
        _json_format_obj(_sanitize_json(json_format)),
    )


def _test_errorresponse_twomappers_onstatuscode_textformat(code1, text1, code2, text2, fail=False):
    _test_errorresponse_twomappers(
f'''
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
  error_response_overrides:
  - on_status_code: "{code1}"
    body:
      text_format: {text1}
  - on_status_code: {code2}
    body:
      text_format: {text2}
''',
        (_status_code_filter_eq_obj(code1), _text_format_obj(text1)),
        (_status_code_filter_eq_obj(code2), _text_format_obj(text2)),
        fail=fail
    )


def _test_errorresponse_invalid_configs(yaml):
    _test_errorresponse(yaml, list(), expect_fail=True)


@pytest.mark.compilertest
def test_errorresponse_twomappers_onstatuscode_textformat():
    _test_errorresponse_twomappers_onstatuscode_textformat(
        '400', 'bad request my friend', '504', 'waited too long for an upstream resonse'
    )
    _test_errorresponse_twomappers_onstatuscode_textformat(
        '503', 'boom', '403', 'go away'
    )


@pytest.mark.compilertest
def test_errorresponse_onemapper_onstatuscode_textformat():
    _test_errorresponse_onemapper_onstatuscode_textformat(429, '429 the int')
    _test_errorresponse_onemapper_onstatuscode_textformat('501', 'five oh one')
    _test_errorresponse_onemapper_onstatuscode_textformat('400', 'bad req')
    _test_errorresponse_onemapper_onstatuscode_textformat(
        '429', 'too fast, too furious on host %REQ(:authority)%'
    )


@pytest.mark.compilertest
def test_errorresponse_invalid_envoy_operator():
    _test_errorresponse_onemapper_onstatuscode_textformat(404, '%FAILME%', fail=True)


@pytest.mark.compilertest
def test_errorresponse_onemapper_onstatuscode_textformat_contenttype():
    _test_errorresponse_onemapper_onstatuscode_textformat_contenttype('503', 'oops', 'text/what')
    _test_errorresponse_onemapper_onstatuscode_textformat_contenttype(
        '429', '<html>too fast, too furious on host %REQ(:authority)%</html>', 'text/html'
    )
    _test_errorresponse_onemapper_onstatuscode_textformat_contenttype(
        '404', "{\'error\':\'notfound\'}", 'application/json'
    )


@pytest.mark.compilertest
def test_errorresponse_onemapper_onstatuscode_jsonformat():
    _test_errorresponse_onemapper_onstatuscode_jsonformat('501',
        {
            'response_code': '%RESPONSE_CODE%',
            'upstream_cluster': '%UPSTREAM_CLUSTER%',
            'badness': 'yup'
        }
    )
    # Test both a JSON object whose Python type has non-string primitives...
    _test_errorresponse_onemapper_onstatuscode_jsonformat('401',
        {
            'unauthorized': 'yeah',
            'your_address': '%DOWNSTREAM_REMOTE_ADDRESS%',
            'security_level': 9000,
            'awesome': True,
            'floaty': 0.75
        }
    )
    # ...and a JSON object where the Python type already has strings
    _test_errorresponse_onemapper_onstatuscode_jsonformat('403',
        {
            'whoareyou': 'dunno',
            'your_address': '%DOWNSTREAM_REMOTE_ADDRESS%',
            'security_level': '11000',
            'awesome': 'false',
            'floaty': '0.95'
        }
    )


@pytest.mark.compilertest
def test_errorresponse_onemapper_onstatuscode_textformatsource():
    _test_errorresponse_onemapper_onstatuscode_textformat_datasource(
            '400', 'badness', '/tmp/badness', 'text/plain')
    _test_errorresponse_onemapper_onstatuscode_textformat_datasource(
            '404', 'badness', '/tmp/notfound.dat', 'application/specialsauce')
    _test_errorresponse_onemapper_onstatuscode_textformat_datasource(
            '429', '2fast', '/tmp/2fast.html', 'text/html' )
    _test_errorresponse_onemapper_onstatuscode_textformat_datasource(
            '503', 'something went wrong', '/tmp/replies/503.html', 'text/html; charset=UTF-8' )


@pytest.mark.compilertest
def test_errorresponse_invalid_configs():
    # status code must be an int
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: bad
    body:
      text_format: 'good'
''')
    # cannot match on code < 400 nor >= 600
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 200
    body:
      text_format: 'good'
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 399
    body:
      text_format: 'good'
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 600
    body:
      text_format: 'good'
''')
    # body must be a dict
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body: 'bad'
''')
    # body requires a valid format field
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      bad: 'good'
''')
    # body field must be present
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 501
    bad:
      text_format: 'good'
''')
    # body field cannot be an empty dict
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 501
    body: {{}}
''')
    # response override must be a non-empty array
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides: []
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides: 'great sadness'
''')
    # (not an array, bad)
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
    on_status_code: 401
    body:
      text_format: 'good'
''')
    # text_format_source must have a single string 'filename'
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format_source: "this obviously cannot be a string"
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format_source:
        filename: []
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format_source:
        notfilename: "/tmp/good"
''')
    # json_format field must be an object field
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      json_format: "this also cannot be a string"
''')
    # json_format cannot have values that do not cast to string trivially
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      json_format:
        "x":
          "yo": 1
        "field": "good"
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      json_format:
        "a": []
        "x": true
''')
    # content type, if it exists, must be a string
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: "good"
      content_type: []
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: "good"
      content_type: 4.2
''')
    # only one of text_format, json_format, or text_format_source may be set
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: "bad"
      json_format:
        "bad": 1
        "invalid": "bad"
''')
    _test_errorresponse_invalid_configs(
        _ambassador_module_config() + f'''
  error_response_overrides:
  - on_status_code: 401
    body:
      text_format: "goodgood"
      text_format_source:
        filename: "/etc/issue"
''')

if __name__ == '__main__':
    pytest.main(sys.argv)
