package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":diagnostics",
        ":envoy_stats",
    ],
)

py_library(
    name = "diagnostics",
    srcs = ["diagnostics.py"],
    deps = [
        ":envoy_stats",
        "//ambassador/envoy:__init__",
        "//ambassador/ir:__init__",
        "//ambassador/ir:irbasemappinggroup",
        "//ambassador/ir:irhttpmappinggroup",
    ],
)

py_library(
    name = "envoy_stats",
    srcs = ["envoy_stats.py"],
    deps = [
        requirement("flask"),
        requirement("requests"),
    ],
)
