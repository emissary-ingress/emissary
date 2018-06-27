#!/usr/bin/env python

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

from concurrent import futures
import time
import os

import grpc

from grpc_reflection.v1alpha import reflection
from grpc_reflection.v1alpha import reflection_pb2_grpc

import helloworld_pb2
import helloworld_pb2_grpc

listen_address = '[::]:50051'
_ONE_DAY_IN_SECONDS = 60 * 60 * 24


class Greeter(helloworld_pb2_grpc.GreeterServicer):

    def SayHello(self, request, context):
        return helloworld_pb2.HelloReply(message='Hello, {} (host: {})!'.format(request.name, os.getenv("HOSTNAME")))


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    greeter = Greeter()
    helloworld_pb2_grpc.add_GreeterServicer_to_server(greeter, server)
    server.add_insecure_port(listen_address)

    reflection.enable_server_reflection(["helloworld.Greeter"], server)
    server.start()

    channel = grpc.insecure_channel('localhost:%d' % 50051)
    reflection_pb2_grpc.ServerReflectionStub(channel)

    try:
        print("Server started, listening on: {}".format(listen_address))
        while True:
            time.sleep(_ONE_DAY_IN_SECONDS)
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == '__main__':
    serve()
