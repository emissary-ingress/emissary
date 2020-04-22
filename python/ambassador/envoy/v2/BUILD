package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

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
    deps = [],
)

py_library(
    name = "v2admin",
    srcs = ["v2admin.py"],
    deps = [],
)

py_library(
    name = "v2bootstrap",
    srcs = ["v2bootstrap.py"],
    deps = [
        ":v2cluster",
        "//python/ambassador/ir:ircluster",
        "//python/ambassador/ir:irlogservice",
        "//python/ambassador/ir:irratelimit",
        "//python/ambassador/ir:irtracing",
    ],
)

py_library(
    name = "v2cluster",
    srcs = ["v2cluster.py"],
    deps = [
        ":v2tls",
        "//python/ambassador/ir:ircluster",
    ],
)

py_library(
    name = "v2config",
    srcs = ["v2config.py"],
    deps = [
        ":v2_static_resources",
        ":v2admin",
        ":v2bootstrap",
        ":v2cluster",
        ":v2listener",
        ":v2ratelimit",
        ":v2route",
        ":v2tracing",
        "//python/ambassador/envoy:common",
        "//python/ambassador/ir:__init__",
    ],
)

py_library(
    name = "v2listener",
    srcs = ["v2listener.py"],
    deps = [
        ":v2route",
        ":v2tls",
        "//python/ambassador/ir:irauth",
        "//python/ambassador/ir:irbuffer",
        "//python/ambassador/ir:ircluster",
        "//python/ambassador/ir:ircors",
        "//python/ambassador/ir:irfilter",
        "//python/ambassador/ir:irgzip",
        "//python/ambassador/ir:irlistener",
        "//python/ambassador/ir:irratelimit",
        "//python/ambassador/ir:irtcpmappinggroup",
        "//python/ambassador/ir:irtlscontext",
        "//python/ambassador:utils",
        "//python/multi:__init__",
    ],
)

py_library(
    name = "v2ratelimit",
    srcs = ["v2ratelimit.py"],
    deps = [
        "//python/ambassador/ir:irratelimit",
    ],
)

py_library(
    name = "v2ratelimitaction",
    srcs = ["v2ratelimitaction.py"],
    deps = [],
)

py_library(
    name = "v2route",
    srcs = ["v2route.py"],
    deps = [
        ":v2ratelimitaction",
        "//python/ambassador/envoy:common",
        "//python/ambassador/ir:irbasemapping",
        "//python/ambassador/ir:irhttpmappinggroup",
    ],
)

py_library(
    name = "v2tls",
    srcs = ["v2tls.py"],
    deps = [
        "//python/ambassador/ir:irtlscontext",
    ],
)

py_library(
    name = "v2tracing",
    srcs = ["v2tracing.py"],
    deps = [
        "//python/ambassador/ir:irtracing",
    ],
)
