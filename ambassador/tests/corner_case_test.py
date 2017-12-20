import sys

import difflib
import json
import logging
import functools
import os
import pytest
import uuid

from shell import shell

from ambassador.VERSION import Version

os.environ['SCOUT_DISABLE'] = "1"

DIR = os.path.dirname(__file__)
CORNER_CASE_DIR = os.path.join(DIR, "corner_cases")

def shell_command(testname, argv, must_fail=False, need_stdout=None, need_stderr=None, verbose=True):
    errors = []

    cmd = os.path.basename(argv[0])
    command = shell(argv, verbose=True)

    if must_fail:
        if command.code == 0:
            errors.append("%s: %s succeeded but should have failed?" % (testname, cmd))
    else:
        if command.code != 0:
            errors.append("%s: %s failed (%d)?" % (testname, cmd, command.code))

    if need_stdout:
        command_stdout = command.output(raw=True)

        if need_stdout not in command_stdout:
            errors.append("%s: %s stdout does not contain %s" % (testname, cmd, need_stdout))

    if need_stderr:
        command_stderr = command.errors(raw=True)

        if need_stderr not in command_stderr:
            errors.append("%s: %s stderr does not contain %s" % (testname, cmd, need_stderr))

    if errors:
        print("---- ERRORS")
        print("%s" % "\n".join(errors))

    assert not errors, ("failing, errors: %d" % len(errors))

def test_bad_config_input():
    shell_command("test_bad_config_input",
                  [ 'ambassador', 'config', 'no-such-directory', 'no-such-file' ],
                  must_fail=True,
                  need_stderr='Exception: ERROR ERROR ERROR')

def test_bad_dump_input():
    shell_command("test_bad_dump_input",
                  [ 'ambassador', 'dump', 'no-such-directory' ],
                  must_fail=True,
                  need_stderr='Exception: ERROR ERROR ERROR')

def test_bad_yaml():
    shell_command("test_bad_yaml",
                  [ 'ambassador', 'config', CORNER_CASE_DIR, 'no-such-file' ],
                  need_stderr='ERROR ERROR ERROR Starting with configuration errors')

def test_version():
    shell_command("test_version",
                  [ 'ambassador', '--version' ],
                  need_stdout='Ambassador %s' % Version)

def test_showid():
    install_id = uuid.uuid4().hex.upper()
    os.environ['AMBASSADOR_SCOUT_ID'] = install_id

    os.system("env | sort | grep SCOUT")

    shell_command("test_showid",
                  [ 'ambassador', '--showid' ],
                  need_stdout=install_id)
