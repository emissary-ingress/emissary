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

#load("@io_bazel_rules_docker//python:image.bzl", "py_layer", "py_image")
load("@io_bazel_rules_docker//python:image.bzl", "py_layer")

#load("@io_bazel_rules_docker//python3:image.bzl", "py3_image")
load("//build-aux-local:py.bzl", "py_image")

gazelle(name = "gazelle")

# ambassador ###################################################################

py_image(
    base = "@alpine_glibc_with_packages//image",
    name = ".ambassador.stage0",
    binary = "//python:grab-snapshots",
)

py_image(
    base = ":.ambassador.stage0",
    name = ".ambassador.stage1",
    binary = "//python:ert",
)

py_image(
    base = ":.ambassador.stage1",
    name = ".ambassador.stage2",
    binary = "//python:mockery",
)

py_image(
    base = ":.ambassador.stage2",
    name = ".ambassador.stage3",
    binary = "//python:ambassador",
)

py_image(
    base = ":.ambassador.stage3",
    name = ".ambassador.stage4",
    binary = "//python:diagd",
)

py_image(
    base = ":.ambassador.stage4",
    name = ".ambassador.stage5",
    binary = "//python:post_update.py",
)

py_image(
    base = ":.ambassador.stage5",
    name = ".ambassador.stage6",
    binary = "//python:kubewatch.py",
)

py_image(
    base = ":.ambassador.stage6",
    name = ".ambassador.stage7",
    binary = "//python:watch_hook.py",
)

go_image(
    base = ":.ambassador.stage7",
    name = ".ambassador.stage8",
    binary = "//cmd/watt:watt.for-container",
)

go_image(
    base = ":.ambassador.stage8",
    name = ".ambassador.stage9",
    binary = "//cmd/kubestatus:kubestatus.for-container",
)

container_image(
    base = ":.ambassador.stage9",
    name = "ambassador",
    workdir = "/ambassador",
    entrypoint = None,
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
