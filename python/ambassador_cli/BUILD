package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [],
)

py_binary(
    name = "ambassador",
    srcs = ["ambassador.py"],
    deps = [
        "//python/ambassador/config:resourcefetcher",
        "//python/ambassador/envoy:__init__",
        "//python/ambassador:__init__",
        "//python/ambassador:utils",
        requirement("clize"),
    ],
)

py_binary(
    name = "ert",
    srcs = ["ert.py"],
    deps = [
        requirement("click"),
        requirement("dpath"),
    ],
)

py_binary(
    name = "grab_snapshots",
    srcs = ["grab_snapshots.py"],
    deps = [
        requirement("click"),
    ],
)

py_binary(
    name = "mockery",
    srcs = ["mockery.py"],
    deps = [
        "//python/ambassador/config:resourcefetcher",
        "//python/ambassador:__init__",
        "//python/ambassador:utils",
        "//python/kat:utils",
        "//python:watch_hook",
        requirement("click"),
    ],
)
