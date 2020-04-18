package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":common",
        "//ambassador/envoy/v2:__init__",
    ],
)

py_library(
    name = "common",
    srcs = ["common.py"],
    deps = [
        "//ambassador/ir:__init__",
    ],
)
