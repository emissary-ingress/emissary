// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/api/v2/auth/common.proto

package envoy_api_v2_auth

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
var _common_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on TlsParameters with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *TlsParameters) Validate() error {
	if m == nil {
		return nil
	}

	if _, ok := TlsParameters_TlsProtocol_name[int32(m.GetTlsMinimumProtocolVersion())]; !ok {
		return TlsParametersValidationError{
			field:  "TlsMinimumProtocolVersion",
			reason: "value must be one of the defined enum values",
		}
	}

	if _, ok := TlsParameters_TlsProtocol_name[int32(m.GetTlsMaximumProtocolVersion())]; !ok {
		return TlsParametersValidationError{
			field:  "TlsMaximumProtocolVersion",
			reason: "value must be one of the defined enum values",
		}
	}

	return nil
}

// TlsParametersValidationError is the validation error returned by
// TlsParameters.Validate if the designated constraints aren't met.
type TlsParametersValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e TlsParametersValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e TlsParametersValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e TlsParametersValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e TlsParametersValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e TlsParametersValidationError) ErrorName() string { return "TlsParametersValidationError" }

// Error satisfies the builtin error interface
func (e TlsParametersValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sTlsParameters.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = TlsParametersValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = TlsParametersValidationError{}

// Validate checks the field values on PrivateKeyProvider with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned.
func (m *PrivateKeyProvider) Validate() error {
	if m == nil {
		return nil
	}

	if len(m.GetProviderName()) < 1 {
		return PrivateKeyProviderValidationError{
			field:  "ProviderName",
			reason: "value length must be at least 1 bytes",
		}
	}

	switch m.ConfigType.(type) {

	case *PrivateKeyProvider_Config:

		if v, ok := interface{}(m.GetConfig()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return PrivateKeyProviderValidationError{
					field:  "Config",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	case *PrivateKeyProvider_TypedConfig:

		if v, ok := interface{}(m.GetTypedConfig()).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return PrivateKeyProviderValidationError{
					field:  "TypedConfig",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	return nil
}

// PrivateKeyProviderValidationError is the validation error returned by
// PrivateKeyProvider.Validate if the designated constraints aren't met.
type PrivateKeyProviderValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e PrivateKeyProviderValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e PrivateKeyProviderValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e PrivateKeyProviderValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e PrivateKeyProviderValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e PrivateKeyProviderValidationError) ErrorName() string {
	return "PrivateKeyProviderValidationError"
}

// Error satisfies the builtin error interface
func (e PrivateKeyProviderValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sPrivateKeyProvider.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = PrivateKeyProviderValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = PrivateKeyProviderValidationError{}

// Validate checks the field values on TlsCertificate with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *TlsCertificate) Validate() error {
	if m == nil {
		return nil
	}

	if v, ok := interface{}(m.GetCertificateChain()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TlsCertificateValidationError{
				field:  "CertificateChain",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if v, ok := interface{}(m.GetPrivateKey()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TlsCertificateValidationError{
				field:  "PrivateKey",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if v, ok := interface{}(m.GetPrivateKeyProvider()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TlsCertificateValidationError{
				field:  "PrivateKeyProvider",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if v, ok := interface{}(m.GetPassword()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TlsCertificateValidationError{
				field:  "Password",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if v, ok := interface{}(m.GetOcspStaple()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return TlsCertificateValidationError{
				field:  "OcspStaple",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	for idx, item := range m.GetSignedCertificateTimestamp() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return TlsCertificateValidationError{
					field:  fmt.Sprintf("SignedCertificateTimestamp[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	return nil
}

// TlsCertificateValidationError is the validation error returned by
// TlsCertificate.Validate if the designated constraints aren't met.
type TlsCertificateValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e TlsCertificateValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e TlsCertificateValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e TlsCertificateValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e TlsCertificateValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e TlsCertificateValidationError) ErrorName() string { return "TlsCertificateValidationError" }

// Error satisfies the builtin error interface
func (e TlsCertificateValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sTlsCertificate.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = TlsCertificateValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = TlsCertificateValidationError{}

// Validate checks the field values on TlsSessionTicketKeys with the rules
// defined in the proto definition for this message. If any rules are
// violated, an error is returned.
func (m *TlsSessionTicketKeys) Validate() error {
	if m == nil {
		return nil
	}

	if len(m.GetKeys()) < 1 {
		return TlsSessionTicketKeysValidationError{
			field:  "Keys",
			reason: "value must contain at least 1 item(s)",
		}
	}

	for idx, item := range m.GetKeys() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return TlsSessionTicketKeysValidationError{
					field:  fmt.Sprintf("Keys[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	return nil
}

// TlsSessionTicketKeysValidationError is the validation error returned by
// TlsSessionTicketKeys.Validate if the designated constraints aren't met.
type TlsSessionTicketKeysValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e TlsSessionTicketKeysValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e TlsSessionTicketKeysValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e TlsSessionTicketKeysValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e TlsSessionTicketKeysValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e TlsSessionTicketKeysValidationError) ErrorName() string {
	return "TlsSessionTicketKeysValidationError"
}

// Error satisfies the builtin error interface
func (e TlsSessionTicketKeysValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sTlsSessionTicketKeys.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = TlsSessionTicketKeysValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = TlsSessionTicketKeysValidationError{}

// Validate checks the field values on CertificateValidationContext with the
// rules defined in the proto definition for this message. If any rules are
// violated, an error is returned.
func (m *CertificateValidationContext) Validate() error {
	if m == nil {
		return nil
	}

	if v, ok := interface{}(m.GetTrustedCa()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CertificateValidationContextValidationError{
				field:  "TrustedCa",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	for idx, item := range m.GetVerifyCertificateSpki() {
		_, _ = idx, item

		if len(item) != 44 {
			return CertificateValidationContextValidationError{
				field:  fmt.Sprintf("VerifyCertificateSpki[%v]", idx),
				reason: "value length must be 44 bytes",
			}
		}

	}

	for idx, item := range m.GetVerifyCertificateHash() {
		_, _ = idx, item

		if l := len(item); l < 64 || l > 95 {
			return CertificateValidationContextValidationError{
				field:  fmt.Sprintf("VerifyCertificateHash[%v]", idx),
				reason: "value length must be between 64 and 95 bytes, inclusive",
			}
		}

	}

	for idx, item := range m.GetMatchSubjectAltNames() {
		_, _ = idx, item

		if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return CertificateValidationContextValidationError{
					field:  fmt.Sprintf("MatchSubjectAltNames[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if v, ok := interface{}(m.GetRequireOcspStaple()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CertificateValidationContextValidationError{
				field:  "RequireOcspStaple",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if v, ok := interface{}(m.GetRequireSignedCertificateTimestamp()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CertificateValidationContextValidationError{
				field:  "RequireSignedCertificateTimestamp",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if v, ok := interface{}(m.GetCrl()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return CertificateValidationContextValidationError{
				field:  "Crl",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for AllowExpiredCertificate

	if _, ok := CertificateValidationContext_TrustChainVerification_name[int32(m.GetTrustChainVerification())]; !ok {
		return CertificateValidationContextValidationError{
			field:  "TrustChainVerification",
			reason: "value must be one of the defined enum values",
		}
	}

	return nil
}

// CertificateValidationContextValidationError is the validation error returned
// by CertificateValidationContext.Validate if the designated constraints
// aren't met.
type CertificateValidationContextValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e CertificateValidationContextValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e CertificateValidationContextValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e CertificateValidationContextValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e CertificateValidationContextValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e CertificateValidationContextValidationError) ErrorName() string {
	return "CertificateValidationContextValidationError"
}

// Error satisfies the builtin error interface
func (e CertificateValidationContextValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sCertificateValidationContext.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = CertificateValidationContextValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = CertificateValidationContextValidationError{}
