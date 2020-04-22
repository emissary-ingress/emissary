package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [],
)

py_library(
    name = "harness",
    srcs = ["harness.py"],
    deps = [
        ":parser",
        ":utils",
        "//multi:__init__",
        requirement("packaging"),
        requirement("pytest"),
        requirement("pyyaml"),
    ],
)

py_library(
    name = "parser",
    srcs = ["parser.py"],
    deps = [
        requirement("pyyaml"),
    ],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
    deps = [],
)
