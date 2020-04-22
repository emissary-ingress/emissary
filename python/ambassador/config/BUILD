package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":acmapping",
        ":acpragma",
        ":acresource",
        ":config",
        ":resourcefetcher",
    ],
)

py_library(
    name = "acmapping",
    srcs = ["acmapping.py"],
    deps = [
        ":acresource",
    ],
)

py_library(
    name = "acpragma",
    srcs = ["acpragma.py"],
    deps = [
        ":acresource",
    ],
)

py_library(
    name = "acresource",
    srcs = ["acresource.py"],
    deps = [
        "//python/ambassador:resource",
    ],
)

py_library(
    name = "config",
    srcs = ["config.py"],
    deps = [
        ":acmapping",
        ":acresource",
        "//python/ambassador:resource",
        "//python/ambassador:utils",
        "//python/multi:__init__",
        requirement("jsonschema"),
        requirement("protobuf"),
        requirement("setuptools"),
    ],
)

py_library(
    name = "resourcefetcher",
    srcs = ["resourcefetcher.py"],
    deps = [
        ":acresource",
        ":config",
        "//python/ambassador:utils",
        requirement("pyyaml"),
    ],
)
