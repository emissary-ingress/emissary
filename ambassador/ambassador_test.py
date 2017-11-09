import sys

import difflib
import json
import logging
import functools
import os
import pytest

from shell import shell

from diag_paranoia import diag_paranoia

VALIDATOR_IMAGE = "dwflynn/ambassador-envoy:v1.4.0-49-g008635a04"

DIR = os.path.dirname(__file__)
EXCLUDES = [ "__pycache__" ] 

AMBASSADOR = os.path.join(DIR, "ambassador.py")
TESTDIR = os.path.join(DIR, "tests")
DEFAULT_CONFIG = os.path.join(DIR, "default-config")
MATCHES = [ n for n in os.listdir(TESTDIR) if (os.path.isdir(os.path.join(TESTDIR, n)) and (n not in EXCLUDES)) ]

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

#### Test functions

@pytest.mark.parametrize("directory", MATCHES)
@standard_setup
def test_config(testname, dirpath, configdir):
    errors = []

    if not os.path.isdir(configdir):
        errors.append("configdir %s is not a directory" % configdir)

    envoy_json_out = os.path.join(dirpath, "envoy.json")

    ambassador = shell([ 'python', AMBASSADOR, 'config', configdir, envoy_json_out],
                       verbose=True)

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
                               '-c', '/etc/ambassador-config/envoy.json' ],
                      verbose=True)

        envoy_succeeded = (envoy.code == 0)
        print("envoy code %d" % envoy.code)
        print("envoy succeeded %d" % envoy_succeeded)

        if not envoy_succeeded:
            errors.append('envoy failed! %s' % envoy.code)

        envoy_output = list(envoy.output())

        if envoy_succeeded:
            if not envoy_output[-1].strip().endswith(' OK'):
                errors.append('envoy validation failed!')

        gold_path = os.path.join(dirpath, "gold.json")

        if os.path.exists(gold_path):
            gold = json.dumps(json.load(open(gold_path, "r")), indent=4, sort_keys=True)
            current = json.dumps(json.load(open(envoy_json_out, "r")), indent=4, sort_keys=True)

            udiff = list(difflib.unified_diff(gold.split("\n"), current.split("\n"),
                                              fromfile="gold.json", tofile="envoy.json",
                                              lineterm=""))

            if udiff:
                errors.append("gold.json and envoy.json do not match!\n\n%s" % "\n".join(udiff))

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

    assert errorcount == 0, ("failing, errors: %d" % errorcount)
