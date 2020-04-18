package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_binary(
    name = "VERSION",
    srcs = ["VERSION.py"],
)

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":VERSION",
        ":ambscout",
        "//ambassador/config:__init__",
        "//ambassador/diagnostics:__init__",
        "//ambassador/envoy:__init__",
        "//ambassador/ir:__init__",
    ],
)

py_library(
    name = "ambscout",
    srcs = ["ambscout.py"],
    deps = [
        ":VERSION",
        requirement("scout.py"),
        requirement("semantic_version"),
    ],
)

py_library(
    name = "constants",
    srcs = ["constants.py"],
)

py_library(
    name = "resource",
    srcs = ["resource.py"],
    deps = [
        ":utils",
    ],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
    deps = [
        ":VERSION",
        requirement("requests"),
        requirement("pyyaml"),
    ],
)
