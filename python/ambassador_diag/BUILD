package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [],
)

py_binary(
    name = "diagd",
    srcs = ["diagd.py"],
    deps = [
        "//python/ambassador/config:resourcefetcher",
        "//python/ambassador/diagnostics:__init__",
        "//python/ambassador:__init__",
        "//python/ambassador:constants",
        "//python/ambassador:utils",
        requirement("clize"),
        requirement("expiringdict"),
        requirement("flask"),
        requirement("gunicorn"),
        requirement("jsonpatch"),
        requirement("requests"),
        requirement("setuptools"),
    ],
)
