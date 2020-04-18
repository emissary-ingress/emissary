package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":v2config",
    ],
)

py_library(
    name = "v2_static_resources",
    srcs = ["v2_static_resources.py"],
)

py_library(
    name = "v2admin",
    srcs = ["v2admin.py"],
)

py_library(
    name = "v2bootstrap",
    srcs = ["v2bootstrap.py"],
    deps = [
        "//ambassador/ir:__init__",
        ":v2cluster",
    ],
)

py_library(
    name = "v2cluster",
    srcs = ["v2cluster.py"],
    deps = [
        "//ambassador/ir:__init__",
        ":v2tls",
    ],
)

py_library(
    name = "v2config",
    srcs = ["v2config.py"],
    deps = [
        "//ambassador/envoy:common",
        "//ambassador/ir:__init__",
        ":v2_static_resources",
        ":v2admin",
        ":v2bootstrap",
        ":v2cluster",
        ":v2listener",
        ":v2ratelimit",
        ":v2route",
        ":v2tracing",
    ],
)

py_library(
    name = "v2listener",
    srcs = ["v2listener.py"],
    deps = [
        "//ambassador/ir:__init__",
        "//multi:__init__",
        "//ambassador:utils",
        ":v2route",
        ":v2tls",
    ],
)

py_library(
    name = "v2ratelimit",
    srcs = ["v2ratelimit.py"],
    deps = [
        "//ambassador/ir:__init__",
    ],
)

py_library(
    name = "v2ratelimitaction",
    srcs = ["v2ratelimitaction.py"],
)

py_library(
    name = "v2route",
    srcs = ["v2route.py"],
    deps = [
        "//ambassador/envoy:common",
        "//ambassador/ir:__init__",
        ":v2ratelimitaction",
    ],
)

py_library(
    name = "v2tls",
    srcs = ["v2tls.py"],
    deps = [
        "//ambassador/ir:__init__",
    ],
)

py_library(
    name = "v2tracing",
    srcs = ["v2tracing.py"],
    deps = [
        "//ambassador/ir:__init__",
    ],
)
