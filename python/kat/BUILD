package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
)

py_library(
    name = "harness",
    srcs = ["harness.py"],
    deps = [
        "//multi:__init__",
        requirement("packaging"),
        ":parser",
        requirement("pytest"),
        ":utils",
        requirement("pyyaml"),
    ],
)

py_library(
    name = "parser",
    srcs = ["parser.py"],
    deps = [requirement("pyyaml")],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
)
