load(":go_proto_library.bzl", "go_proto_library")
load("@com_google_protobuf//:protobuf.bzl", "proto_gen", "cc_proto_library")

def pgv_go_proto_library(name, srcs = None, deps = [], **kwargs):
    go_proto_library(name,
                     srcs,
                     deps = ["//validate:go_default_library"] + deps,
                     protoc = "@com_google_protobuf//:protoc",
                     visibility = ["//visibility:public"],
                     validate = 1,
                     **kwargs)

def _CcValidateHdrs(srcs):
    ret = [s[:-len(".proto")] + ".pb.validate.h" for s in srcs]
    return ret

def _CcValidateSrcs(srcs):
    ret = [s[:-len(".proto")] + ".pb.validate.cc" for s in srcs]
    return ret

def pgv_cc_proto_library(
        name,
        srcs=[],
        deps=[],
        external_deps=[],
        cc_libs=[],
        include=None,
        protoc="@com_google_protobuf//:protoc",
        protoc_gen_validate = "@com_lyft_protoc_gen_validate//:protoc-gen-validate",
        internal_bootstrap_hack=False,
        use_grpc_plugin=False,
        default_runtime="@com_google_protobuf//:protobuf",
        **kargs):
  """Bazel rule to create a C++ protobuf validation library from proto source files

  Args:
    name: the name of the pgv_cc_proto_library.
    srcs: the .proto files of the pgv_cc_proto_library.
    deps: a list of PGV dependency labels; must be pgv_cc_proto_library.
    external_deps: a list of dependency labels; must be cc_proto_library.
    include: a string indicating the include path of the .proto files.
    protoc: the label of the protocol compiler to generate the sources.
    protoc_gen_validate: override the default version of protoc_gen_validate.
                   Most users won't need this.
    default_runtime: the implicitly default runtime which will be depended on by
        the generated cc_library target.
    **kargs: other keyword arguments that are passed to cc_library.

  """

  # Generate the C++ protos
  cc_proto_library(
      name=name + "_proto",
      srcs=srcs,
      deps=[d + "_proto" for d in deps] + [
          "@com_lyft_protoc_gen_validate//validate:validate_cc",
      ] + external_deps,
      cc_libs=cc_libs,
      incude=include,
      protoc=protoc,
      internal_bootstrap_hack=internal_bootstrap_hack,
      use_grpc_plugin=use_grpc_plugin,
      default_runtime=default_runtime,
      **kargs)

  includes = []
  if include != None:
    includes = [include]

  gen_hdrs = _CcValidateHdrs(srcs)
  gen_srcs = _CcValidateSrcs(srcs)

  proto_gen(
      name=name + "_validate",
      srcs=srcs,
      # This is a hack to work around the fact that all the deps must have an
      # import_flags field, which is only set on the proto_gen rules, so depend
      # on the cc rule
      deps=[d + "_validate" for d in deps] + [
          "@com_lyft_protoc_gen_validate//validate:validate_cc_genproto"
      ] + [d + "_genproto" for d in external_deps],
      includes=includes,
      protoc=protoc,
      plugin=protoc_gen_validate,
      plugin_options=["lang=cc"],
      outs=gen_hdrs + gen_srcs,
      visibility=["//visibility:public"],
  )

  if default_runtime and not default_runtime in cc_libs:
    cc_libs = cc_libs + [default_runtime]

  native.cc_library(
      name=name,
      hdrs=gen_hdrs,
      srcs=gen_srcs,
      deps=cc_libs + deps + [
          ":" + name + "_proto",
          "@com_lyft_protoc_gen_validate//validate:cc_validate",
      ],
      includes=includes,
      alwayslink=1,
      **kargs)
