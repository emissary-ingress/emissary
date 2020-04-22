package(default_visibility = ["//visibility:public"])
load("@ambassador_pip//:requirements.bzl", "requirement")

py_library(
    name = "abstract_tests",
    srcs = ["abstract_tests.py"],
    deps = [
        "//python/kat:harness",
        "//python/kat:utils",
        requirement("pytest"),
        requirement("pyyaml"),
    ],
)

py_library(
    name = "t_basics",
    srcs = ["t_basics.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_circuitbreaker",
    srcs = ["t_circuitbreaker.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_consul",
    srcs = ["t_consul.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_cors",
    srcs = ["t_cors.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_envoy_logs",
    srcs = ["t_envoy_logs.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        "//python/kat:utils",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_extauth",
    srcs = ["t_extauth.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_grpc",
    srcs = ["t_grpc.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_grpc_bridge",
    srcs = ["t_grpc_bridge.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_grpc_web",
    srcs = ["t_grpc_web.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_gzip",
    srcs = ["t_gzip.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_headerrouting",
    srcs = ["t_headerrouting.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_hosts",
    srcs = ["t_hosts.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_ingress",
    srcs = ["t_ingress.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_listeneridletimeout",
    srcs = ["t_listeneridletimeout.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_loadbalancer",
    srcs = ["t_loadbalancer.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_logservice",
    srcs = ["t_logservice.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_lua_scripts",
    srcs = ["t_lua_scripts.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_mappingtests",
    srcs = ["t_mappingtests.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_optiontests",
    srcs = ["t_optiontests.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_plain",
    srcs = ["t_plain.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_ratelimit",
    srcs = ["t_ratelimit.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_redirect",
    srcs = ["t_redirect.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_retrypolicy",
    srcs = ["t_retrypolicy.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_shadow",
    srcs = ["t_shadow.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_stats",
    srcs = ["t_stats.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_tcpmapping",
    srcs = ["t_tcpmapping.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
    ],
)

py_library(
    name = "t_tls",
    srcs = ["t_tls.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_library(
    name = "t_tracing",
    srcs = ["t_tracing.py"],
    deps = [
        ":abstract_tests",
        "//python/kat:harness",
        requirement("pytest"),
    ],
)

py_test(
    name = "test_ambassador",
    srcs = ["test_ambassador.py"],
    deps = [
        ":abstract_tests",
        ":t_basics",
        ":t_consul",
        ":t_cors",
        ":t_envoy_logs",
        ":t_extauth",
        ":t_grpc",
        ":t_grpc_bridge",
        ":t_grpc_web",
        ":t_gzip",
        ":t_headerrouting",
        ":t_hosts",
        ":t_ingress",
        ":t_listeneridletimeout",
        ":t_loadbalancer",
        ":t_logservice",
        ":t_lua_scripts",
        ":t_mappingtests",
        ":t_optiontests",
        ":t_plain",
        ":t_ratelimit",
        ":t_redirect",
        ":t_retrypolicy",
        ":t_tcpmapping",
        ":t_tls",
        ":t_tracing",
        "//python/kat:harness",
    ],
)

py_test(
    name = "test_docker",
    srcs = ["test_docker.py"],
    deps = [
        requirement("pexpect"),
        requirement("requests"),
    ],
)

py_test(
    name = "test_envvar_expansion",
    srcs = ["test_envvar_expansion.py"],
    deps = [
        "//python/ambassador/config:__init__",
        "//python/ambassador/ir:__init__",
        "//python/ambassador/ir:irbuffer",
        "//python/ambassador:__init__",
        "//python/ambassador:utils",
    ],
)

py_test(
    name = "test_knative",
    srcs = ["test_knative.py"],
    deps = [
        ":utils",
        "//python/kat:harness",
        requirement("pytest"),
        requirement("retry"),
        requirement("urllib"),
    ],
)

py_test(
    name = "test_lookup",
    srcs = ["test_lookup.py"],
    deps = [
        "//python/ambassador/config:__init__",
        "//python/ambassador/ir:__init__",
        "//python/ambassador/ir:irbuffer",
        "//python/ambassador:__init__",
        "//python/ambassador:utils",
    ],
)

py_test(
    name = "test_schemas",
    srcs = ["test_schemas.py"],
    deps = [
        requirement("jsonschema"),
    ],
)

py_test(
    name = "test_scout",
    srcs = ["test_scout.py"],
    deps = [
        requirement("pexpect"),
        requirement("requests"),
    ],
)

py_test(
    name = "test_watt_scaling",
    srcs = ["test_watt_scaling.py"],
    deps = [
        ":utils",
        requirement("urllib"),
    ],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
    deps = [
        requirement("pyyaml"),
    ],
)
