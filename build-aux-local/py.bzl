load("@io_bazel_rules_docker//lang:image.bzl", "app_layer")

def py_image(name, base, binary):
    layers = [".py-library"]

    for index, dep in enumerate(layers):
        base = app_layer(name = "%s.%d" % (name, index), base = base, dep = dep)
        base = app_layer(name = "%s.%d-symlinks" % (name, index), base = base, dep = dep, binary = binary)

    app_layer(
        name = name,
        base = base,
        entrypoint = ["/usr/bin/python"],
        binary = binary,
        # The targets of the symlinks in the symlink layers are relative to the
        # workspace directory under the app directory. Thus, create an empty
        # workspace directory to ensure the symlinks are valid. See
        # https://github.com/bazelbuild/rules_docker/issues/161 for details.
        create_empty_workspace_dir = True,
    )
