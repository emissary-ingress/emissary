import os
import sys
import time
from typing import Optional

import pexpect
import pytest
import requests

DOCKER_IMAGE = os.environ.get("AMBASSADOR_DOCKER_IMAGE", "")

child: Optional[pexpect.spawnbase.SpawnBase] = None  # see docker_start()
ambassador_host: Optional[str] = None  # see docker_start()


def docker_start(logfile) -> bool:
    # Use a global here so that the child process doesn't get killed
    global child

    global ambassador_host

    cmd = (
        f"docker run --rm --name test_docker_ambassador -p 9987:8080 -u8888:0 {DOCKER_IMAGE} --demo"
    )
    ambassador_host = "localhost:9987"

    child = pexpect.spawn(cmd, encoding="utf-8")
    child.logfile = logfile

    i = child.expect([pexpect.EOF, pexpect.TIMEOUT, "AMBASSADOR DEMO RUNNING"])

    if i == 0:
        logfile.write("ambassador died?\n")
        return False
    elif i == 1:
        logfile.write("ambassador timed out?\n")
        return False
    else:
        logfile.write("ambassador running\n")
        return True


def docker_kill(logfile):
    cmd = f"docker kill test_docker_ambassador"

    child = pexpect.spawn(cmd, encoding="utf-8")
    child.logfile = logfile

    child.expect([pexpect.EOF, pexpect.TIMEOUT])


def check_http(logfile) -> bool:
    try:
        logfile.write("QotM: making request\n")
        response = requests.get(
            f"http://{ambassador_host}/qotm/?json=true", headers={"Host": "localhost"}
        )
        text = response.text

        logfile.write(f"QotM: got status {response.status_code}, text {text}\n")

        if response.status_code != 200:
            logfile.write(f"QotM: wanted 200 but got {response.status_code} {text}\n")
            return False

        return True
    except Exception as e:
        logfile.write(f"Could not do HTTP: {e}\n")

        return False


def check_cli() -> bool:
    child = pexpect.spawn(f"docker run --rm --entrypoint ambassador {DOCKER_IMAGE} --help")

    # v_encoded = subprocess.check_output(cmd, stderr=subprocess.STDOUT)
    i = child.expect([pexpect.EOF, pexpect.TIMEOUT, "Usage: ambassador"])

    if i == 0:
        print("ambassador died without usage statement?")
        return False
    elif i == 1:
        print("ambassador timed out without usage statement?")
        return False

    i = child.expect([pexpect.EOF, pexpect.TIMEOUT])

    if i == 0:
        return True
    else:
        print("ambassador timed out after usage statement?")
        return False


def check_grab_snapshots() -> bool:
    child = pexpect.spawn(f"docker run --rm --entrypoint grab-snapshots {DOCKER_IMAGE} --help")

    i = child.expect([pexpect.EOF, pexpect.TIMEOUT, "Usage: grab-snapshots"])

    if i == 0:
        print("grab-snapshots died without usage statement?")
        return False
    elif i == 1:
        print("grab-snapshots timed out without usage statement?")
        return False

    i = child.expect([pexpect.EOF, pexpect.TIMEOUT])

    if i == 0:
        return True
    else:
        print("grab-snapshots timed out after usage statement?")
        return False


def test_cli():
    assert check_cli(), "CLI check failed"


def test_grab_snapshots():
    assert check_grab_snapshots(), "grab-snapshots check failed"


def test_demo():
    test_status = False

    # And this tests that the Ambasasdor can run with the `--demo` argument
    # and run normally with a sample /qotm/ Mapping.
    with open("/tmp/test_docker_output", "w") as logfile:
        if not DOCKER_IMAGE:
            logfile.write("No $AMBASSADOR_DOCKER_IMAGE??\n")
        else:
            if docker_start(logfile):
                logfile.write("Demo started, first check...")

                if check_http(logfile):
                    logfile.write("Sleeping for second check...")
                    time.sleep(10)

                    if check_http(logfile):
                        test_status = True

                docker_kill(logfile)

    if not test_status:
        with open("/tmp/test_docker_output", "r") as logfile:
            for line in logfile:
                print(line.rstrip())

    assert test_status, "test failed"


if __name__ == "__main__":
    pytest.main(sys.argv)
