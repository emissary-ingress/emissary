// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: envoy/config/filter/network/zookeeper_proxy/v1alpha1/zookeeper_proxy.proto

package envoy_config_filter_network_zookeeper_proxy_v1alpha1

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

	"github.com/gogo/protobuf/types"
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
	_ = types.DynamicAny{}
)

// define the regex for a UUID once up-front
var _zookeeper_proxy_uuidPattern = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

// Validate checks the field values on ZooKeeperProxy with the rules defined in
// the proto definition for this message. If any rules are violated, an error
// is returned.
func (m *ZooKeeperProxy) Validate() error {
	if m == nil {
		return nil
	}

	if len(m.GetStatPrefix()) < 1 {
		return ZooKeeperProxyValidationError{
			field:  "StatPrefix",
			reason: "value length must be at least 1 bytes",
		}
	}

	// no validation rules for AccessLog

	{
		tmp := m.GetMaxPacketBytes()

		if v, ok := interface{}(tmp).(interface{ Validate() error }); ok {

			if err := v.Validate(); err != nil {
				return ZooKeeperProxyValidationError{
					field:  "MaxPacketBytes",
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}
	}

	return nil
}

// ZooKeeperProxyValidationError is the validation error returned by
// ZooKeeperProxy.Validate if the designated constraints aren't met.
type ZooKeeperProxyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ZooKeeperProxyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ZooKeeperProxyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ZooKeeperProxyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ZooKeeperProxyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ZooKeeperProxyValidationError) ErrorName() string { return "ZooKeeperProxyValidationError" }

// Error satisfies the builtin error interface
func (e ZooKeeperProxyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sZooKeeperProxy.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ZooKeeperProxyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ZooKeeperProxyValidationError{}
