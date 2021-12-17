# envconfig

[![PkgGoDev](https://pkg.go.dev/badge/github.com/datawire/envconfig)](https://pkg.go.dev/github.com/datawire/envconfig)
[![Go Report Card](https://goreportcard.com/badge/github.com/datawire/envconfig)](https://goreportcard.com/report/github.com/datawire/envconfig)
[![Quality Assurance](https://github.com/datawire/envconfig/actions/workflows/qa.yml/badge.svg)](https://github.com/datawire/envconfig/actions)
[![Coverage Status](https://coveralls.io/repos/github/datawire/envconfig/badge.svg)](https://coveralls.io/github/datawire/envconfig)

envconfig populates struct members based on environment variables.

Yes, there are several popular existing solutions:
 - https://github.com/sethvargo/go-envconfig
 - https://github.com/kelseyhightower/envconfig
 - https://github.com/spf13/viper

This one has several attractive properties:
 - It is designed to "meet you where you are".  You probably have an
   existing codebase that does a bunch of ad-hoc env-var parsing with
   weird semantics.  It allows you to clean up your code internally
   without changing the behavior for your users.
 - It supports falling back to a default value not just if an env-var
   is missing, but if the env-var's value is invalid.
 - It allows one member's default value to refer to another.
 - Distinguishes between warnings and fatal errors
 - Allows setting different parse-modes ("parser"), without using
   weird types.  It is easy to add new parsers.
 - Supports nested structs, though it is not possible to add a prefix
   (as https://github.com/sethvargo/go-envconfig allows you to do.
 - Tag options are parsed more idiomatically
   (`"env:comma,separated,list"`) than
   https://github.com/kelseyhightower/envconfig.

# Example

```go
import (
	"time"

	"github.com/datawire/envconfig"
)

type Config struct {
	Port    int           `env:"PORT    ,parser=strconv.ParseInt               "`
	Timeout time.Duration `env:"TIMEOUT ,parser=time.ParseDuration ,default=5s "`
}

func ConfigFromEnv() (cfg Config, warn []error, fatal []error) {
	parser, err := envconfig.GenerateParser(reflect.TypeOf(Config{}), nil)
	if err != nil {
		// panic, because it means that the definition of
		// 'Config' is invalid, which is a bug, not a
		// runtime error.
		panic(err)
	}
	warn, fatal = parser.ParseFromEnv(&cfg)
	return
}
```

# Tag Syntax

As is idiomatic for struct-tag systems, envconfig interprets struct
tags as sequence of whitespace-separated `key:"value"` pairs.  It
looks exclusively at the `env` key, and interprets the value as
`NAME[,CFG1[,CFG2]]` where `NAME` is the name of the environment
variable to look at, and the `CFG$N` configuration flags are as listed
below.  Each item in this comma-separated sequence gets
whitespace-trimmed; it is allowable to pad your options with
whitespace for readability.

 - `parser`=parsername

   The `parser=` flag is required.  It tells envconfig how to parse
   the env-var and the default value (if you specify a default).  You
   pass your list of parsers to `envconfig.GenerateParser`, or pass in
   nil to use the list from `envconfig.DefaultFieldTypeHandlers()`.
   See [`envconfig_types.go`](./envconfig_types.go) for how to define
   your own parsers.

 - `const`

   The `const` flag indicates that this value should *not* be read
   from an environment variable, but instead should be the constant
   value specified in the `default=` flag.  If `const` is set, then
   the `NAME` must be empty; conversely, if `const` is not set, then
   the `NAME` must not be empty.

   ```go
   struct {
   	NonConfigurableDir   string   `env:",const=true   ,parser=nonempty-string    ,default=/opt/some-dir  "`
   }
   ```

 - `default`=defaultstring

   The `default=` flag is optional, and specifies a default value for
   this member if the env-var is not set or if it contains a value
   that the `parser=` could not interpret without error.  If the
   `default=` flag is not present, then this struct member is
   considered to be **required**, and `ParseFromEnv` will return an
   error if the env-var is unset or invalid.  The string passed to the
   `default=` flag is interpretted according to the `parser=`.

 - `defaultFrom`=membername

   Similar to `default=`, the `defaultFrom=` flag specifies a default
   value for this member, but it does so by referring to another
   member earlier in the same struct.  The member being referred to
   _must_ be mentioned earlier (forward references do not work).  The
   member being referred to must be it must of the same type as this
   member; the value is copied directly, rather than going through the
   parser.  This allows members to be chained to support multiple ways
   of setting the same thing.

   It is invalid to set both `default=` and `defaultFrom=`.

   The following example, allows a legacy `TIMEOUT_S` variable to be
   set to an integer number of seconds, but that is overridden by a
   newer `TIMEOUT` variable that takes a friendlier duration-string,
   and then that gets copied in to a `Timeout` member for easy access
   in your Go code.

   ```go
   struct {
   	Timeout_LowPrecendence   time.Duration  `env:"TIMEOUT_S  ,parser=integer-seconds     ,default=5                         "`
   	Timeout_HighPrecendence  time.Duration  `env:"TIMEOUT    ,parser=time.ParseDuration  ,defaultFrom=TimeoutLowPrecedence  "`
   	Timeout                  time.Duration  `env:",const     ,parser=time.ParseDuration  ,defaultFrom=TimeoutHighPrecedence "`
   }
   ```
