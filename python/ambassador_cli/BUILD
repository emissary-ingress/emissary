package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
)

py_binary(
    name = "ambassador",
    main = "ambassador_x.py",
    srcs = ["ambassador_x.py"],
    deps = [
        "//ambassador:utils",
        "//ambassador/config:resourcefetcher",
        "//ambassador:__init__",
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
    deps = [requirement("click")],
)

py_binary(
    name = "mockery",
    srcs = ["mockery.py"],
    deps = [
        "//:watch_hook",
        "//ambassador:utils",
        "//ambassador/config:resourcefetcher",
        "//kat:utils",
        "//ambassador:__init__",
        requirement("click"),
    ],
)
