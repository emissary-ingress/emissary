package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_binary(
    name = "Host_pb2",
    srcs = ["Host_pb2.py"],
    deps = [
        requirement("google"),
        requirement("k8s"),
    ],
)
