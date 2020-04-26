load("@rules_python//python:defs.bzl", "py_library", "py_binary", "py_test")
load(":requirements.bzl", "all_pip_requirements")

package(default_visibility = ["//visibility:public"])

py_library(
    name = "library",
    srcs = glob(["**/*.py"], exclude=["kat/**", "**/tests/**"]),
    data = glob([
        "templates/**",
        "schemas/**",
    ]),
    srcs_version = "PY3",
    visibility = ["//visibility:public"],
    imports = ["."],
    deps = all_pip_requirements,
)

py_binary(
    name = "ambassador.exe",
    main = "ambassador_cli/ambassador.py",
    srcs = ["ambassador_cli/ambassador.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "ert.exe",
    main = "ambassador_cli/ert.py",
    srcs = ["ambassador_cli/ert.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "grab-snapshots.exe",
    main = "ambassador_cli/grab_snapshots.py",
    srcs = ["ambassador_cli/grab_snapshots.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "mockery.exe",
    main = "ambassador_cli/mockery.py",
    srcs = ["ambassador_cli/mockery.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "diagd.exe",
    main = "ambassador_diag/diagd.py",
    srcs = ["ambassador_diag/diagd.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "kubewatch.exe",
    main = "kubewatch.py",
    srcs = ["kubewatch.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "post_update.exe",
    main = "post_update.py",
    srcs = ["post_update.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "watch_hook.exe",
    main = "watch_hook.py",
    srcs = ["watch_hook.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

exports_files([
    "entrypoint.sh",
])
