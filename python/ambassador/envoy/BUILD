package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":common",
        "//python/ambassador/envoy/v2:__init__",
    ],
)

py_library(
    name = "common",
    srcs = ["common.py"],
    deps = [
        "//python/ambassador/ir:__init__",
        "//python/ambassador/ir:irhttpmappinggroup",
    ],
)
