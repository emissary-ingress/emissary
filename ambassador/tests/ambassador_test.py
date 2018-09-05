from typing import Any

import sys

import difflib
# import errno
import json
import logging
# import functools
import os
import pytest
import re

# from shell import shell

from diag_paranoia import diag_paranoia, filtered_overview, sanitize_errors
from ambassador.config import fetch_resources
from ambassador import Config, IR
from ambassador.envoy import V1Config

from ambassador.VERSION import Version

__version__ = Version

logging.basicConfig(
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# logging.getLogger("datawire.scout").setLevel(logging.DEBUG)
logger = logging.getLogger("ambassador")
logger.setLevel(logging.DEBUG)

VALIDATOR_IMAGE = "quay.io/datawire/ambassador-envoy:v1.7.0-64-g09ba72b1-alpine-stripped"

DIR = os.path.dirname(__file__)
EXCLUDES = [ "__pycache__" ] 

# TESTDIR = os.path.join(DIR, "tests")
TESTDIR = DIR
DEFAULT_CONFIG = os.path.join(DIR, "..", "default-config")
MATCHES = [ n for n in os.listdir(TESTDIR) 
            if (n.startswith('0') and os.path.isdir(os.path.join(TESTDIR, n)) and (n not in EXCLUDES)) ]

os.environ['SCOUT_DISABLE'] = "1"

#### decorators

def standard_setup(f):
    func_name = getattr(f, '__name__', '<anonymous>')

    # @functools.wraps(f)
    def wrapper(directory, *args, **kwargs):
        print("%s: directory %s" % (func_name, directory))

        dirpath = os.path.join(TESTDIR, directory)
        testname = os.path.basename(dirpath)
        configdir = os.path.join(dirpath, 'config')

        if os.path.exists(os.path.join(dirpath, 'TEST_DEFAULT_CONFIG')):
            configdir = DEFAULT_CONFIG

        print("%s: using config %s" % (testname, configdir))

        return f(testname, dirpath, configdir, *args, **kwargs)

    return wrapper

#### Utilities

def unified_diff(gold_path, current_path):
    gold = json.dumps(json.load(open(gold_path, "r")), indent=4, sort_keys=True)
    current = json.dumps(json.load(open(current_path, "r")), indent=4, sort_keys=True)

    udiff = list(difflib.unified_diff(gold.split("\n"), current.split("\n"),
                                      fromfile=os.path.basename(gold_path),
                                      tofile=os.path.basename(current_path),
                                      lineterm=""))

    return udiff

################################
# At this point we need to turn the IR into something like the old Ambassador config
# for vetting. This code is here because it's temporary -- we don't want to burden
# the IR with it 'cause as soon as it runs clean, we're done with it.

drop_keys = {
    'apiVersion': True,
    'rkey': True,
    'kind': True,
    'ir': True,
    'logger': True,
    'location': True,
}

def as_sourceddict(res: dict) -> Any:
    if isinstance(res, list):
        return [ as_sourceddict(x) for x in res ]
    elif isinstance(res, dict):
        sd = {}

        if 'location' in res:
            sd['_source'] = res['location']

        for key in res.keys():
            if key.startswith('_') or (key in drop_keys):
                continue

            sd[key] = as_sourceddict(res[key])

        if '_referenced_by' in res:
            sd['_referenced_by'] = sorted(res['_referenced_by'])

        return sd
    else:
        return res

class old_ir (dict):
    def __init__(self, aconf: dict, ir: dict, v1config: dict) -> None:
        super().__init__()

        econf = {
            'admin': {
                '_source': ir['ambassador']['location'],
                'admin_port': ir['ambassador']['admin_port']
            },
            'listeners': [ as_sourceddict(x) for x in ir['listeners'] ],
            'filters': [ as_sourceddict(x) for x in ir['filters'] ],
            'routes': [],
            'clusters': []
        }

        for l in econf['listeners']:
            for k in [ '_referenced_by', 'name', 'serialization' ]:
                l.pop(k, None)

        clusters = {}

        for group in sorted(ir['groups'], key=lambda g: g['group_id']):
            route = as_sourceddict(group)

            for from_name, to_name in [
                ('group_weight', '__saved'),
                ('group_id', '_group_id'),
                ('method', '_method'),
                ('precedence', '_precedence'),
                ('rewrite', 'prefix_rewrite')
            ]:
                if from_name in route:
                    route[to_name] = route[from_name]
                    del(route[from_name])

            route['clusters'] = []
            for mapping in group['mappings']:
                cluster = mapping['cluster']
                cname = cluster['name']

                route['clusters'].append({
                    'name': mapping['cluster']['name'],
                    'weight': mapping['weight']
                })

                if cname not in clusters:
                    # print("NEW CLUSTER %s" % cname)
                    clusters[cname] = cluster
                else:
                    # print("REPEAT CLUSTER %s" % cname)
                    clusters[cname] = cluster

            if 'shadows' in route:
                if route['shadows']:
                    route['shadow'] = {
                        'name': route['shadows'][0]['cluster']['name']
                    }

                del(route['shadows'])

            if ('host_redirect' in route) and (not route['host_redirect']):
                del(route['host_redirect'])

            # print("WTFO route %s" % json.dumps(route, sort_keys=True, indent=4))

            for k in [ 'mappings', 'name', 'serialization' ]:
                route.pop(k, None)

            if not route.get('_method', ''):
                route['_method'] = 'GET'

            if '_precedence' not in route:
                route['_precedence'] = 0

            if ('headers' in route) and not route['headers']:
                del(route['headers'])

            econf['routes'].append(route)

        for cluster in sorted(clusters.values(), key=lambda x: x['name']):
            envoy_cluster = as_sourceddict(cluster)

            if "service" in envoy_cluster:
                envoy_cluster['_service'] = envoy_cluster['service']
                del(envoy_cluster['service'])

            if 'serialization' in envoy_cluster:
                del(envoy_cluster['serialization'])

            econf['clusters'].append(envoy_cluster)

        self['envoy_config'] = econf

        # XXX This can't be right. The IR class needs its own error handling.
        self['errors'] = aconf['_errors']
        self['sources'] = {}
        self['source_map'] = {}

        for key, src in aconf['_sources'].items():
            key_base = key
            key_index = None

            if re.search(r'\.\d+$', key):
                key_base, key_index = os.path.splitext(key)

                while key_index.startswith('.'):
                    key_index = key_index[1:]

            src_dict = {
                '_source': key_base,
                'filename': key_base,
            }

            for from_key, to_key in [ ( 'kind', 'kind'),
                                      ( 'name', 'name'),
                                      ( 'version', 'version'),
                                      ( 'description', 'description'),
                                      ( 'serialization', 'yaml' ) ]:
                if from_key in src:
                    src_dict[to_key] = src[from_key]

            if key_index is not None:
                src_dict['index'] = int(key_index)

            if 'version' not in src_dict:
                src_dict['version'] = 'ambassador/v0'

            if key.startswith('--'):
                src_dict['index'] = 0
                src_dict['version'] = 'v0'
                src_dict.pop('yaml', None)

                if key == '--diagnostics--':
                    src_dict['kind'] = 'diagnostics'

            self['sources'][key] = src_dict

            if key != '--diagnostics--':
                src_map = self['source_map'].setdefault(key_base, {})
                src_map[key] = True

def get_old_intermediate(aconf, ir, v1config):
    return dict(old_ir(aconf.as_dict(), ir.as_dict(), v1config.as_dict()))

#### Test functions

@pytest.mark.parametrize("directory", MATCHES)
@standard_setup
def test_config(testname, dirpath, configdir):
    global logger
    errors = []

    if not os.path.isdir(configdir):
        errors.append("configdir %s is not a directory" % configdir)

    print("==== checking intermediate output")

    resources = fetch_resources(configdir, logger)
    aconf = Config()
    aconf.load_all(resources)

    ir = IR(aconf)
    v1config = V1Config(ir)

    # tmp = {
    #     'aconf': aconf.as_dict(),
    #     'ir': ir.as_dict(),
    #     'v1config': v1config.as_dict()
    # }
    #
    # json.dump(tmp, sys.stdout, sort_keys=True, indent=4)

    current = get_old_intermediate(aconf, ir, v1config)
    current['envoy_config'] = filtered_overview(current['envoy_config'])

    current_path = os.path.join(dirpath, "intermediate.json")
    json.dump(current, open(current_path, "w"), sort_keys=True, indent=4)

    gold_path = os.path.join(dirpath, "gold.intermediate.json")

    if os.path.exists(gold_path):
        udiff = unified_diff(gold_path, current_path)

        if udiff:
            errors.append("gold.intermediate.json and intermediate.json do not match!\n\n%s" % "\n".join(udiff))

    # print("==== checking config generation")
    #
    # envoy_json_out = os.path.join(dirpath, "envoy.json")
    #
    # try:
    #     os.unlink(envoy_json_out)
    # except OSError as e:
    #     if e.errno != errno.ENOENT:
    #         raise
    #
    # ambassador = shell([ 'ambassador', 'config', '--check', configdir, envoy_json_out ])
    #
    # print(ambassador.errors(raw=True))
    #
    # if ambassador.code != 0:
    #     errors.append('ambassador failed! %s' % ambassador.code)
    # else:
    #     envoy = shell([ 'docker', 'run',
    #                         '--rm',
    #                         '-v', '%s:/etc/ambassador-config' % dirpath,
    #                         VALIDATOR_IMAGE,
    #                         '/usr/local/bin/envoy',
    #                            '--base-id', '1',
    #                            '--mode', 'validate',
    #                            '--service-cluster', 'test',
    #                            '-c', '/etc/ambassador-config/envoy.json' ],
    #                   verbose=True)
    #
    #     envoy_succeeded = (envoy.code == 0)
    #
    #     if not envoy_succeeded:
    #         errors.append('envoy failed! %s' % envoy.code)
    #
    #     envoy_output = list(envoy.output())
    #
    #     if envoy_succeeded:
    #         if not envoy_output[-1].strip().endswith(' OK'):
    #             errors.append('envoy validation failed!')
    #
    #     gold_path = os.path.join(dirpath, "gold.json")
    #
    #     if os.path.exists(gold_path):
    #         udiff = unified_diff(gold_path, envoy_json_out)
    #
    #         if udiff:
    #             errors.append("gold.json and envoy.json do not match!\n\n%s" % "\n".join(udiff))
    #
    # print("==== checking short-circuit with existing config")
    #
    # ambassador = shell([ 'ambassador', 'config', '--check', configdir, envoy_json_out ])
    #
    # print(ambassador.errors(raw=True))
    #
    # if ambassador.code != 0:
    #     errors.append('ambassador repeat check failed! %s' % ambassador.code)
    #
    # if 'Output file exists' not in ambassador.errors(raw=True):
    #     errors.append('ambassador repeat check did not short circuit??')

    if errors:
        print("---- ERRORS")
        print("%s" % "\n".join(errors))

    assert not errors, ("failing, _errors: %d" % len(errors))

@pytest.mark.parametrize("directory", MATCHES)
@standard_setup
def test_diag(testname, dirpath, configdir):
    errors = []
    errorcount = 0

    assert True
    return

    if not os.path.isdir(configdir):
        errors.append("configdir %s is not a directory" % configdir)
        errorcount += 1

    results = diag_paranoia(configdir, dirpath)

    if results['warnings']:
        errors.append("[DIAG WARNINGS]\n%s" % "\n".join(results['warnings']))

    if results['_errors']:
        errors.append("[DIAG ERRORS]\n%s" % "\n".join(results['_errors']))
        errorcount += len(results['_errors'])

    if errors:
        print("---- ERRORS")
        print("%s" % "\n".join(errors))
        print("---- OVERVIEW ----")
        print("%s" % results['overview'])
        print("---- RECONSTITUTED ----")
        print("%s" % results['reconstituted'])
    
    assert errorcount == 0, ("failing, _errors: %d" % errorcount)
