# Load Testing APro

The `./bin_linux_amd64/max-load` program is the basis of my
load-testing efforts.  It is built on top of the library form of
[vegeta][].  It will attempt to determine latency as a function of
RPS, and determine the maximum RPS that the service can support.  The
`./bin_linux_amd64/max-load --help` text should be helpful.

[vegeta]: https://github.com/tsenart/vegeta

The `./loadtest.sh` script calls `max-load` with a variety of parameters
to test a buncha situations.

## Locust

The `./bin_xxx_amd64/locust-slave` program is a [Locust][] slave that
runs gRPC queries against the Lyft RLS and the no-op gRPC ExtAuth in
`cmd/model-cluster-load-grpc-auth`. This tool has not been added to the
`loadtest.sh` script (yet?).

[Locust]: https://docs.locust.io/en/stable/index.html

To use it, first create a `dummy.py` file somewhere.

```python
from locust import Locust, TaskSet, task
class MyTaskSet(TaskSet):
    @task(20)
    def hello(self):
        pass
class Dummy(Locust):
    task_set = MyTaskSet
```

Next, launch everything in separate terminals. These examples assume that
you are starting in the `apro` directory.

```console
1 $ locust --master -f /path/to/dummy.py

2 $ bin_darwin_amd64/model-cluster-grpc-auth

3 $ docker run -d -p 6379:6379 redis

4 $ mkdir -p /tmp/config/config
4 $ env REDIS_URL=localhost:6379 REDIS_SOCKET_TYPE=tcp USE_STATSD=false RUNTIME_ROOT=/tmp/config RUNTIME_SUBDIRECTORY=config PORT=7000 bin_darwin_amd64/amb-sidecar ratelimit

5 $ bin_darwin_amd64/locust-slave
```

Finally, access [Locust's Web UI](http://localhost:8089/). From there you
can launch a new swarm to hit the services. I suggest launching just two
workers as a starting point.
