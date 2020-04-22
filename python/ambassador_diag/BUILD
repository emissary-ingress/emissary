package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [],
)

py_binary(
    name = "diagd",
    srcs = ["diagd.py"],
    deps = [
        "//ambassador/config:resourcefetcher",
        "//ambassador:__init__",
        "//ambassador:constants",
        "//ambassador:utils",
        requirement("clize"),
        requirement("expiringdict"),
        requirement("flask"),
        requirement("gunicorn"),
        requirement("jsonpatch"),
        requirement("requests"),
    ],
)
