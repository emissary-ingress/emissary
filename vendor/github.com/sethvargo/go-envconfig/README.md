# Envconfig

[![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](https://pkg.go.dev/mod/github.com/sethvargo/go-envconfig)
[![GitHub Actions](https://img.shields.io/github/workflow/status/sethvargo/go-envconfig/Test?style=flat-square)](https://github.com/sethvargo/go-envconfig/actions?query=workflow%3ATest)

Envconfig populates struct field values based on environment variables or
arbitrary lookup functions. It supports pre-setting mutations, which is useful
for things like converting values to uppercase, trimming whitespace, or looking
up secrets.

**Note:** Versions prior to v0.2 used a different import path. This README and
examples are for v0.2+.

## Usage

Define a struct with fields using the `env` tag:

```go
type MyConfig struct {
  Port     int    `env:"PORT"`
  Username string `env:"USERNAME"`
}
```

Set some environment variables:

```sh
export PORT=5555
export USERNAME=yoyo
```

Process it using envconfig:

```go
package main

import (
  "context"
  "log"

  "github.com/sethvargo/go-envconfig"
)

func main() {
  ctx := context.Background()

  var c MyConfig
  if err := envconfig.Process(ctx, &c); err != nil {
    log.Fatal(err)
  }

  // c.Port = 5555
  // c.Username = "yoyo"
}
```

You can also use nested structs, just remember that any fields you want to
process must be public:

```go
type MyConfig struct {
  Database *DatabaseConfig
}

type DatabaseConfig struct {
  Port     int    `env:"PORT"`
  Username string `env:"USERNAME"`
}
```

## Configuration

Use the `env` struct tag to define configuration.

### Required

If a field is required, processing will error if the environment variable is
unset.

```go
type MyStruct struct {
  Port int `env:"PORT,required"`
}
```

It is invalid to have a field as both `required` and `default`.

### Default

If an environment variable is not set, the field will be set to the default
value. Note that the environment variable must not be set (e.g. `unset PORT`).
If the environment variable is the empty string, that counts as a "value" and
the default will not be used.

```go
type MyStruct struct {
  Port int `env:"PORT,default=5555"`
}
```

You can also set the default value to another field or value from the
environment, for example:

```go
type MyStruct struct {
  DefaultPort int `env:"DEFAULT_PORT,default=5555"`
  Port        int `env:"OVERRIDE_PORT,default=$DEFAULT_PORT"`
}
```

The value for `Port` will default to the value of `DEFAULT_PORT`.

It is invalid to have a field as both `required` and `default`.

### Prefix

For shared, embedded structs, you can define a prefix to use when processing
struct values for that embed.

```go
type SharedConfig struct {
  Port int `env:"PORT,default=5555"`
}

type Server1 struct {
  // This processes Port from $FOO_PORT.
  *SharedConfig `env:",prefix=FOO_"`
}

type Server2 struct {
  // This processes Port from $BAR_PORT.
  *SharedConfig `env:",prefix=BAR_"`
}
```

It is invalid to specify a prefix on non-struct fields.

## Complex Types

### Durations

In the environment, `time.Duration` values are specified as a parsable Go
duration:

```go
type MyStruct struct {
  MyVar time.Duration `env:"MYVAR"`
}
```

```bash
export MYVAR="10m" # 10 * time.Minute
```

### TextUnmarshaler / BinaryUnmarshaler

Types that implement `TextUnmarshaler` or `BinaryUnmarshaler` are processed as such.

### json.Unmarshaler

Types that implement `json.Unmarshaler` are processed as such.

### gob.Decoder

Types that implement `gob.Decoder` are processed as such.


### Slices

Slices are specified as comma-separated values:

```go
type MyStruct struct {
  MyVar []string `env:"MYVAR"`
}
```

```bash
export MYVAR="a,b,c,d" # []string{"a", "b", "c", "d"}
```

Note that byte slices are special cased and interpreted as strings from the
environment.

### Maps

Maps are specified as comma-separated key:value pairs:

```go
type MyStruct struct {
  MyVar map[string]string `env:"MYVAR"`
}
```

```bash
export MYVAR="a:b,c:d" # map[string]string{"a":"b", "c":"d"}
```

### Structs

Envconfig walks the entire struct, so deeply-nested fields are also supported. You can also define your own decoder (see below).


## Prefixing

You can define a custom prefix using the `PrefixLookuper`. This will lookup
values in the environment by prefixing the keys with the provided value:

```go
type MyStruct struct {
  MyVar string `env:"MYVAR"`
}
```

```go
// Process variables, but look for the "APP_" prefix.
l := envconfig.PrefixLookuper("APP_", envconfig.OsLookuper())
if err := envconfig.ProcessWith(ctx, &c, l); err != nil {
  panic(err)
}
```

```bash
export APP_MYVAR="foo"
```


## Extension

All built-in types are supported except Func and Chan. If you need to define a
custom decoder, implement `Decoder`:

```go
type MyStruct struct {
  field string
}

func (v *MyStruct) EnvDecode(val string) error {
  v.field = fmt.Sprintf("PREFIX-%s", val)
  return nil
}
```

If you need to modify environment variable values before processing, you can
specify a custom `Mutator`:

```go
type Config struct {
  Password `env:"PASSWORD"`
}

func resolveSecretFunc(ctx context.Context, key, value string) (string, error) {
  if strings.HasPrefix(key, "secret://") {
    return secretmanager.Resolve(ctx, value) // example
  }
  return value, nil
}

var config Config
envconfig.ProcessWith(ctx, &config, envconfig.OsLookuper(), resolveSecretFunc)
```

## Testing

Relying on the environment in tests can be troublesome because environment
variables are global, which makes it difficult to parallelize the tests.
Envconfig supports extracting data from anything that returns a value:

```go
lookuper := envconfig.MapLookuper(map[string]string{
  "FOO": "bar",
  "ZIP": "zap",
})

var config Config
envconfig.ProcessWith(ctx, &config, lookuper)
```

Now you can parallelize all your tests by providing a map for the lookup
function. In fact, that's how the tests in this repo work, so check there for an
example.

You can also combine multiple lookupers with `MultiLookuper`. See the GoDoc for
more information and examples.


## Inspiration

This library is conceptually similar to [kelseyhightower/envconfig](https://github.com/kelseyhightower/envconfig), with the following
major behavioral differences:

-   Adds support for specifying a custom lookup function (such as a map), which
    is useful for testing.

-   Only populates fields if they contain zero or nil values. This means you can
    pre-initialize a struct and any pre-populated fields will not be overwritten
    during processing.

-   Support for interpolation. The default value for a field can be the value of
    another field.

-   Support for arbitrary mutators that change/resolve data before type
    conversion.
