package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

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
