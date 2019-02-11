# protoc-gen-star (PGS) [![Build Status](https://travis-ci.org/lyft/protoc-gen-star.svg?branch=master)](https://travis-ci.org/lyft/protoc-gen-star)

**!!! THIS PROJECT IS A WORK-IN-PROGRESS | THE API SHOULD BE CONSIDERED UNSTABLE !!!**

_PGS is a protoc plugin library for efficient proto-based code generation_

```go
package main

import "github.com/lyft/protoc-gen-star"

func main() {
	pgs.Init(pgs.IncludeGo()).
		RegisterPlugin(&myProtocGenGoPlugin{}).
		RegisterModule(&myPGSModule{}).
		RegisterPostProcessor(&myPostProcessor{}).
		Render()
}
```

`protoc-gen-star` (PGS) is built on top of the official [`protoc-gen-go`][pgg] (PGG) protoc plugin. PGG contains a mechanism for extending its behavior with plugins (for instance, gRPC support via a plugin). However, this feature is not accessible from the outside and requires either forking PGG or replicating its behavior using its library code. Further still, the PGG plugins are designed specifically for extending the officially generated Go code, not creating other new files or packages.

PGS leverages the existing PGG library code to properly build up the [Protocol Buffer][pb] (PB) descriptors' dependency graph before handing it off to custom [`Modules`][module] to generate anything above-and-beyond the officially generated code. In fact, by default PGS does not generate the official Go code. While PGS is written in Go and relies on PGG, this library can be used to generate code in any language.

## Features

### Documentation

While this README seeks to describe many of the nuances of `protoc` plugin development and using PGS, the true documentation source is the code itself. The Go language is self-documenting and provides tools for easily reading through it and viewing examples. Until this package is open sourced, the documentation can be viewed locally by running `make docs`, which will start a `godoc` server and open the documentation in the default browser.

### Roadmap

- [x] Full support for official Go PB output and `protoc-gen-go` plugins, can replace `protoc-gen-go`
- [x] Interface-based and fully-linked dependency graph with access to raw descriptors
- [x] Built-in context-aware debugging capabilities 
- [x] Exhaustive, near 100% unit test coverage
- [x] End-to-end testable via overrideable IO
- [x] [`Visitor`][visitor] pattern and helpers for efficiently walking the dependency graph
- [x] [`BuildContext`][context] to facilitate complex generation
- [x] Parsed, typed command-line [`Parameters`][params] access
- [x] Extensible `PluginBase` for quickly creating `protoc-gen-go` plugins
- [x] Extensible `ModuleBase` for quickly creating `Modules` and facilitating code generation
- [x] Configurable post-processing (eg, gofmt/goimports) of generated files
- [x] Support processing proto files from multiple packages (normally disallowed by `protoc-gen-go`)
- [ ] Load plugins/modules at runtime using Go shared libraries
- [x] Load comments from proto files into gathered AST for easy access
- [ ] More intelligent Go import path resolution

### Examples

[`protoc-gen-example`][pge], can be found in the `testdata` directory. It includes both `Plugin` and `Module` implementations using a variety of the features available. It's `protoc` execution is included in the `demo` [Makefile][make] target. Test examples are also accessible via the documentation by running `make docs`.

## How It Works

### The `protoc` Flow

Because the process is somewhat confusing, this section will cover the entire flow of how proto files are converted to generated code, using a hypothetical PGS plugin: `protoc-gen-myplugin`. A typical execution looks like this:

```sh
protoc \
	-I . \
	--myplugin_out="plugins=grpc:../generated" \
	./pkg/*.proto
```

`protoc`, the PB compiler, is configured using a set of flags (documented under `protoc -h`) and handed a set of files as arguments. In this case, the `I` flag can be specified multiple times and is the lookup path it should use for imported dependencies in a proto file. By default, the official descriptor protos are already included.

`myplugin_out` tells `protoc` to use the `protoc-gen-myplugin` protoc-plugin. These plugins are automatically resolved from the system's `PATH` environment variable, or can be explicitly specified with another flag. The official protoc-plugins (eg, `protoc-gen-python`) are already registered with `protoc`. The flag's value is specific to the particular plugin, with the exception of the `:../generated` suffix. This suffix indicates the root directory in which `protoc` will place the generated files from that package (relative to the current working directory). This generated output directory is _not_ propagated to `protoc-gen-myplugin`, however, so it needs to be duplicated in the left-hand side of the flag. PGS supports this via an `output_path` parameter.

`protoc` parses the passed in proto files, ensures they are syntactically correct, and loads any imported dependencies. It converts these files and the dependencies into descriptors (which are themselves PB messages) and creates a `CodeGeneratorRequest` (yet another PB). `protoc` serializes this request and then executes each configured protoc-plugin, sending the payload via `stdin`.

`protoc-gen-myplugin` starts up, receiving the request payload, which it unmarshals. There are two phases to a PGS-based protoc-plugin. First, the standard PGG process is executed against the input. This allows for generation of the official Go code if desired, as well as applying any PGG plugins we've specified in its protoc flag (in this case, we opted to use `grpc`). PGS, also injects a plugin here called the `gatherer`, which constructs a dependency graph from the incoming descriptors. This process populates a `CodeGeneratorResponse` PB message containing the files to generate.

When this step is complete, PGS then executes any registered `Modules`, handing it the constructed graph. `Modules` can be written to generate more files, adding them to the response PB, writing them to disk directly, or just performing some form of validation over the provided graph without any other side effects. `Modules` provide the most flexibility in terms of operating against the PBs.

Once all `Modules` are complete, `protoc-gen-myplugin` serializes the `CodeGeneratorResponse` and writes the data to its `stdout`. `protoc` receives this payload, unmarshals it, and writes any requested files to disk after all its plugins have returned. This whole flow looked something like this:

```
foo.proto → protoc → CodeGeneratorRequest → protoc-gen-myplugin → CodeGeneratorResponse → protoc → foo.pb.go
```

The PGS library hides away nearly all of this complexity required to implement a protoc-plugin!

### Plugins

Plugins in this context refer to libraries registered with the PGG library to extend the functionality of the offiically generated Go code. The only officially supported extension is `grpc`, which generates the server and client code for services defined in the proto files. This is one of the best ways to extend the behavior of the already generated code within its package.

PGS provides a `PluginBase` struct to simplify development of these plugins. Out of the box, it satisfies the interface for a `generator.Plugin`, only requiring the creation of the `Name` and `Generate` methods. `PluginBase` is best used as an anonymous embedded field of a wrapping `Plugin` implementation. A minimal plugin would look like the following:

```go
// GraffitiPlugin tags the generated Go source
type graffitiPlugin struct {
	*pgs.PluginBase
	tag string
}

// New configures the plugin with an instance of PluginBase and captures the tag
// which will be used during code generation.
func New(tag string) pgs.Plugin { 
	return &graffitiPlugin{
		PluginBase: new(pgs.PluginBase),
		tag:        tag,
	}
}

// Name is the identifier used in the protoc execution to enable this plugin for 
// code generation.
func (p *graffitiPlugin) Name() string { return "graffiti" }

// Generate is handed each file descriptor loaded by protoc, including 
// dependencies not targeted for building. Don't worry though, the underlying 
// library ensures that writes only occur for those specified by protoc.
func (p *graffitiPlugin) Generate(f *generator.FileDescriptor) {
	p.Push(f.GetName()).Debug("tagging")
	p.C(80, p.tag)
	p.Pop()
}
```

`PluginBase` exposes a PGS [`BuildContext`][context] instance, already prefixed with the plugin's name. Calling `Push` and `Pop` allows adding further information to error and debugging messages. Above, the name of the file being generated is pushed onto the context before logging the "tagging" debug message. 

The base also provides helper methods for rendering into the output file. `p.P` prints a line with arbitrary arguments, similar to `fmt.Println`. `p.C` renders a comment in a similar fashion to `p.P` but intelligently wraps the comment to multiple lines at the specified width (above, 80 is used to wrap the supplied tag value). While `p.P` and `p.C` are very procedural, sometimes smarter generation is required: `p.T` renders Go templates (of either the `text` or `html` variety).

Typically, plugins are registered globally, usually within an `init` method on the plugin's package, but PGS provides some utilities to facilitate development. When registering it with a PGS `Generator`, however, the `init` methodology should be avoided in favor of the following:

```go
g := pgs.Init(pgs.IncludeGo())
g.RegisterPlugin(graffiti.New("rodaine was here"))
```

`IncludeGo` must be specified or none of the official Go code will be generated. If the plugin also implements the PGS `Plugin` interface (which is achieved for free by composing over `PluginBase`), a shared pre-configured BuildContext will be provided to the plugin for consistent logging and error handling mechanisms.

### Modules

While plugins allow for injecting into the PGG generated code file-by-file, some code generation tasks require knowing the entire dependency graph of a proto file first or intend to create files on disk outside of the output directory specified by `protoc` (or with custom permissions). `Modules` fill this gap.

PGS `Modules` are evaluated after the normal PGG flow and are handed a complete graph of the PB entities from the `gatherer` that are targeted for generation as well as all dependencies. A `Module` can then add files to the protoc `CodeGeneratorResponse` or write files directly to disk as `Artifacts`.

PGS provides a `ModuleBase` struct to simplify developing modules. Out of the box, it satisfies the interface for a `Module`, only requiring the creation of `Name` and `Execute` methods. `ModuleBase` is best used as an anonyomous embedded field of a wrapping `Module` implementation. A minimal module would look like the following:

```go
// ReportModule creates a report of all the target messages generated by the 
// protoc run, writing the file into the /tmp directory.
type reportModule struct {
	*pgs.ModuleBase
}

// New configures the module with an instance of ModuleBase
func New() pgs.Module { return &reportModule{&pgs.ModuleBase{}} }

// Name is the identifier used to identify the module. This value is 
// automatically attached to the BuildContext associated with the ModuleBase.
func (m *reportModule) Name() string { return "reporter" }

// Execute is passed the target pkg as well as its dependencies in the pkgs map.
// The implementation should return a slice of Artifacts that represent the 
// files to be generated. In this case, "/tmp/report.txt" will be created 
// outside of the normal protoc flow.
func (m *reportModule) Execute(pkg pgs.Package, pkgs map[string]pgs.Package) []pgs.Artifact {
	buf := &bytes.Buffer{}

	for _, f := range pkg.Files() {
		m.Push(f.Name().String()).Debug("reporting")

		fmt.Fprintf(buf, "--- %v ---", f.Name())
	
		for i, msg := range f.AllMessages() {
			fmt.Fprintf(buf, "%03d. %v", msg.Name())
		}

		m.Pop()
	}

	m.OverwriteCustomFile(
		"/tmp/report.txt",
		buf.String(),
		0644,
	)

	return m.Artifacts()
}
```

`ModuleBase` exposes a PGS [`BuildContext`][context] instance, already prefixed with the module's name. Calling `Push` and `Pop` allows adding further information to error and debugging messages. Above, each file from the target package is pushed onto the context before logging the "reporting" debug message.

The base also provides helper methods for adding or overwriting both protoc-generated and custom files. The above execute method creates a custom file at `/tmp/report.txt` specifying that it should overwrite an existing file with that name. If it instead called `AddCustomFile` and the file existed, no file would have been generated (though a debug message would be logged out). Similar methods exist for adding generator files, appends, and injections. Likewise, methods such as `AddCustomTemplateFile` allows for `Templates` to be rendered instead.

After all modules have been executed, the returned `Artifacts` are either placed into the `CodeGenerationResponse` payload for protoc or written out to the file system. For testing purposes, the file system has been abstracted such that a custom one (such as an in-memory FS) can be provided to the PGS generator with the `FileSystem` `InitOption`.

Modules are registered with PGS similar to `Plugins`:

 ```go
g := pgs.Init(pgs.IncludeGo())
g.RegisterModule(reporter.New())
```

#### Multi-Package Aware Modules

If the `MultiPackage` `InitOption` is enabled and multiple packages are passed into the PGS plugin, a `Module` can be upgraded to a `MultiModule` interface to support handling more than one target package simultaneously. Implementing this on the `reportModule` above might look like the following:

```go
// MultiExecute satisfies the MultiModule interface. Instead of calling Execute 
// and generating a file for each target package, the report can be written 
// including all files from all packages in one.
func (m *reportModule) MultiExecute(targets map[string]Package, pkgs map[string]Package) []Artifact {
	buf := &bytes.Buffer{}

	for _, pkg := range targets {
		m.Push(pkg.Name().String())
		for _, f := range pkg.Files() {
			m.Push(f.Name().String()).Debug("reporting")

			fmt.Fprintf(buf, "--- %v ---", f.Name())
		
			for i, msg := range f.AllMessages() {
				fmt.Fprintf(buf, "%03d. %v", msg.Name())
			}

			m.Pop()
		}
		m.Pop()
	}

	m.OverwriteCustomFile(
		"/tmp/report.txt",
		buf.String(),
		0644,
	)

	return m.Artifacts()
}
```

Without `MultiExecute`, the module's `Execute` method would be called for each individual target `Package` processed. In the above example, the report file would be created for each, possibly overwriting each other. If a `Module` implements `MultiExecute`, however, the method recieves all target packages at once and can choose how to process them, in this case, creating a single report file for all.

See the **Multi-Package Workflow** section below for more details.

#### Post Processing

`Artifacts` generated by `Modules` sometimes require some mutations prior to writing to disk or sending in the reponse to protoc. This could range from running `gofmt` against Go source or adding copyright headers to all generated source files. To simplify this task in PGS, a `PostProcessor` can be utilized. A minimal looking `PostProcessor` implementation might look like this:

```go
// New returns a PostProcessor that adds a copyright comment to the top
// of all generated files.
func New(owner string) pgs.PostProcessor { return copyrightPostProcessor{owner} }

type copyrightPostProcessor struct {
	owner string
}

// Match returns true only for Custom and Generated files (including templates).
func (cpp copyrightPostProcessor) Match(a pgs.Artifact) bool {
	switch a := a.(type) {
	case pgs.GeneratorFile, pgs.GeneratorTemplateFile, 
		pgs.CustomFile, pgs.CustomTemplateFile:
			return true
	default:
			return false
	}
}

// Process attaches the copyright header to the top of the input bytes
func (cpp copyrightPostProcessor) Process(in []byte) (out []byte, err error) {
	cmt := fmt.Sprintf("// Copyright © %d %s. All rights reserved\n", 
		time.Now().Year(), 
		cpp.owner)

	return append([]byte(cmt), in...), nil
}
``` 

The `copyrightPostProcessor` struct satisfies the `PostProcessor` interface by implementing the `Match` and `Process` methods. After PGS recieves all `Artifacts`, each is handed in turn to each registered processor's `Match` method. In the above case, we return `true` if the file is a part of the targeted Artifact types. If `true` is returned, `Process` is immediately called with the rendered contents of the file. This method mutates the input, returning the modified value to out or an error if something goes wrong. Above, the notice is prepended to the input.

PostProcessors are registered with PGS similar to `Plugins` and `Modules`:

```go
g := pgs.Init(pgs.IncludeGo())
g.RegisterModule(some.NewModule())
g.RegisterPostProcessor(copyright.New("PGS Authors"))
```

## Protocol Buffer AST

While `protoc` ensures that all the dependencies required to generate a proto file are loaded in as descriptors, it's up to the protoc-plugins to recognize the relationships between them. PGG handles this to some extent, but does not expose it in a easily accessible or testable manner outside of its sub-plugins and standard generation. To get around this, PGS uses the `gatherer` plugin to construct an abstract syntax tree (AST) of all the `Entities` loaded into the plugin. This AST is provided to every `Module` to facilitate code generation.

### Hierarchy

The hierarchy generated by the PGS `gatherer` is fully linked, starting at a top-level `Package` down to each individual `Field` of a `Message`. The AST can be represented with the following digraph:

 <p align=center><img src="/testdata/ast/ast.png"></p>

A `Package` describes a set of `Files` loaded within the same namespace. As would be expected, a `File` represents a single proto file, which contains any number of `Message`, `Enum` or `Service` entities. An `Enum` describes an integer-based enumeration type, containing each individual `EnumValue`. A `Service` describes a set of RPC `Methods`, which in turn refer to their input and output `Messages`. 

A `Message` can contain other nested `Messages` and `Enums` as well as each of its `Fields`. For non-scalar types, a `Field` may also reference its `Message` or `Enum` type. As a mechanism for achieving union types, a `Message` can also contain `OneOf` entities that refer to some of its `Fields`.

### Visitor Pattern

The structure of the AST can be fairly complex and unpredictable. Likewise, `Module's` are typically concerned with only a subset of the entities in the graph. To separate the `Module's` algorithm from understanding and traversing the structure of the AST, PGS implements the `Visitor` pattern to decouple the two. Implementing this interface is straightforward and can greatly simplify code generation.

Two base `Visitor` structs are provided by PGS to simplify developing implementations. First, the `NilVisitor` returns an instance that short-circuits execution for all Entity types. This is useful when certain branches of the AST are not interesting to code generation. For instance, if the `Module` is only concerned with `Services`, it can use a `NilVisitor` as an anonymous field and only implement the desired interface methods:

```go
// ServiceVisitor logs out each Method's name
type serviceVisitor struct {
	pgs.Visitor
	pgs.DebuggerCommon
}

func New(d pgs.DebuggerCommon) pgs.Visitor { 
	return serviceVistor{
		Visitor:        pgs.NilVisitor(),
		DebuggerCommon: d,
	} 
}

// Passthrough Packages, Files, and Services. All other methods can be 
// ignored since Services can only live in Files and Files can only live in a 
// Package.
func (v serviceVisitor) VisitPackage(pgs.Package) (pgs.Visitor, error) { return v, nil }
func (v serviceVisitor) VisitFile(pgs.File) (pgs.Visitor, error)       { return v, nil }
func (v serviceVisitor) VisitService(pgs.Service) (pgs.Visitor, error) { return v, nil }

// VisitMethod logs out ServiceName#MethodName for m.
func (v serviceVisitor) VisitMethod(m pgs.Method) (pgs.Vistitor, error) {
	v.Logf("%v#%v", m.Service().Name(), m.Name())
	return nil, nil
}
```

If access to deeply nested `Nodes` is desired, a `PassthroughVisitor` can be used instead. Unlike `NilVisitor` and as the name suggests, this implementation passes through all nodes instead of short-circuiting on the first unimplemented interface method. Setup of this type as an anonymous field is a bit more complex but avoids implementing each method of the interface explicitly:

```go
type fieldVisitor struct {
	pgs.Visitor
	pgs.DebuggerCommon
}

func New(d pgs.DebuggerCommon) pgs.Visitor {
	v := &fieldVisitor{DebuggerCommon: d}
	v.Visitor = pgs.PassThroughVisitor(v)
	return v
}

func (v *fieldVisitor) VisitField(f pgs.Field) (pgs.Visitor, error) {
	v.Logf("%v.%v", f.Message().Name(), f.Name())
	return nil, nil
}
```

Walking the AST with any `Visitor` is straightforward:

```go
v := visitor.New(d)
err := pgs.Walk(v, pkg)
```

All `Entity` types and `Package` can be passed into `Walk`, allowing for starting a `Visitor` lower than the top-level `Package` if desired.

## Build Context

`Plugins` and `Modules` registered with the PGS `Generator` are initialized with an instance of `BuildContext` that encapsulates contextual  paths, debugging, and parameter information.

### Output Paths

The `BuildContext's` `OutputPath` method returns the output directory that the PGS plugin is targeting. For `Plugins`, this path is initially `.` and is relative to the generation output directory specified in the protoc execution. For `Modules`, this path is also initially `.` but refers to the directory in which `protoc` is executed. This default behavior can be overridden for `Modules` by providing an `output_path` in the flag. 

This value can be used to create file names for `Artifacts`, using `JoinPath(name ...string)` which is essentially an alias for `filepath.Join(ctx.Outpath, name...)`. Manually tracking directories relative to the `OutputPath` can be tedious, especially if the names are dynamic. Instead, a `BuildContext` can manage these, via `PushDir` and `PopDir`. 

```go
ctx.OutputPath()                // foo
ctx.JoinPath("fizz", "buzz.go") // foo/fizz/buzz.go

ctx = ctx.PushDir("bar/baz")
ctx.OutputPath()                // foo/bar/baz
ctx.JoinPath("quux.go")         // foo/bar/baz/quux.go

ctx = ctx.PopDir()
ctx.OutputPath()                // foo
```

Both `PluginBase` and `ModuleBase` wrap these methods to mutate their underlying `BuildContexts`. Those methods should be used instead of the ones on the contained `BuildContext` directly.

### Debugging

The `BuildContext` exposes a `DebuggerCommon` interface which provides utilities for logging, error checking, and assertions. `Log` and the formatted `Logf` print messages to `os.Stderr`, typically prefixed with the `Plugin` or `Module` name. `Debug` and `Debugf` behave the same, but only print if enabled via the `DebugMode` or `DebugEnv` `InitOptions`.

`Fail` and `Failf` immediately stops execution of the protoc-plugin and causes `protoc` to fail generation with the provided message. `CheckErr` and `Assert` also fail with the provided messages if an error is passed in or if an expression evaluates to false, respectively.

Additional contextual prefixes can be provided by calling `Push` and `Pop` on the `BuildContext`. This behavior is similar to `PushDir` and `PopDir` but only impacts log messages. Both `PluginBase` and `ModuleBase` wrap these methods to mutate their underlying `BuildContexts`. Those methods should be used instead of the ones on the contained `BuildContext` directly.

### Parameters

The `BuildContext` also provides access to the pre-processed `Parameters` from the specified protoc flag. PGG allows for certain KV pairs in the parameters body, such as "plugins", "import_path", and "import_prefix" as well as import maps with the "M" prefix. PGS exposes these plus typed access to any other KV pairs passed in. The only PGS-specific key expected is "output_path", which is utilized by the a module's `BuildContext` for its `OutputPath`. 

PGS permits mutating the `Parameters` via the `MutateParams` `InitOption`. By passing in a `ParamMutator` function here, these KV pairs can be modified or verified prior to the PGG workflow begins.

## Execution Workflows

Internally, PGS determines its behavior based off workflows. These are not publicly exposed to the API but can be modified based off `InitOptions` when initializing the `Generator`.

### Standard Workflow

The standard workflow follows the steps described above in **The `protoc` Flow**. This is the out-of-the-box behavior of PGS-based plugins.

### Multi-Package Workflow

Due to [purely philosophical reasons][single], PGG does not support passing in more than one package (ie, directory) of proto files at a time. In most circumstances, this is OK (if a bit annoying), however there are some generation patterns that may require loading in multiple packages/directories of protos simultaneously. By enabling this workflow, a PGS plugin will support running against multiple packages. 

This is achieved by splitting the `CodeGeneratorRequest` into multiple sub-requests, spawning a handful of child processes of the PGS plugin, and executing the PGG workflow against each sub-request independently. The parent process acts like `protoc` in this case, and captures the response of these before merging them together into a single `CodeGeneratorResponse`. `Modules` are not executed in the child processes; instead, the parent process executes them. If a `Module` implements the `MultiModule` interface, the `MultiExecute` method will be called with _all_ target `Packages` simultaneously. Otherwise, the `Execute` method is called separately for each target `Package`.

**CAVEATS:** This workflow significantly changes the behavior from the Standard workflow and should be considered experimental. Also, the `ProtocInput` `InitOption` cannot be specified alongside this workflow. Changing the input will prevent the sub-requests from being properly executed. (A future update may make this possible.) Only enable this option if your plugin necessitates multi-package support. 

To enable this workflow, pass the `MultiPackage` `InitOption` to `Init`.

### Exclude Go Workflow

It is not always desirable for a PGS plugin to also generate the official Go source code coming from the PGG library (eg, when not generating Go code). In fact, by default, these files are not generated by PGS plugins. This is achieved by this workflow which decorates another workflow (typically, Standard or Multi-Package) to remove these files from the set of generated files. 

To disable this workflow, pass the `IncludeGo` `InitOption` to `Init`.

## PGS Development & Make Targets

PGS seeks to provide all the tools necessary to rapidly and ergonomically extend and build on top of the Protocol Buffer IDL. Whether the goal is to modify the official protoc-gen-go output or create entirely new files and packages, this library should offer a user-friendly wrapper around the complexities of the PB descriptors and the protoc-plugin workflow.

### Setup

For developing on PGS, you should install the package within the `GOPATH`. PGS uses [glide][glide] for dependency management.

```sh
go get -u github.com/lyft/protoc-gen-star
cd $GOPATH/github.com/lyft/protoc-gen-star
make install 
```

To upgrade dependencies, please make the necessary modifications in `glide.yaml` and run `glide update`.

### Linting & Static Analysis

To avoid style nits and also to enforce some best practices for Go packages, PGS requires passing `golint`, `go vet`, and `go fmt -s` for all code changes.

```sh
make lint
```

### Testing

PGS strives to have near 100% code coverage by unit tests. Most unit tests are run in parallel to catch potential race conditions. There are three ways of running unit tests, each taking longer than the next but providing more insight into test coverage:

```sh
# run unit tests without race detection or code coverage reporting
make quick 

# run unit tests with race detection and code coverage
make tests 

# run unit tests with race detection and generates a code coverage report, opening in a browser
make cover 
```

### Documentation

As PGS is intended to be an open-source utility, good documentation is important for consumers. Go is a self-documenting language, and provides a built in utility to view locally: `godoc`. The following command starts a godoc server and opens a browser window to this package's documentation. If you see a 404 or unavailable page initially, just refresh.

```sh
make docs
```

#### Demo

PGS comes with a "kitchen sink" example: [`protoc-gen-example`][pge]. This protoc plugin built on top of PGS prints out the target package's AST as a tree to stderr. This provides an end-to-end way of validating each of the nuanced types and nesting in PB descriptors:

```sh
make demo
```

#### CI

PGS uses [TravisCI][travis] to validate all code changes. Please view the [configuration][travis.yml] for what tests are involved in the validation.

[glide]: http://glide.sh
[pgg]: https://github.com/golang/protobuf/tree/master/protoc-gen-go
[pge]: https://github.com/lyft/protoc-gen-star/tree/master/testdata/protoc-gen-example
[travis]: https://travis-ci.com/lyft/protoc-gen-star
[travis.yml]: https://github.com/lyft/protoc-gen-star/tree/master/.travis.yml
[module]: https://github.com/lyft/protoc-gen-star/blob/master/module.go
[pb]: https://developers.google.com/protocol-buffers/
[context]: https://github.com/lyft/protoc-gen-star/tree/master/build_context.go
[visitor]: https://github.com/lyft/protoc-gen-star/tree/master/node.go
[params]: https://github.com/lyft/protoc-gen-star/tree/master/parameters.go
[make]: https://github.com/lyft/protoc-gen-star/blob/master/Makefile
[single]: https://github.com/golang/protobuf/pull/40
