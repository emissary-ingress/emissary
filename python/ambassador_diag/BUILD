package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
)

py_binary(
    name = "diagd",
    srcs = ["diagd.py"],
    deps = [
        "//ambassador:constants",
        "//ambassador:utils",
        "//ambassador/config:resourcefetcher",
        "//ambassador:__init__",
        requirement("clize"),
        requirement("expiringdict"),
        requirement("flask"),
        requirement("gunicorn"),
        requirement("jsonpatch"),
        requirement("requests"),
    ],
)
