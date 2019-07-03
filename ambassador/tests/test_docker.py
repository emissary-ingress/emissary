import asyncio
import os

DockerImage = os.environ["AMBASSADOR_DOCKER_IMAGE"]

@asyncio.coroutine
def docker_ready():
    cmd = [ 'docker', 'run', '--rm', '--name', 'ambassador', '-p8888:8080',
            DockerImage, '--demo']

    print(f'Starting demo Ambassador:')
    print(' '.join(cmd))

    subproc_future = asyncio.create_subprocess_exec(*cmd, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE)

    subproc = yield from subproc_future
    status = 0

    while True:
        data = yield from subproc.stdout.readline()
        text = data.decode('utf-8').rstrip()

        if not text:
            print("Ambassador died?")
            status = subproc.returncode

            stdout, stderr = yield from subproc.communicate()

            if stderr:
                print("stderr:")
                print(stderr.decode('utf-8').rstrip())

            break

        if text.startswith('AMBASSADOR'):
            print(f'<-- {text}')

        if text == 'AMBASSADOR DEMO RUNNING':
            print("Done!!")
            break

    return status

@asyncio.coroutine
def docker_kill():
    cmd = [ 'docker', 'kill', 'ambassador' ]

    subproc_future = asyncio.create_subprocess_exec(*cmd, stdout=asyncio.subprocess.PIPE)

    subproc = yield from subproc_future

    yield from subproc.communicate()

async def do_http():
    reader, writer = await asyncio.open_connection('127.0.0.1', 8888)

    query = (
        "GET /qotm/?json=true HTTP/1.1\r\n"
        "Host: localhost\r\n"
        "Connection: close\r\n"
        "\r\n"
    )

    writer.write(query.encode('utf-8'))

    in_headers = True

    while True:
        line = await reader.readline()

        if line:
            line = line.decode('utf-8').rstrip()

        if line:
            if in_headers:
                print(f'HTTP header> {line}')
            else:
                print(f'HTTP body> {line}')
        else:
            if not in_headers:
                break
            else:
                in_headers = False

    writer.close()

async def asynchronicity():
    ready = False
    succeeded = False

    try:
        returncode = await asyncio.wait_for(docker_ready(), timeout=20.0)

        if returncode == 0:
            ready = True
    except asyncio.TimeoutError:
        print('timeout')

    if ready:
        try:
            await do_http()
            succeeded = True
        except Exception as e:
            print(f'Could not do HTTP: {e}')
    else:
        print("Not ready in time")

    await asyncio.wait_for(docker_kill(), timeout=5.0)

    assert succeeded

def test_docker():
    if not DockerImage:
        assert False, f'You must set $AMBASSADOR_DOCKER_IMAGE'
    else:
        loop = asyncio.get_event_loop()
        loop.run_until_complete(asynchronicity())
        loop.close()

if __name__ == '__main__':
    test_docker()

