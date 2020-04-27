# gazelle:prefix github.com/datawire/ambassador
# gazelle:build_file_name BUILD
# gazelle:proto disable_global
# gazelle:exclude cxx
# gazelle:exclude build-aux
# gazelle:exclude python
# gazelle:map_kind go_binary go_binary //build-aux-local:go.bzl

load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_docker//container:container.bzl", "container_image", "container_push")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")
load("@io_bazel_rules_docker//docker/util:run.bzl", "container_run_and_extract")

#load("@io_bazel_rules_docker//python:image.bzl", "py_layer", "py_image")
load("@io_bazel_rules_docker//python:image.bzl", "py_layer")

#load("@io_bazel_rules_docker//python3:image.bzl", "py3_image")
load("//build-aux-local:py.bzl", "py_image")
#load("//build-aux-local:misc.bzl", "my_image", "piece_binary")

gazelle(name = "gazelle")

# my_image(
#     name = "test-image",
#     base = "@alpine_glibc_with_packages//image",
#     pieces = [
#         piece_binary(name = "kat-server", binary = "//cmd/kat-server:kat-server.for-container"),
#         piece_binary(name = "kat-client", binary = "//cmd/kat-server:kat-client.for-container"),
#     ],
# )

# ambassador ###################################################################

# Surely there's a better way to get the file out of the image...
container_run_and_extract(
    name = "envoy.exe",
    commands = ["true"],
    extract_file = "/usr/local/bin/envoy-static-stripped",
    image = "@base_envoy//image",
)

# polyglot_image(
#     name = "ambassador",
#     pieces = [
#         py_distribution(
#             lib = "//python:library",
#             #'ambassador=ambassador_cli.ambassador:main',
#             #'diagd=ambassador_diag.diagd:main',
#             #'mockery=ambassador_cli.mockery:main',
#             #'grab-snapshots=ambassador_cli.grab_snapshots:main',
#             #'ert=ambassador_cli.ert:main'
#         ),
#         go_binary("//cmd/ambassador:ambassador.for-container"),
#     ],
# )
# py_library()

py_layer(
    name = ".py-library",
    deps = ["//python:library"],
)

py_image(
    base = "@alpine_glibc_with_packages//image",
    name = ".ambassador.stage0",
    binary = "//python:grab-snapshots.exe",
)

py_image(
    base = ":.ambassador.stage0",
    name = ".ambassador.stage1",
    binary = "//python:ert.exe",
)

py_image(
    base = ":.ambassador.stage1",
    name = ".ambassador.stage2",
    binary = "//python:mockery.exe",
)

py_image(
    base = ":.ambassador.stage2",
    name = ".ambassador.stage3",
    binary = "//python:ambassador.exe",
)

py_image(
    base = ":.ambassador.stage3",
    name = ".ambassador.stage4",
    binary = "//python:diagd.exe",
)

py_image(
    base = ":.ambassador.stage4",
    name = ".ambassador.stage5",
    binary = "//python:post_update.exe",
)

py_image(
    base = ":.ambassador.stage5",
    name = ".ambassador.stage6",
    binary = "//python:kubewatch.exe",
)

py_image(
    base = ":.ambassador.stage6",
    name = ".ambassador.stage7",
    binary = "//python:watch_hook.exe",
)

go_image(
    base = ":.ambassador.stage7",
    name = ".ambassador.stage8",
    binary = "//cmd/ambassador:ambassador.for-container",
)

container_image(
    base = ":.ambassador.stage8",
    name = "ambassador",
    # files to add
    directory = "/usr/local/bin",
    mode = "0o755",
    files = [
        "//python:entrypoint.sh",
    ],
    # symlinks to add
    symlinks = {
        # Add sane names for the silly Bazel names
        "/usr/local/bin/ambassador.py": "/app/python/ambassador.exe",
        "/usr/local/bin/ert": "/app/python/ert.exe",
        "/usr/local/bin/kubewatch.py": "/app/python/kubewatch.exe",
        "/usr/local/bin/post_update.py": "/app/python/post_update.exe",
        "/usr/local/bin/diagd": "/app/python/diagd.exe",
        "/usr/local/bin/grab-snapshots": "/app/python/grab-snapshots.exe",
        "/usr/local/bin/mockery": "/app/python/mockery.exe",
        "/usr/local/bin/watch_hook.py": "/app/python/watch_hook.exe",
        "/usr/local/bin/ambassador": "/app/cmd/ambassador/ambassador.for-container",
        # Multi-call binary
        "/usr/local/bin/ambex": "/usr/local/bin/ambassador",
        "/usr/local/bin/kubestatus": "/usr/local/bin/ambassador",
        "/usr/local/bin/watt": "/usr/local/bin/ambassador",
        # Bazel's launcher scripts use 'python'
        "/usr/bin/python": "python3",
    },
    # runtime info
    workdir = "/ambassador",
    entrypoint = None,
    cmd = ["entrypoint.sh"],
)

container_push(
    name = "ambassador.push",
    image = ":ambassador",
    format = "Docker",
    registry = "docker.io/lukeshu",
    repository = "ambassador",
    tag = "dev",
)

# kat-client ###################################################################

go_image(
    name = ".kat-client.stage0",
    base = "@alpine_glibc//image",
    binary = "//cmd/kat-client:kat-client.for-container",
)

container_image(
    name = "kat-client",
    base = ":.kat-client.stage0",
    # symlinks to add
    symlinks = {
        "/usr/local/bin/kat-client": "/app/cmd/kat-client/kat-client.for-container",
        "/work/kat_client": "/usr/local/bin/kat-client",
    },
    # runtime info
    workdir = "/work",
    entrypoint = None,
    cmd = [
        "sleep",
        "3600",
    ],
)

container_push(
    name = "kat-client.push",
    image = ":kat-client",
    format = "Docker",
    registry = "docker.io/lukeshu",
    repository = "kat-client",
    tag = "dev",
)

# kat-server ###################################################################

go_image(
    name = ".kat-server.stage0",
    base = "@alpine_glibc//image",
    binary = "//cmd/kat-server:kat-server.for-container",
)

container_image(
    name = "kat-server",
    base = ":.kat-server.stage0",
    # files to add
    directory = "/work",
    mode = "0o644",
    files = [
        ":server.crt",
        ":server.key",
    ],
    # symlinks to add
    symlinks = {
        "/usr/local/bin/kat-server": "/app/cmd/kat-server/kat-server.for-container",
    },
    # runtime info
    env = {
        "GRPC_VERBOSITY": "debug",
        "GRPC_TRACE": "tcp,http,api",
    },
    workdir = "/work",
    entrypoint = None,
    cmd = ["kat-server"],
)

container_push(
    name = "kat-server.push",
    image = ":kat-server",
    format = "Docker",
    registry = "docker.io/lukeshu",
    repository = "kat-server",
    tag = "dev",
)
