package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_binary(
    name = "kubewatch",
    srcs = ["kubewatch.py"],
    deps = [
        "//python/ambassador:VERSION",
        requirement("click"),
        requirement("kubernetes"),
    ],
)

py_binary(
    name = "post_update",
    srcs = ["post_update.py"],
    deps = [
        requirement("requests"),
    ],
)

py_binary(
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
        "//python/ambassador/config:resourcefetcher",
        "//python/ambassador:__init__",
        "//python/ambassador:utils",
    ],
)
