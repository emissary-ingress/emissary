# gazelle:prefix github.com/datawire/ambassador
# gazelle:build_file_name BUILD
# gazelle:proto disable_global
# gazelle:external vendored
# gazelle:exclude cxx
# gazelle:exclude build-aux

load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_docker//container:container.bzl", "container_image")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")

gazelle(name = "gazelle")

# container_image(
#     name = "ambassador",
#     base = "@alpine_glibc//image",
#     files = ["//java/com/example/app:Hello_deploy.jar"],
#     cmd = ["Hello_deploy.jar"]
# )

# container_image(
#     name = "kat-client",
#     base = "@alpine_glibc//image",
#     files = ["//java/com/example/app:Hello_deploy.jar"],
#     cmd = ["Hello_deploy.jar"]
# )

# kat-client ###################################################################

go_image(
    name = "_kat-client_stage0",
    base = "@alpine_glibc//image",
    embed = ["//cmd/kat-client:go_default_library"],
)

container_image(
    name = "kat-client",
    base = "_kat-client_stage0",

    symlinks = {
        "/usr/local/bin/kat-client": "/app/kat-client.binary",
        "/work/kat_client": "/app/kat-client.binary",
    },

    workdir = "/work",
    entrypoint = None,
    cmd = [ "sleep", "3600" ],
)

# kat-server ###################################################################

go_image(
    name = "_kat-server_stage0",
    base = "@alpine_glibc//image",
    embed = ["//cmd/kat-server:go_default_library"],
)

container_image(
    name = "kat-server",
    base = "_kat-server_stage0",

    directory = "/work",
    mode = "0o644",
    files = [":server.crt", ":server.key"],

    symlinks = {
        "/usr/local/bin/kat-server": "/app/kat-server.binary",
    },

    workdir = "/work",
    entrypoint = None,
    cmd = [ "kat-server" ],
)

container_image(
    name = "kat-server-test",
    base = "@alpine_glibc//image",

    files = ["//cmd/kat-server:kat-server"],
)
