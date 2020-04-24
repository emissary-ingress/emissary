load("@rules_python//python:defs.bzl", "py_library", "py_binary", "py_test")
load("@ambassador_pip//:requirements.bzl", pip_requirement = "requirement")

package(default_visibility = ["//visibility:public"])

py_library(
    name = "library",
    srcs = glob(["**/*.py"]),
    data = glob(["**/*"], exclude=["**/*.py", "**/* *", "BUILD", "WORKSPACE", "configure"]),
    srcs_version = "PY3",
    visibility = ["//visibility:public"],
    imports = ["."],
    deps = [
        pip_requirement("click"),
        pip_requirement("clize"),
        pip_requirement("dpath"),
        pip_requirement("expiringdict"),
        pip_requirement("flask"),
        pip_requirement("gunicorn"),
        pip_requirement("jsonpatch"),
        pip_requirement("jsonschema"),
        pip_requirement("k8s"),
        pip_requirement("kubernetes"),
        pip_requirement("packaging"),
        pip_requirement("pexpect"),
        pip_requirement("protobuf"),
        pip_requirement("pytest"),
        pip_requirement("pyyaml"),
        pip_requirement("requests"),
        pip_requirement("retry"),
        pip_requirement("scout.py"),
        pip_requirement("semantic-version"),
        pip_requirement("setuptools"),
        pip_requirement("urllib"),
    ],
)

py_binary(
    name = "ambassador.py",
    main = "ambassador_cli/ambassador.py",
    srcs = ["ambassador_cli/ambassador.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "ert",
    main = "ambassador_cli/ert.py",
    srcs = ["ambassador_cli/ert.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "grab-snapshots",
    main = "ambassador_cli/grab_snapshots.py",
    srcs = ["ambassador_cli/grab_snapshots.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "mockery",
    main = "ambassador_cli/mockery.py",
    srcs = ["ambassador_cli/mockery.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "diagd",
    main = "ambassador_diag/diagd.py",
    srcs = ["ambassador_diag/diagd.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "kubewatch.py",
    main = "kubewatch.py",
    srcs = ["kubewatch.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "post_update.py",
    main = "post_update.py",
    srcs = ["post_update.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)

py_binary(
    name = "watch_hook.py",
    main = "watch_hook.py",
    srcs = ["watch_hook.py"],
    srcs_version = "PY3",
    python_version = "PY3",
    deps = [":library"],
)
