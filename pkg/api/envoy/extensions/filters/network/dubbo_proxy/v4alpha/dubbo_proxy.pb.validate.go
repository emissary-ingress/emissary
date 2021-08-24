// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/extensions/filters/network/dubbo_proxy/v4alpha/dubbo_proxy.proto

package envoy_extensions_filters_network_dubbo_proxy_v4alpha

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
var _dubbo_proxy_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on DubboProxy with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *DubboProxy) Validate() error {
	if m == nil {
		return nil
	}

	if len(m.GetStatPrefix()) < 1 {
		return DubboProxyValidationError{
			field:  "StatPrefix",
			reason: "value length must be at least 1 bytes",
		}
	}

	if _, ok := ProtocolType_name[int32(m.GetProtocolType())]; !ok {
		return DubboProxyValidationError{
			field:  "ProtocolType",
			reason: "value must be one of the defined enum values",
		}
	}

	if _, ok := SerializationType_name[int32(m.GetSerializationType())]; !ok {
		return DubboProxyValidationError{
			field:  "SerializationType",
			reason: "value must be one of the defined enum values",
		}
	}

	for idx, item := range m.GetRouteConfig() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return DubboProxyValidationError{
					field:  fmt.Sprintf("RouteConfig[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	for idx, item := range m.GetDubboFilters() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return DubboProxyValidationError{
					field:  fmt.Sprintf("DubboFilters[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	return nil
}

// DubboProxyValidationError is the validation error returned by
// DubboProxy.Validate if the designated constraints aren't met.
type DubboProxyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DubboProxyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DubboProxyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DubboProxyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DubboProxyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DubboProxyValidationError) ErrorName() string { return "DubboProxyValidationError" }

// Error satisfies the builtin error interface
func (e DubboProxyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDubboProxy.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DubboProxyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DubboProxyValidationError{}

// Validate checks the field values on DubboFilter with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *DubboFilter) Validate() error {
	if m == nil {
		return nil
	}

	if len(m.GetName()) < 1 {
		return DubboFilterValidationError{
			field:  "Name",
			reason: "value length must be at least 1 bytes",
		}
	}

	if v, ok := interface{}(m.GetConfig()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return DubboFilterValidationError{
				field:  "Config",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	return nil
}

// DubboFilterValidationError is the validation error returned by
// DubboFilter.Validate if the designated constraints aren't met.
type DubboFilterValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e DubboFilterValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e DubboFilterValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e DubboFilterValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e DubboFilterValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e DubboFilterValidationError) ErrorName() string { return "DubboFilterValidationError" }

// Error satisfies the builtin error interface
func (e DubboFilterValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sDubboFilter.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = DubboFilterValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = DubboFilterValidationError{}
