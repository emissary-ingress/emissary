package envconfig

// This file is a work-alike of github.com/kelseyhightower/envconfig, but:
//  - Has more more idiomatic "tag options" (comma separated things)
//  - Supports falling back to a default if a provided value is invalid
//  - Distinguishes between warnings and fatal errors
//  - Allows setting different parse-modes ("parser"), without using weird types
//
// That said, it is less externally-pluggable: All extensions happen in <envconfig_types.go>.

import (
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type envTag struct {
	Name    string
	Options map[string]string
}

type envTagOption struct {
	Name      string
	Default   *string
	Validator func(string) error
}

var ErrorNotSet = errors.New("is not set")

func parseTagValue(str string, validOptions []envTagOption) (envTag, error) {
	parts := strings.Split(str, ",")
	ret := envTag{
		Name:    strings.TrimSpace(parts[0]),
		Options: make(map[string]string, len(parts)-1),
	}
	for _, optionStr := range parts[1:] {
		optionStr = strings.TrimSpace(optionStr)
		kv := strings.SplitN(optionStr, "=", 2)
		if len(kv) != 2 {
			return envTag{}, errors.Errorf("env option is not a key=value pair: %q", optionStr)
		}
		k := kv[0]
		v := kv[1]
		kOK := false
		for _, optionSpec := range validOptions {
			if k == optionSpec.Name {
				kOK = true
				break
			}
		}
		if !kOK {
			return envTag{}, errors.Errorf("env option %q: unrecognized", k)
		}
		if _, set := ret.Options[k]; set {
			return envTag{}, errors.Errorf("env option %q: is set multiple times", k)
		}
		ret.Options[k] = v
	}
	for _, optionSpec := range validOptions {
		_, haveVal := ret.Options[optionSpec.Name]
		if !haveVal && optionSpec.Default != nil {
			haveVal = true
			ret.Options[optionSpec.Name] = *optionSpec.Default
		}
		if !haveVal {
			continue
		}
		if err := optionSpec.Validator(ret.Options[optionSpec.Name]); err != nil {
			return envTag{}, errors.Wrapf(err, "env option %q", optionSpec.Name)
		}
	}
	return ret, nil
}

func stringPointer(str string) *string {
	return &str
}

type fieldTypeHandler struct {
	Parsers map[string]func(string) (interface{}, error)
	Setter  func(reflect.Value, interface{})
}

func (h fieldTypeHandler) parserNames() []string {
	ret := make([]string, 0, len(h.Parsers))
	for name := range h.Parsers {
		ret = append(ret, name)
	}
	return ret
}

type StructParser struct {
	structType    reflect.Type
	fieldHandlers []func(structValue reflect.Value) (warn, fatal error)
}

// generateParser takes a struct (not a struct pointer) type with `"env:..."` tags on each of its fields, and returns a
// parser for it.
func GenerateParser(structInfo reflect.Type) (StructParser, error) {
	if structInfo.Kind() != reflect.Struct {
		return StructParser{}, errors.Errorf("structInfo does not describe a struct, it describes a %s", structInfo.Kind())
	}

	ret := StructParser{
		structType:    structInfo,
		fieldHandlers: make([]func(structValue reflect.Value) (warn, fatal error), 0, structInfo.NumField()),
	}

	seen := make(map[string]reflect.Type, structInfo.NumField())
	for i := 0; i < structInfo.NumField(); i++ {
		i := i // capture loop variable
		var fieldInfo reflect.StructField = structInfo.Field(i)

		typeHandler, typeHandlerOK := envConfigTypes[fieldInfo.Type] // envConfigTypes is set in envconfig_types.go
		if !typeHandlerOK {
			return StructParser{}, errors.Errorf("struct field %q: unsupported type %s", fieldInfo.Name, fieldInfo.Type)
		}
		validTagOptions := []envTagOption{
			{
				Name:    "const",
				Default: stringPointer("false"),
				Validator: func(val string) error {
					_, err := strconv.ParseBool(val)
					return err
				},
			},
			{
				Name:    "default",
				Default: nil,
				Validator: func(_ string) error {
					return nil
				},
			},
			{
				Name:    "defaultFrom",
				Default: nil,
				Validator: func(val string) error {
					typ, typOK := seen[val]
					switch {
					case !typOK:
						return errors.Errorf("referenced field %q does not exist (yet?)", val)
					case typ != fieldInfo.Type:
						return errors.Errorf("referenced field %q is of type %s, but we need type %s", val, typ, fieldInfo.Type)
					default:
						return nil
					}
				},
			},
			{
				Name:    "parser",
				Default: nil,
				Validator: func(name string) error {
					if _, ok := typeHandler.Parsers[name]; !ok {
						return errors.Errorf("value %q is not one of %v", name, typeHandler.parserNames())
					}
					return nil
				},
			},
		}

		tag, err := parseTagValue(fieldInfo.Tag.Get("env"), validTagOptions)
		if err != nil {
			return StructParser{}, errors.Wrapf(err, "struct field %q", fieldInfo.Name)
		}
		// validate .Name vs "const"
		tagOptionConst, _ := strconv.ParseBool(tag.Options["const"])
		if (tag.Name == "") != tagOptionConst {
			return StructParser{}, errors.Errorf("struct field %q: does not have an environment variable name (and const=false)", fieldInfo.Name)
		}

		// validate "parser" (existence)
		if _, parserNameOK := tag.Options["parser"]; !parserNameOK {
			return StructParser{}, errors.Errorf("struct field %q: type %s requires a \"parser\" setting (valid parsers are %v)", fieldInfo.Name, fieldInfo.Type, typeHandler.parserNames())
		}

		_, haveDef := tag.Options["default"]
		_, haveDefFrom := tag.Options["defaultFrom"]
		// validate "default" vs "defaultFrom"
		if haveDef && haveDefFrom {
			return StructParser{}, errors.Errorf("struct field %q: has both default and defaultFrom", fieldInfo.Name)
		}
		// validate "default" vs "parser"
		if haveDef {
			parserFn := typeHandler.Parsers[tag.Options["parser"]]
			if _, err := parserFn(tag.Options["default"]); err != nil {
				return StructParser{}, errors.Wrapf(err, "struct field %q: invalid default", fieldInfo.Name)
			}
		}

		ret.fieldHandlers = append(ret.fieldHandlers, generateFieldHandler(i, tag, typeHandler))
		seen[fieldInfo.Name] = fieldInfo.Type
	}

	return ret, nil
}

func generateFieldHandler(i int, tag envTag, typeHandler fieldTypeHandler) func(structValue reflect.Value) (warn, fatal error) {
	return func(structValue reflect.Value) (warn, fatal error) {
		var defValue interface{}
		if defStr, haveDef := tag.Options["default"]; haveDef {
			var err error
			defValue, err = typeHandler.Parsers[tag.Options["parser"]](defStr)
			if err != nil {
				panic(err)
			}
		} else if defFromStr, haveDefFrom := tag.Options["defaultFrom"]; haveDefFrom {
			defValue = structValue.FieldByName(defFromStr).Interface()
		}

		var val interface{}
		var err error
		if tag.Name != "" {
			val, err = typeHandler.Parsers[tag.Options["parser"]](os.Getenv(tag.Name))
			if err != nil {
				if defValue == nil {
					// fatal
					return nil, errors.Wrapf(err, "invalid %s (aborting)", tag.Name)
				} else {
					// fall back to default
					val = nil
					if tag.Name != "" && os.Getenv(tag.Name) != "" {
						// Only print a warning if the env-var isn't ""; pretend like "" is
						// unset.  We don't do a str!="" check above, because some parsers
						// accept an empty string.
						if defStr, haveDefStr := tag.Options["default"]; haveDefStr {
							warn = errors.Wrapf(err, "invalid %s (falling back to default %q)", tag.Name, defStr)
						} else {
							warn = errors.Wrapf(err, "invalid %s (falling back to default)", tag.Name)
						}
					}
				}
			} else if val == nil {
				panic(errors.Errorf("should not happen, check the %q %q parser", tag.Name, tag.Options["parser"]))
			}
		}
		if val == nil {
			if defValue == nil {
				return nil, errors.Wrapf(ErrorNotSet, "invalid %s (aborting)", tag.Name)
			}
			val = defValue
		}
		typeHandler.Setter(structValue.Field(i), val)
		return warn, nil
	}
}

// ParseFromEnv populates structPtr from environment variables, returning warnings and fatal errors.  It panics if
// structPtr is of the wrong type for this parser.
func (p StructParser) ParseFromEnv(structPtr interface{}) (warn, fatal []error) {
	structPtrValue := reflect.ValueOf(structPtr)
	if structPtrValue.Kind() != reflect.Ptr {
		panic(errors.New("structPtr is not a pointer"))
	}
	structValue := structPtrValue.Elem()
	if structValue.Type() != p.structType {
		panic(errors.Errorf("wrong type (%s) for parser (%s)", structValue.Elem().Type(), p.structType))
	}

	for _, fieldHandler := range p.fieldHandlers {
		_warn, _fatal := fieldHandler(structValue)
		if _warn != nil {
			warn = append(warn, _warn)
		}
		if _fatal != nil {
			fatal = append(fatal, _fatal)
		}
	}

	return warn, fatal
}
