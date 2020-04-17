## Documentation generation from protobufs

### Desired state

We want the documentation to be generated from a single source of truth.
This serves many benefits, primary one being we do not have to update documentation in multiple places when a single
change is made.
This source of truth is expected to be the protobuf definitions of our CRDs. All of our CRDs will be defined as
protobuf and will be consumed as is without any unecessary conversion to other formats (JSON Schema, Go code, etc).
The consumption in ambassador is mostly to validate incoming user created resources.

We want the documentation generation to be automated. Whenever protobuf definitions are modified (or created), the docs
should be automatically generated from the comments and hosted on the website.

### Current state

The documentation generation happens via a protoc plugin: https://github.com/pseudomuto/protoc-gen-doc. This plugin
takes .proto files as input and generates documentation from the comments in that file. Visit the plugin's documentation
on steps to install as they might change. Today, it can be installed by running the command:
```
go get -u github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc
```
Once the plugin is installed, run the command:
```
make generate-crd-docs
```
Today, only one CRD (Host) is defined in a .proto file [here](Host.proto). Other CRDs are defined as JSON Schema
[here](../../../python/schemas/). All of these JSON Schemas need to be moved to protobuf.

Right now, the generation only happens for the [Host.proto](Host.proto) file. As and when files are added to this
directory, it's trivial to modify the command in [generate.mk](../../../build-aux-local/generate.mk) to generate
documentation from all the .proto files in this directory.

The documentation is generated based on Markdown template [here](../../../docs/reference/markdown.tmpl). This Markdown
template generates decently formatted documentation right now, but it needs heavy work and vetting from project managers
to get it "right"!

There is a known bug in the generated documentation: the intra-page links in the resulting webpage which allow moving
around are slightly broken, and by slightly broken I mean they do not work at all. They are slightly broken because in
the attribute list towards the top of the page, the links are in camel case while the actual ID of the elements is in
lower case. For example, the list is going to point to `#ACMEProviderSpec` while the ID for this element is
`#acmeproviderspec`.

The docs are generated and saved at [/docs/reference/](../../../docs/reference/) from where the website should pick them
up and deploy under https://getambassador.io/docs/latest/reference/.

Now all of this is about manual generation of docs. For automatic generation:
- When a user makes any change to the .proto files in this directory and sends a PR, the CI should kick in and generate
the CRD files for the website preview. This way the generated docs can be previewed and reviewed and then merged.
[/.ci/pr-build-website-preview](../../../.ci/pr-build-website-preview) has been modified to allow the same, but it fails
because the docs plugin is not yet installed in this repository. This needs to be fixed.
- When a PR is merged, the resulting generated documentation for the CRDs should be merged by CI to the branch where the
PR is merged. This has not been done yet.
