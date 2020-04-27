load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# Go ###########################################################################

git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    commit = "3edc6d5417aedaa82b3c042ae8b1fd08f155aa0d",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

# Go: Generate BUILD files #####################################################

http_archive(
    name = "bazel_gazelle",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
    ],
    sha256 = "d8c45ee70ec39a57e7a05e5027c32b1576cc7f16d9dd37135b0eddde45cf1b10",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
gazelle_dependencies()

# Docker #######################################################################

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "dc97fccceacd4c6be14e800b2a00693d5e8d07f69ee187babfd04a80a9f8e250",
    strip_prefix = "rules_docker-0.14.1",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.14.1/rules_docker-v0.14.1.tar.gz"],
)

load("@io_bazel_rules_docker//repositories:repositories.bzl", docker_repos = "repositories")
docker_repos()

load("@io_bazel_rules_docker//repositories:deps.bzl", docker_deps = "deps")
docker_deps()

load("@io_bazel_rules_docker//container:container.bzl", "container_pull")
container_pull(
    name = "alpine_glibc",
    registry = "docker.io",
    repository = "frolvlad/alpine-glibc",
    tag = "alpine-3.10" # TODO: consider using 'digest' instead of 'tag'
)
load("@io_bazel_rules_docker//container:container.bzl", "container_pull")
container_pull(
    name = "built_ambassador",
    registry = "quay.io",
    repository = "datawire/ambassador",
    tag = "1.4.2",
)
container_pull(
    name = "base_envoy",
    registry = "quay.io",
    repository = "datawire/ambassador-base",
    tag = "envoy-{relver}.{commit}.{compilation_mode}".format(
        commit = 'e24f8ed76240c049068160fa1ad3efdb324e00e4',
        compilation_mode = 'opt',
        relver = '2',
    ),
)

load("@io_bazel_rules_docker//container:container.bzl", "container_load")
load("@io_bazel_rules_docker//contrib:dockerfile_build.bzl", "dockerfile_image")
dockerfile_image(
    name = "dockerfile_alpine_glibc_with_packages",
    dockerfile = "//:Dockerfile.base",
)
container_load(
    name = "alpine_glibc_with_packages",
    file = "@dockerfile_alpine_glibc_with_packages//image:dockerfile_image.tar",
)

load("@io_bazel_rules_docker//repositories:deps.bzl", docker_deps = "deps")
docker_deps()

load("@io_bazel_rules_docker//python:image.bzl", docker_python_repos = "repositories")
docker_python_repos()

load("@io_bazel_rules_docker//python3:image.bzl", docker_python3_repos = "repositories")
docker_python3_repos()

load("@io_bazel_rules_docker//python3:image.bzl", docker_python3_repos = "repositories")
docker_python3_repos()

# Python #######################################################################

git_repository(
    name = "rules_python",
    remote = "https://github.com/bazelbuild/rules_python.git",
    commit = "a0fbf98d4e3a232144df4d0d80b577c7a693b570",
)

load("@rules_python//python:repositories.bzl", "py_repositories")
py_repositories()

# Python: PIP ##################################################################

git_repository(
    name = "rules_python_external",
    remote = "https://github.com/dillon-giacoppo/rules_python_external.git",
    commit = "9c03622c102659a27c8538040678ae86ba3be0d2",
)

load("@rules_python_external//:repositories.bzl", "rules_python_external_dependencies")
rules_python_external_dependencies()

load("@rules_python_external//:defs.bzl", "pip_install")
pip_install(
    name = "ambassador_pip",
    requirements = "//python:requirements.txt",
)
