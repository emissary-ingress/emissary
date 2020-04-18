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
        "//ambassador:VERSION",
        "//ambassador/config:__init__",
        "//ambassador:constants",
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
        "//ambassador:utils",
    ],
)

py_library(
    name = "iradmin",
    srcs = ["iradmin.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
    ],
)

py_library(
    name = "irambassador",
    srcs = ["irambassador.py"],
    deps = [
        "//ambassador/config:__init__",
        "//ambassador:constants",
        ":irbuffer",
        ":ircors",
        ":irfilter",
        ":irgzip",
        ":irhttpmapping",
        ":irresource",
        ":irretrypolicy",
        ":irtls",
        ":irtlscontext",
    ],
)

py_library(
    name = "irauth",
    srcs = ["irauth.py"],
    deps = [
        "//ambassador/config:__init__",
        ":ircluster",
        ":irfilter",
        ":irretrypolicy",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irbasemapping",
    srcs = ["irbasemapping.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
    ],
)

py_library(
    name = "irbasemappinggroup",
    srcs = ["irbasemappinggroup.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
    ],
)

py_library(
    name = "irbuffer",
    srcs = ["irbuffer.py"],
    deps = [
        "//ambassador/config:__init__",
        ":ircluster",
        ":irfilter",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "ircluster",
    srcs = ["ircluster.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
        ":irtlscontext",
        "//ambassador:utils",
    ],
)

py_library(
    name = "ircors",
    srcs = ["ircors.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irfilter",
    srcs = ["irfilter.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
    ],
)

py_library(
    name = "irgzip",
    srcs = ["irgzip.py"],
    deps = [
        "//ambassador/config:__init__",
        ":ircluster",
        ":irfilter",
        "//ambassador:resource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irhost",
    srcs = ["irhost.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
        ":irtlscontext",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irhttpmapping",
    srcs = ["irhttpmapping.py"],
    deps = [
        "//ambassador:utils",
        "//ambassador/config:__init__",
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircors",
        ":irhttpmappinggroup",
        ":irretrypolicy",
    ],
)

py_library(
    name = "irhttpmappinggroup",
    srcs = ["irhttpmappinggroup.py"],
    deps = [
        "//ambassador:utils",
        "//ambassador/config:__init__",
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircluster",
        ":irresource",
    ],
)

py_library(
    name = "irlistener",
    srcs = ["irlistener.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
        ":irtlscontext",
    ],
)

py_library(
    name = "irlogservice",
    srcs = ["irlogservice.py"],
    deps = [
        "//ambassador/config:__init__",
        ":ircluster",
        ":irresource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irmappingfactory",
    srcs = ["irmappingfactory.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irbasemapping",
        ":irhttpmapping",
        ":irtcpmapping",
    ],
)

py_library(
    name = "irratelimit",
    srcs = ["irratelimit.py"],
    deps = [
        "//ambassador/config:__init__",
        ":ircluster",
        ":irfilter",
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
        "//ambassador/config:__init__",
        ":irresource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irserviceresolver",
    srcs = ["irserviceresolver.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
        ":irtlscontext",
        "//multi:__init__",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irtcpmapping",
    srcs = ["irtcpmapping.py"],
    deps = [
        "//ambassador:utils",
        "//ambassador/config:__init__",
        ":irbasemapping",
        ":irbasemappinggroup",
        ":irtcpmappinggroup",
    ],
)

py_library(
    name = "irtcpmappinggroup",
    srcs = ["irtcpmappinggroup.py"],
    deps = [
        "//ambassador:utils",
        "//ambassador/config:__init__",
        ":irbasemapping",
        ":irbasemappinggroup",
        ":ircluster",
        ":irresource",
    ],
)

py_library(
    name = "irtls",
    srcs = ["irtls.py"],
    deps = [
        "//ambassador:utils",
        "//ambassador/config:__init__",
        ":irresource",
        ":irtlscontext",
    ],
)

py_library(
    name = "irtlscontext",
    srcs = ["irtlscontext.py"],
    deps = [
        "//ambassador/config:__init__",
        ":irresource",
        "//ambassador:utils",
    ],
)

py_library(
    name = "irtracing",
    srcs = ["irtracing.py"],
    deps = [
        "//ambassador/config:__init__",
        ":ircluster",
        ":irresource",
        "//ambassador:utils",
    ],
)
