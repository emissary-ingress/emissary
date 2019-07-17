import sys
import os

import pexpect
import requests

DockerImage = os.environ["AMBASSADOR_DOCKER_IMAGE"]
child = None    # see docker_start()

def docker_start() -> bool:
    # Use a global here so that the child process doesn't get killed
    global child

    cmd = f'docker run --rm --name ambassador -p8888:8080 {os.environ["AMBASSADOR_DOCKER_IMAGE"]} --demo'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = sys.stdout

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, 'AMBASSADOR DEMO RUNNING' ])

    if i == 0:
        print('ambassador died?')
        return False
    elif i == 1:
        print('ambassador timed out?')
        return False
    else:
        return True

def docker_kill():
    cmd = f'docker kill ambassador'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = sys.stdout

    child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

def check_http() -> bool:
    try:
        response = requests.get('http://localhost:8888/qotm/?json=true', headers={ 'Host': 'localhost' })
        text = response.text

        if response.status_code != 200:
            print(f'QotM: wanted 200 but got {response.status_code} {text}')
            return False

        return True
    except Exception as e:
        print(f'Could not do HTTP: {e}')

        return False

def test_docker():
    test_status = True

    if not DockerImage:
        assert False, f'You must set $AMBASSADOR_DOCKER_IMAGE'
    else:
        assert docker_start(), 'ambassador could not start'

        if not check_http():
            test_status = False

        docker_kill()

        assert test_status, 'test failed'

if __name__ == '__main__':
    test_docker()

