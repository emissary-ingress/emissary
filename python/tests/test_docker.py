import os

import pexpect
import requests
import time

DockerImage = os.environ.get("AMBASSADOR_DOCKER_IMAGE", None)

child = None    # see docker_start()

def docker_start(logfile) -> bool:
    # Use a global here so that the child process doesn't get killed
    global child

    print(os.environ["DOCKER_NETWORK"])

    cmd = f'docker run --rm --name test_docker_ambassador --network {os.environ["DOCKER_NETWORK"]} --network-alias docker-ambassador -u8888:0 {os.environ["AMBASSADOR_DOCKER_IMAGE"]} --demo'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, 'AMBASSADOR DEMO RUNNING' ])

    if i == 0:
        print('ambassador died?')
        return False
    elif i == 1:
        print('ambassador timed out?')
        return False
    else:
        return True

def docker_kill(logfile):
    cmd = f'docker kill test_docker_ambassador'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

def check_http(logfile) -> bool:
    try:
        logfile.write("QotM: making request\n")
        response = requests.get('http://docker-ambassador:8080/qotm/?json=true', headers={ 'Host': 'localhost' })
        text = response.text

        logfile.write(f"QotM: got status {response.status_code}, text {text}\n")

        if response.status_code != 200:
            logfile.write(f'QotM: wanted 200 but got {response.status_code} {text}\n')
            return False

        return True
    except Exception as e:
        logfile.write(f'Could not do HTTP: {e}\n')

        return False

def clicheck() -> bool:
    child = pexpect.spawn("ambassador --help")

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, "Usage: ambassador" ])

    if i == 0:
        print('ambassador died without usage statement?')
        return False
    elif i == 1:
        print('ambassador timed out without usage statement?')
        return False

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

    if i == 0:
        return True
    else:
        print("ambassador timed out after usage statement?")
        return False


def test_docker():
    test_status = False

    # We're running in the build container here, so the Ambassador CLI should work.
    assert clicheck(), "CLI check failed"

    with open('/tmp/test_docker_output', 'w') as logfile:
        if not DockerImage:
            logfile.write('No $AMBASSADOR_DOCKER_IMAGE??\n')
        else:
            if docker_start(logfile):
                logfile.write("Demo started, sleeping 10 seconds just in case...")
                time.sleep(10)

                if check_http(logfile):
                    test_status = True

                docker_kill(logfile)

    if not test_status:
        with open('/tmp/test_docker_output', 'r') as logfile:
            for line in logfile:
                print(line.rstrip())

    assert test_status, 'test failed'

if __name__ == '__main__':
    test_docker()
