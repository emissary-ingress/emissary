load("@bazel_skylib//lib:dicts.bzl", "dicts")
load("@io_bazel_rules_docker//container:container.bzl", "container")

Piece = provider(fields = {
    "name": "desc here",
    "xxx": "desc here",
})

def piece_binary(name, binary):
    return Piece(
        name = name,
        xxx = binary,
    )

def piece_pydistribution(name, library, scripts):
    pass

def _dep_list(deps):
    return deps.to_list() if type(deps) == "depset" else deps

def somefn(ctx, f):
    return "/".join(["/app", f.short_path])

def runfilesfn(piece):
    return []

def _my_image_impl(
        ctx,
        name,
        pieces,
):
    file_map = dict()
    for piece in pieces:
        runfiles = _dep_list(runfilesfn(piece))
        file_map.update({
            somefn(ctx, f): f
            for f in runfiles
        })
    
    return container.image.implementation(
        ctx,
        directory = "/",
        file_map = file_map,
    )

my_image = rule(
    attrs = dicts.add(container.image.attrs, {
        "base": attr.label(mandatory = True),
        "pieces": attr.label_list(),
    }),
    executable = True,
    outputs = container.image.outputs,
    toolchains = ["@io_bazel_rules_docker//toolchains/docker:toolchain_type"],
    implementation = _my_image_impl,
)
