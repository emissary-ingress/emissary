import sys

import difflib
import errno
import json
import logging
import functools
import os
import pytest

from shell import shell

from diag_paranoia import diag_paranoia, filtered_overview, sanitize_errors

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

#### Test functions

@pytest.mark.parametrize("directory", MATCHES)
@standard_setup
def test_config(testname, dirpath, configdir):
    errors = []

    if not os.path.isdir(configdir):
        errors.append("configdir %s is not a directory" % configdir)

    print("==== checking intermediate output")

    ambassador = shell([ 'ambassador', 'dump', configdir ])

    if ambassador.code != 0:
        errors.append('ambassador dump failed! %s' % ambassador.code)
    else:
        current_raw = ambassador.output(raw=True)
        current = None
        gold = None

        try:
            current = sanitize_errors(json.loads(current_raw))
        except json.decoder.JSONDecodeError as e:
            errors.append("current intermediate was unparseable?")

        if current:
            current['envoy_config'] = filtered_overview(current['envoy_config'])

            current_path = os.path.join(dirpath, "intermediate.json")
            json.dump(current, open(current_path, "w"), sort_keys=True, indent=4)

            gold_path = os.path.join(dirpath, "gold.intermediate.json")

            if os.path.exists(gold_path):
                udiff = unified_diff(gold_path, current_path)

                if udiff:
                    errors.append("gold.intermediate.json and intermediate.json do not match!\n\n%s" % "\n".join(udiff))

    print("==== checking config generation")

    envoy_json_out = os.path.join(dirpath, "envoy.json")

    try:
        os.unlink(envoy_json_out)
    except OSError as e:
        if e.errno != errno.ENOENT:
            raise

    ambassador = shell([ 'ambassador', 'config', '--check', configdir, envoy_json_out ])

    print(ambassador.errors(raw=True))    

    if ambassador.code != 0:
        errors.append('ambassador failed! %s' % ambassador.code)
    else:
        envoy = shell([ 'docker', 'run', 
                            '--rm',
                            '-v', '%s:/etc/ambassador-config' % dirpath,
                            VALIDATOR_IMAGE,
                            '/usr/local/bin/envoy',
                               '--base-id', '1',
                               '--mode', 'validate',
                               '--service-cluster', 'test',
                               '-c', '/etc/ambassador-config/envoy.json' ],
                      verbose=True)

        envoy_succeeded = (envoy.code == 0)

        if not envoy_succeeded:
            errors.append('envoy failed! %s' % envoy.code)

        envoy_output = list(envoy.output())

        if envoy_succeeded:
            if not envoy_output[-1].strip().endswith(' OK'):
                errors.append('envoy validation failed!')

        gold_path = os.path.join(dirpath, "gold.json")

        if os.path.exists(gold_path):
            udiff = unified_diff(gold_path, envoy_json_out)

            if udiff:
                errors.append("gold.json and envoy.json do not match!\n\n%s" % "\n".join(udiff))

    print("==== checking short-circuit with existing config")

    ambassador = shell([ 'ambassador', 'config', '--check', configdir, envoy_json_out ])

    print(ambassador.errors(raw=True))

    if ambassador.code != 0:
        errors.append('ambassador repeat check failed! %s' % ambassador.code)

    if 'Output file exists' not in ambassador.errors(raw=True):
        errors.append('ambassador repeat check did not short circuit??')

    if errors:
        print("---- ERRORS")
        print("%s" % "\n".join(errors))

    assert not errors, ("failing, errors: %d" % len(errors))

@pytest.mark.parametrize("directory", MATCHES)
@standard_setup
def test_diag(testname, dirpath, configdir):
    errors = []
    errorcount = 0

    if not os.path.isdir(configdir):
        errors.append("configdir %s is not a directory" % configdir)
        errorcount += 1

    results = diag_paranoia(configdir, dirpath)

    if results['warnings']:
        errors.append("[DIAG WARNINGS]\n%s" % "\n".join(results['warnings']))

    if results['errors']:
        errors.append("[DIAG ERRORS]\n%s" % "\n".join(results['errors']))
        errorcount += len(results['errors'])

    if errors:
        print("---- ERRORS")
        print("%s" % "\n".join(errors))
        print("---- OVERVIEW ----")
        print("%s" % results['overview'])
        print("---- RECONSTITUTED ----")
        print("%s" % results['reconstituted'])
    
    assert errorcount == 0, ("failing, errors: %d" % errorcount)
