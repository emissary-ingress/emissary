import copy
import logging
import sys
import os
import httpretty
import json

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler


def _get_envoy_config(yaml, version='V3'):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return EnvoyConfig.generate(ir, version)


def del_env(env_key):
    if env_key in os.environ:
        del os.environ[env_key]


def teardown_function(function):
    del_env('STATSD_ENABLED')
    del_env('DOGSTATSD')
    del_env('STATSD_HOST')


@pytest.mark.compilertest
@httpretty.activate
def test_statsd_default():
    httpretty.register_uri(
        httpretty.GET,
        "statsd-sink",
        body='{"origin": "127.0.0.1"}'
    )
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  thing-rest
case_sensitive: false
hostname: "*"
prefix: /reset/
rewrite: /RESET/
service: beepboop

"""
    expected_stats_sinks = {
        "name":"envoy.stats_sinks.statsd",
        "typed_config":{
           "@type":"type.googleapis.com/envoy.config.metrics.v3.StatsdSink",
           "address":{
              "socket_address":{
                 "protocol":"UDP",
                 "address":"127.0.0.1",
                 "port_value":8125
              }
           }
        }
     }

    os.environ['STATSD_ENABLED'] = 'true'
    econf = _get_envoy_config(yaml, version='V3')

    assert econf

    econf_dict = econf.as_dict()

    assert 'stats_sinks' in econf_dict['bootstrap']
    assert len(econf_dict['bootstrap']['stats_sinks']) > 0
    assert econf_dict['bootstrap']['stats_sinks'][0] == expected_stats_sinks
    assert 'stats_flush_interval' in econf_dict['bootstrap']
    assert econf_dict['bootstrap']['stats_flush_interval']['seconds'] == '1'


@pytest.mark.compilertest
@httpretty.activate
def test_statsd_default():
    httpretty.register_uri(
        httpretty.GET,
        "other-statsd-sink",
        body='{"origin": "127.0.0.1"}'
    )
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  thing-rest
case_sensitive: false
hostname: "*"
prefix: /reset/
rewrite: /RESET/
service: beepboop

"""
    expected_stats_sinks = {
        "name":"envoy.stats_sinks.statsd",
        "typed_config":{
           "@type":"type.googleapis.com/envoy.config.metrics.v3.StatsdSink",
           "address":{
              "socket_address":{
                 "protocol":"UDP",
                 "address":"127.0.0.1",
                 "port_value":8125
              }
           }
        }
     }

    os.environ['STATSD_ENABLED'] = 'true'
    os.environ['STATSD_HOST'] = 'other-statsd-sink'
    econf = _get_envoy_config(yaml, version='V3')

    assert econf

    econf_dict = econf.as_dict()

    assert 'stats_sinks' in econf_dict['bootstrap']
    assert len(econf_dict['bootstrap']['stats_sinks']) > 0
    assert econf_dict['bootstrap']['stats_sinks'][0] == expected_stats_sinks
    assert 'stats_flush_interval' in econf_dict['bootstrap']
    assert econf_dict['bootstrap']['stats_flush_interval']['seconds'] == '1'


@pytest.mark.compilertest
@httpretty.activate
def test_dogstatsd():
    httpretty.register_uri(
        httpretty.GET,
        "statsd-sink",
        body='{"origin": "127.0.0.1"}'
    )
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  thing-rest
case_sensitive: false
hostname: "*"
prefix: /reset/
rewrite: /RESET/
service: beepboop

"""
    expected_stats_sinks = {
        "name":"envoy.stat_sinks.dog_statsd",
        "typed_config":{
           "@type":"type.googleapis.com/envoy.config.metrics.v3.DogStatsdSink",
           "address":{
              "socket_address":{
                 "protocol":"UDP",
                 "address":"127.0.0.1",
                 "port_value":8125
              }
           }
        }
     }


    os.environ['STATSD_ENABLED'] = 'true'
    os.environ['DOGSTATSD'] = 'true'
    econf = _get_envoy_config(yaml, version='V3')

    assert econf

    econf_dict = econf.as_dict()

    assert 'stats_sinks' in econf_dict['bootstrap']
    assert len(econf_dict['bootstrap']['stats_sinks']) > 0
    assert econf_dict['bootstrap']['stats_sinks'][0] == expected_stats_sinks
    assert 'stats_flush_interval' in econf_dict['bootstrap']
    assert econf_dict['bootstrap']['stats_flush_interval']['seconds'] == '1'
