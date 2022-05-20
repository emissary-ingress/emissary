// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/config/filter/network/dubbo_proxy/v2alpha1/route.proto

package v2alpha1

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
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
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on RouteConfiguration with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *RouteConfiguration) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RouteConfiguration with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// RouteConfigurationMultiError, or nil if none found.
func (m *RouteConfiguration) ValidateAll() error {
	return m.validate(true)
}

func (m *RouteConfiguration) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Name

	// no validation rules for Interface

	// no validation rules for Group

	// no validation rules for Version

	for idx, item := range m.GetRoutes() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, RouteConfigurationValidationError{
						field:  fmt.Sprintf("Routes[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, RouteConfigurationValidationError{
						field:  fmt.Sprintf("Routes[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return RouteConfigurationValidationError{
					field:  fmt.Sprintf("Routes[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return RouteConfigurationMultiError(errors)
	}

	return nil
}

// RouteConfigurationMultiError is an error wrapping multiple validation errors
// returned by RouteConfiguration.ValidateAll() if the designated constraints
// aren't met.
type RouteConfigurationMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RouteConfigurationMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RouteConfigurationMultiError) AllErrors() []error { return m }

// RouteConfigurationValidationError is the validation error returned by
// RouteConfiguration.Validate if the designated constraints aren't met.
type RouteConfigurationValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RouteConfigurationValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RouteConfigurationValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RouteConfigurationValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RouteConfigurationValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RouteConfigurationValidationError) ErrorName() string {
	return "RouteConfigurationValidationError"
}

// Error satisfies the builtin error interface
func (e RouteConfigurationValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRouteConfiguration.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RouteConfigurationValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RouteConfigurationValidationError{}

// Validate checks the field values on Route with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Route) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Route with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in RouteMultiError, or nil if none found.
func (m *Route) ValidateAll() error {
	return m.validate(true)
}

func (m *Route) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if m.GetMatch() == nil {
		err := RouteValidationError{
			field:  "Match",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetMatch()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, RouteValidationError{
					field:  "Match",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, RouteValidationError{
					field:  "Match",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMatch()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return RouteValidationError{
				field:  "Match",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if m.GetRoute() == nil {
		err := RouteValidationError{
			field:  "Route",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	if all {
		switch v := interface{}(m.GetRoute()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, RouteValidationError{
					field:  "Route",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, RouteValidationError{
					field:  "Route",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetRoute()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return RouteValidationError{
				field:  "Route",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return RouteMultiError(errors)
	}

	return nil
}

// RouteMultiError is an error wrapping multiple validation errors returned by
// Route.ValidateAll() if the designated constraints aren't met.
type RouteMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RouteMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RouteMultiError) AllErrors() []error { return m }

// RouteValidationError is the validation error returned by Route.Validate if
// the designated constraints aren't met.
type RouteValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RouteValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RouteValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RouteValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RouteValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RouteValidationError) ErrorName() string { return "RouteValidationError" }

// Error satisfies the builtin error interface
func (e RouteValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRoute.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RouteValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RouteValidationError{}

// Validate checks the field values on RouteMatch with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *RouteMatch) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RouteMatch with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in RouteMatchMultiError, or
// nil if none found.
func (m *RouteMatch) ValidateAll() error {
	return m.validate(true)
}

func (m *RouteMatch) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetMethod()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, RouteMatchValidationError{
					field:  "Method",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, RouteMatchValidationError{
					field:  "Method",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetMethod()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return RouteMatchValidationError{
				field:  "Method",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	for idx, item := range m.GetHeaders() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, RouteMatchValidationError{
						field:  fmt.Sprintf("Headers[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, RouteMatchValidationError{
						field:  fmt.Sprintf("Headers[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return RouteMatchValidationError{
					field:  fmt.Sprintf("Headers[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return RouteMatchMultiError(errors)
	}

	return nil
}

// RouteMatchMultiError is an error wrapping multiple validation errors
// returned by RouteMatch.ValidateAll() if the designated constraints aren't met.
type RouteMatchMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RouteMatchMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RouteMatchMultiError) AllErrors() []error { return m }

// RouteMatchValidationError is the validation error returned by
// RouteMatch.Validate if the designated constraints aren't met.
type RouteMatchValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RouteMatchValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RouteMatchValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RouteMatchValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RouteMatchValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RouteMatchValidationError) ErrorName() string { return "RouteMatchValidationError" }

// Error satisfies the builtin error interface
func (e RouteMatchValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRouteMatch.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RouteMatchValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RouteMatchValidationError{}

// Validate checks the field values on RouteAction with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *RouteAction) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on RouteAction with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in RouteActionMultiError, or
// nil if none found.
func (m *RouteAction) ValidateAll() error {
	return m.validate(true)
}

func (m *RouteAction) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	switch m.ClusterSpecifier.(type) {

	case *RouteAction_Cluster:
		// no validation rules for Cluster

	case *RouteAction_WeightedClusters:

		if all {
			switch v := interface{}(m.GetWeightedClusters()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, RouteActionValidationError{
						field:  "WeightedClusters",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, RouteActionValidationError{
						field:  "WeightedClusters",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetWeightedClusters()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return RouteActionValidationError{
					field:  "WeightedClusters",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	default:
		err := RouteActionValidationError{
			field:  "ClusterSpecifier",
			reason: "value is required",
		}
		if !all {
			return err
		}
		errors = append(errors, err)

	}

	if len(errors) > 0 {
		return RouteActionMultiError(errors)
	}

	return nil
}

// RouteActionMultiError is an error wrapping multiple validation errors
// returned by RouteAction.ValidateAll() if the designated constraints aren't met.
type RouteActionMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m RouteActionMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m RouteActionMultiError) AllErrors() []error { return m }

// RouteActionValidationError is the validation error returned by
// RouteAction.Validate if the designated constraints aren't met.
type RouteActionValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e RouteActionValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e RouteActionValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e RouteActionValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e RouteActionValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e RouteActionValidationError) ErrorName() string { return "RouteActionValidationError" }

// Error satisfies the builtin error interface
func (e RouteActionValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sRouteAction.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = RouteActionValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = RouteActionValidationError{}

// Validate checks the field values on MethodMatch with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *MethodMatch) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on MethodMatch with the rules defined in
// the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in MethodMatchMultiError, or
// nil if none found.
func (m *MethodMatch) ValidateAll() error {
	return m.validate(true)
}

func (m *MethodMatch) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetName()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, MethodMatchValidationError{
					field:  "Name",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, MethodMatchValidationError{
					field:  "Name",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetName()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return MethodMatchValidationError{
				field:  "Name",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	{
		sorted_keys := make([]uint32, len(m.GetParamsMatch()))
		i := 0
		for key := range m.GetParamsMatch() {
			sorted_keys[i] = key
			i++
		}
		sort.Slice(sorted_keys, func(i, j int) bool { return sorted_keys[i] < sorted_keys[j] })
		for _, key := range sorted_keys {
			val := m.GetParamsMatch()[key]
			_ = val

			// no validation rules for ParamsMatch[key]

			if all {
				switch v := interface{}(val).(type) {
				case interface{ ValidateAll() error }:
					if err := v.ValidateAll(); err != nil {
						errors = append(errors, MethodMatchValidationError{
							field:  fmt.Sprintf("ParamsMatch[%v]", key),
							reason: "embedded message failed validation",
							cause:  err,
						})
					}
				case interface{ Validate() error }:
					if err := v.Validate(); err != nil {
						errors = append(errors, MethodMatchValidationError{
							field:  fmt.Sprintf("ParamsMatch[%v]", key),
							reason: "embedded message failed validation",
							cause:  err,
						})
					}
				}
			} else if v, ok := interface{}(val).(interface{ Validate() error }); ok {
				if err := v.Validate(); err != nil {
					return MethodMatchValidationError{
						field:  fmt.Sprintf("ParamsMatch[%v]", key),
						reason: "embedded message failed validation",
						cause:  err,
					}
				}
			}

		}
	}

	if len(errors) > 0 {
		return MethodMatchMultiError(errors)
	}

	return nil
}

// MethodMatchMultiError is an error wrapping multiple validation errors
// returned by MethodMatch.ValidateAll() if the designated constraints aren't met.
type MethodMatchMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m MethodMatchMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m MethodMatchMultiError) AllErrors() []error { return m }

// MethodMatchValidationError is the validation error returned by
// MethodMatch.Validate if the designated constraints aren't met.
type MethodMatchValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e MethodMatchValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e MethodMatchValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e MethodMatchValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e MethodMatchValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e MethodMatchValidationError) ErrorName() string { return "MethodMatchValidationError" }

// Error satisfies the builtin error interface
func (e MethodMatchValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sMethodMatch.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = MethodMatchValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = MethodMatchValidationError{}

// Validate checks the field values on MethodMatch_ParameterMatchSpecifier with
// the rules defined in the proto definition for this message. If any rules
// are violated, the first error encountered is returned, or nil if there are
// no violations.
func (m *MethodMatch_ParameterMatchSpecifier) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on MethodMatch_ParameterMatchSpecifier
// with the rules defined in the proto definition for this message. If any
// rules are violated, the result is a list of violation errors wrapped in
// MethodMatch_ParameterMatchSpecifierMultiError, or nil if none found.
func (m *MethodMatch_ParameterMatchSpecifier) ValidateAll() error {
	return m.validate(true)
}

func (m *MethodMatch_ParameterMatchSpecifier) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	switch m.ParameterMatchSpecifier.(type) {

	case *MethodMatch_ParameterMatchSpecifier_ExactMatch:
		// no validation rules for ExactMatch

	case *MethodMatch_ParameterMatchSpecifier_RangeMatch:

		if all {
			switch v := interface{}(m.GetRangeMatch()).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, MethodMatch_ParameterMatchSpecifierValidationError{
						field:  "RangeMatch",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, MethodMatch_ParameterMatchSpecifierValidationError{
						field:  "RangeMatch",
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(m.GetRangeMatch()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return MethodMatch_ParameterMatchSpecifierValidationError{
					field:  "RangeMatch",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return MethodMatch_ParameterMatchSpecifierMultiError(errors)
	}

	return nil
}

// MethodMatch_ParameterMatchSpecifierMultiError is an error wrapping multiple
// validation errors returned by
// MethodMatch_ParameterMatchSpecifier.ValidateAll() if the designated
// constraints aren't met.
type MethodMatch_ParameterMatchSpecifierMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m MethodMatch_ParameterMatchSpecifierMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m MethodMatch_ParameterMatchSpecifierMultiError) AllErrors() []error { return m }

// MethodMatch_ParameterMatchSpecifierValidationError is the validation error
// returned by MethodMatch_ParameterMatchSpecifier.Validate if the designated
// constraints aren't met.
type MethodMatch_ParameterMatchSpecifierValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e MethodMatch_ParameterMatchSpecifierValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e MethodMatch_ParameterMatchSpecifierValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e MethodMatch_ParameterMatchSpecifierValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e MethodMatch_ParameterMatchSpecifierValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e MethodMatch_ParameterMatchSpecifierValidationError) ErrorName() string {
	return "MethodMatch_ParameterMatchSpecifierValidationError"
}

// Error satisfies the builtin error interface
func (e MethodMatch_ParameterMatchSpecifierValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sMethodMatch_ParameterMatchSpecifier.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = MethodMatch_ParameterMatchSpecifierValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = MethodMatch_ParameterMatchSpecifierValidationError{}
