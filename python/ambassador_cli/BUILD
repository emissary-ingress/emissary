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
        "//ambassador/config:resourcefetcher",
        "//ambassador/envoy:__init__",
        "//ambassador:__init__",
        "//ambassador:utils",
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
        "//:watch_hook",
        "//ambassador/config:resourcefetcher",
        "//ambassador:__init__",
        "//ambassador:utils",
        "//kat:utils",
        requirement("click"),
    ],
)
