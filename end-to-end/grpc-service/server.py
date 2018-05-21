#!/usr/bin/env python

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
