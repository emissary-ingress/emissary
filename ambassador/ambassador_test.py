import sys

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
    assert os.path.isdir(configdir)

    envoy_json_out = os.path.join(dirpath, "envoy.json")

    ambassador = shell([ 'python', AMBASSADOR, 'config', configdir, envoy_json_out ])

    print("\n".join(ambassador.output()))

    ambassador_succeeded = (ambassador.code == 0)
    assert ambassador_succeeded
    assert False

@pytest.mark.parametrize("directory", MATCHES)
@standard_setup
def test_config(testname, dirpath, configdir):
    assert os.path.isdir(configdir)

    results = diag_paranoia(configdir, dirpath)

    if results['warnings']:
        print("\n".join(['WARNING: %s' % x for x in results['warnings']]))

    if results['errors']:
        print("\n".join(['ERROR: %s' % x for x in results['errors']]))

    assert not results['errors']
