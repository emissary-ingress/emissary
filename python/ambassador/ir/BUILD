package(default_visibility = ["//visibility:public"])
load("@my_deps//:requirements.bzl", "requirement")

py_library(
    name = "__init__",
    srcs = ["__init__.py"],
    deps = [
        ":ir",
        ":irresource",
    ],
)

py_library(
    name = "ir",
    srcs = ["ir.py"],
    deps = [
        ":irambassador",
        ":irauth",
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircluster",
        ":irfilter",
        ":irhost",
        ":irlistener",
        ":irlogservice",
        ":irmappingfactory",
        ":irratelimit",
        ":irresource",
        ":irserviceresolver",
        ":irtls",
        ":irtlscontext",
        ":irtracing",
        "//ambassador/config:__init__",
        "//ambassador:VERSION",
        "//ambassador:constants",
        "//ambassador:utils",
    ],
)

py_library(
    name = "iradmin",
    srcs = ["iradmin.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
    ],
)

py_library(
    name = "irambassador",
    srcs = ["irambassador.py"],
    deps = [
        ":irbuffer",
        ":ircors",
        ":irfilter",
        ":irgzip",
        ":irhttpmapping",
        ":irresource",
        ":irretrypolicy",
        ":irtls",
        ":irtlscontext",
        "//ambassador/config:__init__",
        "//ambassador:constants",
    ],
)

py_library(
    name = "irauth",
    srcs = ["irauth.py"],
    deps = [
        ":ircluster",
        ":irfilter",
        ":irretrypolicy",
        "//ambassador/config:__init__",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irbasemapping",
    srcs = ["irbasemapping.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
    ],
)

py_library(
    name = "irbasemappinggroup",
    srcs = ["irbasemappinggroup.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
    ],
)

py_library(
    name = "irbuffer",
    srcs = ["irbuffer.py"],
    deps = [
        ":ircluster",
        ":irfilter",
        "//ambassador/config:__init__",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "ircluster",
    srcs = ["ircluster.py"],
    deps = [
        ":irresource",
        ":irtlscontext",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "ircors",
    srcs = ["ircors.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irfilter",
    srcs = ["irfilter.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
    ],
)

py_library(
    name = "irgzip",
    srcs = ["irgzip.py"],
    deps = [
        ":ircluster",
        ":irfilter",
        "//ambassador/config:__init__",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irhost",
    srcs = ["irhost.py"],
    deps = [
        ":irresource",
        ":irtlscontext",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irhttpmapping",
    srcs = ["irhttpmapping.py"],
    deps = [
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircors",
        ":irhttpmappinggroup",
        ":irretrypolicy",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irhttpmappinggroup",
    srcs = ["irhttpmappinggroup.py"],
    deps = [
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircluster",
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irlistener",
    srcs = ["irlistener.py"],
    deps = [
        ":irresource",
        ":irtlscontext",
        "//ambassador/config:__init__",
    ],
)

py_library(
    name = "irlogservice",
    srcs = ["irlogservice.py"],
    deps = [
        ":ircluster",
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irmappingfactory",
    srcs = ["irmappingfactory.py"],
    deps = [
        ":irbasemapping",
        ":irhttpmapping",
        ":irtcpmapping",
        "//ambassador/config:__init__",
    ],
)

py_library(
    name = "irratelimit",
    srcs = ["irratelimit.py"],
    deps = [
        ":ircluster",
        ":irfilter",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irresource",
    srcs = ["irresource.py"],
    deps = [
        "//ambassador/config:__init__",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irretrypolicy",
    srcs = ["irretrypolicy.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irserviceresolver",
    srcs = ["irserviceresolver.py"],
    deps = [
        ":irresource",
        ":irtlscontext",
        "//ambassador/config:__init__",
        "//ambassador:utils",
        "//multi:__init__",
    ],
)

py_library(
    name = "irtcpmapping",
    srcs = ["irtcpmapping.py"],
    deps = [
        ":irbasemapping",
        ":irbasemappinggroup",
        ":irtcpmappinggroup",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irtcpmappinggroup",
    srcs = ["irtcpmappinggroup.py"],
    deps = [
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircluster",
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irtls",
    srcs = ["irtls.py"],
    deps = [
        ":irresource",
        ":irtlscontext",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irtlscontext",
    srcs = ["irtlscontext.py"],
    deps = [
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irtracing",
    srcs = ["irtracing.py"],
    deps = [
        ":ircluster",
        ":irresource",
        "//ambassador/config:__init__",
        "//ambassador:utils",
    ],
)
