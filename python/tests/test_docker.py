import os

import pexpect
import requests

DockerImage = os.environ.get("AMBASSADOR_DOCKER_IMAGE", None)

child = None    # see docker_start()

def docker_start(logfile) -> bool:
    # Use a global here so that the child process doesn't get killed
    global child

    cmd = f'docker run --rm --name ambassador -p8888:8080 {os.environ["AMBASSADOR_DOCKER_IMAGE"]} --demo'

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
    cmd = f'docker kill ambassador'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

def check_http(logfile) -> bool:
    try:
        response = requests.get('http://localhost:8888/qotm/?json=true', headers={ 'Host': 'localhost' })
        text = response.text

        if response.status_code != 200:
            logfile.write(f'QotM: wanted 200 but got {response.status_code} {text}\n')
            return False

        return True
    except Exception as e:
        logfile.write(f'Could not do HTTP: {e}\n')

        return False

def test_docker():
    test_status = False

    with open('/tmp/test_docker_output', 'w') as logfile:
        if not DockerImage:
            logfile.write('No $AMBASSADOR_DOCKER_IMAGE??\n')
        else:
            if docker_start(logfile):
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
