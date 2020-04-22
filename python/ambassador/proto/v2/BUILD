package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "Host_pb2",
    srcs = ["Host_pb2.py"],
    deps = [
        requirement("k8s"),
        requirement("protobuf"),
    ],
)
