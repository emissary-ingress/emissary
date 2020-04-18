package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        "//ambassador/config:acmapping",
        "//ambassador/config:acpragma",
        "//ambassador/config:acresource",
        "//ambassador/config:config",
        "//ambassador/config:resourcefetcher",
    ],
)

py_library(
    name = "acmapping",
    srcs = ["acmapping.py"],
    deps = [
        "//ambassador/config:acresource",
    ],
)

py_library(
    name = "acpragma",
    srcs = ["acpragma.py"],
    deps = [
        "//ambassador/config:acresource",
    ],
)

py_library(
    name = "acresource",
    srcs = ["acresource.py"],
    deps = [
        "//ambassador:resource",
    ],
)

py_library(
    name = "config",
    srcs = ["config.py"],
    deps = [
        "//ambassador/config:acmapping",
        "//ambassador/config:acresource",
        "//ambassador:utils",
        "//ambassador:resource",
        "//multi:__init__",
        requirement("protobuf"),
        requirement("jsonschema"),
    ],
)

py_library(
    name = "resourcefetcher",
    srcs = ["resourcefetcher.py"],
    deps = [
        "//ambassador/config:acresource",
        "//ambassador/config:config",
        "//ambassador:utils",
        requirement("pyyaml"),
    ],
)
