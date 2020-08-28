// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/config/core/v3/extension.proto

package envoy_config_core_v3

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = ptypes.DynamicAny{}
)

// define the regex for a UUID once up-front
var _extension_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on TypedExtensionConfig with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned.
func (m *TypedExtensionConfig) Validate() error {
	if m == nil {
		return nil
	}

	if utf8.RuneCountInString(m.GetName()) < 1 {
		return TypedExtensionConfigValidationError{
			field:  "Name",
			reason: "value length must be at least 1 runes",
		}
	}

	if m.GetTypedConfig() == nil {
		return TypedExtensionConfigValidationError{
			field:  "TypedConfig",
			reason: "value is required",
		}
	}

	if a := m.GetTypedConfig(); a != nil {

	}

	return nil
}

// TypedExtensionConfigValidationError is the validation error returned by
// TypedExtensionConfig.Validate if the designated constraints aren't met.
type TypedExtensionConfigValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e TypedExtensionConfigValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e TypedExtensionConfigValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e TypedExtensionConfigValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e TypedExtensionConfigValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e TypedExtensionConfigValidationError) ErrorName() string {
	return "TypedExtensionConfigValidationError"
}

// Error satisfies the builtin error interface
func (e TypedExtensionConfigValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sTypedExtensionConfig.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = TypedExtensionConfigValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = TypedExtensionConfigValidationError{}
