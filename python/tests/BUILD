package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "abstract_tests",
    srcs = ["abstract_tests.py"],
    deps = [
        "//kat:harness",
        "//kat:utils",
        requirement("pytest"),
        requirement("pyyaml"),
    ],
)

py_library(
    name = "t_basics",
    srcs = ["t_basics.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_circuitbreaker",
    srcs = ["t_circuitbreaker.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_consul",
    srcs = ["t_consul.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_cors",
    srcs = ["t_cors.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_envoy_logs",
    srcs = ["t_envoy_logs.py"],
    deps = [
        "//kat:harness",
        "//kat:utils",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_extauth",
    srcs = ["t_extauth.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_grpc",
    srcs = ["t_grpc.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_grpc_bridge",
    srcs = ["t_grpc_bridge.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_grpc_web",
    srcs = ["t_grpc_web.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_gzip",
    srcs = ["t_gzip.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_headerrouting",
    srcs = ["t_headerrouting.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_hosts",
    srcs = ["t_hosts.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_ingress",
    srcs = ["t_ingress.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_listeneridletimeout",
    srcs = ["t_listeneridletimeout.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_loadbalancer",
    srcs = ["t_loadbalancer.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_logservice",
    srcs = ["t_logservice.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_lua_scripts",
    srcs = ["t_lua_scripts.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_mappingtests",
    srcs = ["t_mappingtests.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_optiontests",
    srcs = ["t_optiontests.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_plain",
    srcs = ["t_plain.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_ratelimit",
    srcs = ["t_ratelimit.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_redirect",
    srcs = ["t_redirect.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_retrypolicy",
    srcs = ["t_retrypolicy.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_shadow",
    srcs = ["t_shadow.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_stats",
    srcs = ["t_stats.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_tcpmapping",
    srcs = ["t_tcpmapping.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
    ],
)

py_library(
    name = "t_tls",
    srcs = ["t_tls.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "t_tracing",
    srcs = ["t_tracing.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("pytest"),
    ],
)

py_library(
    name = "test_ambassador",
    srcs = ["test_ambassador.py"],
    deps = [
        "//kat:harness",
        requirement("abstract_tests"),
        requirement("t_basics"),
        requirement("t_consul"),
        requirement("t_cors"),
        requirement("t_envoy_logs"),
        requirement("t_extauth"),
        requirement("t_grpc"),
        requirement("t_grpc_bridge"),
        requirement("t_grpc_web"),
        requirement("t_gzip"),
        requirement("t_headerrouting"),
        requirement("t_hosts"),
        requirement("t_ingress"),
        requirement("t_listeneridletimeout"),
        requirement("t_loadbalancer"),
        requirement("t_logservice"),
        requirement("t_lua_scripts"),
        requirement("t_mappingtests"),
        requirement("t_optiontests"),
        requirement("t_plain"),
        requirement("t_ratelimit"),
        requirement("t_redirect"),
        requirement("t_retrypolicy"),
        requirement("t_tcpmapping"),
        requirement("t_tls"),
        requirement("t_tracing"),
    ],
)

py_binary(
    name = "test_docker",
    srcs = ["test_docker.py"],
    deps = [
        requirement("pexpect"),
        requirement("requests"),
    ],
)

py_binary(
    name = "test_envvar_expansion",
    srcs = ["test_envvar_expansion.py"],
    deps = [
        "//ambassador/ir:irbuffer",
        "//ambassador:__init__",
        "//ambassador:utils",
    ],
)

py_binary(
    name = "test_knative",
    srcs = ["test_knative.py"],
    deps = [
        "//kat:harness",
        requirement("pytest"),
        requirement("retry"),
        requirement("urllib"),
        requirement("utils"),
    ],
)

py_binary(
    name = "test_lookup",
    srcs = ["test_lookup.py"],
    deps = [
        "//ambassador/ir:irbuffer",
        "//ambassador:__init__",
        "//ambassador:utils",
    ],
)

py_binary(
    name = "test_schemas",
    srcs = ["test_schemas.py"],
    deps = [
        requirement("jsonschema"),
    ],
)

py_binary(
    name = "test_scout",
    srcs = ["test_scout.py"],
    deps = [
        requirement("pexpect"),
        requirement("requests"),
    ],
)

py_binary(
    name = "test_watt_scaling",
    srcs = ["test_watt_scaling.py"],
    deps = [
        requirement("urllib"),
        requirement("utils"),
    ],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
    deps = [
        requirement("pyyaml"),
    ],
)
