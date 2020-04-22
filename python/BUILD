package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_binary(
    name = "kubewatch",
    srcs = ["kubewatch.py"],
    deps = [
        "//ambassador:VERSION",
        requirement("click"),
        requirement("kubernetes"),
    ],
)

py_library(
    name = "post_update",
    srcs = ["post_update.py"],
    deps = [
        requirement("requests"),
    ],
)

py_library(
    name = "setup",
    srcs = ["setup.py"],
    deps = [
        requirement("setuptools"),
    ],
)

py_binary(
    name = "watch_hook",
    srcs = ["watch_hook.py"],
    deps = [
        "//ambassador/config:resourcefetcher",
        "//ambassador:__init__",
        "//ambassador:utils",
    ],
)
