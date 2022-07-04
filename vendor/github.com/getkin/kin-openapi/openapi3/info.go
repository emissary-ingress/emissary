package openapi3

import (
	"context"
	"errors"

	"github.com/getkin/kin-openapi/jsoninfo"
)

// Info is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#infoObject
type Info struct {
	ExtensionProps

	Title          string   `json:"title" yaml:"title"` // Required
	Description    string   `json:"description,omitempty" yaml:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License        *License `json:"license,omitempty" yaml:"license,omitempty"`
	Version        string   `json:"version" yaml:"version"` // Required
}

// MarshalJSON returns the JSON encoding of Info.
func (info *Info) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(info)
}

// UnmarshalJSON sets Info to a copy of data.
func (info *Info) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, info)
}

// Validate returns an error if Info does not comply with the OpenAPI spec.
func (info *Info) Validate(ctx context.Context) error {
	if contact := info.Contact; contact != nil {
		if err := contact.Validate(ctx); err != nil {
			return err
		}
	}

	if license := info.License; license != nil {
		if err := license.Validate(ctx); err != nil {
			return err
		}
	}

	if info.Version == "" {
		return errors.New("value of version must be a non-empty string")
	}

	if info.Title == "" {
		return errors.New("value of title must be a non-empty string")
	}

	return nil
}

// Contact is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#contactObject
type Contact struct {
	ExtensionProps

	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// MarshalJSON returns the JSON encoding of Contact.
func (contact *Contact) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(contact)
}

// UnmarshalJSON sets Contact to a copy of data.
func (contact *Contact) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, contact)
}

// Validate returns an error if Contact does not comply with the OpenAPI spec.
func (contact *Contact) Validate(ctx context.Context) error {
	return nil
}

// License is specified by OpenAPI/Swagger standard version 3.
// See https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.3.md#licenseObject
type License struct {
	ExtensionProps

	Name string `json:"name" yaml:"name"` // Required
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// MarshalJSON returns the JSON encoding of License.
func (license *License) MarshalJSON() ([]byte, error) {
	return jsoninfo.MarshalStrictStruct(license)
}

// UnmarshalJSON sets License to a copy of data.
func (license *License) UnmarshalJSON(data []byte) error {
	return jsoninfo.UnmarshalStrictStruct(data, license)
}

// Validate returns an error if License does not comply with the OpenAPI spec.
func (license *License) Validate(ctx context.Context) error {
	if license.Name == "" {
		return errors.New("value of license name must be a non-empty string")
	}
	return nil
}
