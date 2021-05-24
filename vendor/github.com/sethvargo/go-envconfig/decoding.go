// Copyright The envconfig Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envconfig

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// Base64Bytes is a slice of bytes where the information is base64-encoded in
// the environment variable.
type Base64Bytes []byte

// EnvDecode implements env.Decoder.
func (b *Base64Bytes) EnvDecode(val string) error {
	val = strings.ReplaceAll(val, "+", "-")
	val = strings.ReplaceAll(val, "/", "_")
	val = strings.TrimRight(val, "=")

	var err error
	*b, err = base64.RawURLEncoding.DecodeString(val)
	return err
}

// Bytes returns the underlying bytes.
func (b Base64Bytes) Bytes() []byte {
	return []byte(b)
}

// HexBytes is a slice of bytes where the information is hex-encoded in the
// environment variable.
type HexBytes []byte

// EnvDecode implements env.Decoder.
func (b *HexBytes) EnvDecode(val string) error {
	var err error
	*b, err = hex.DecodeString(val)
	return err
}

// Bytes returns the underlying bytes.
func (b HexBytes) Bytes() []byte {
	return []byte(b)
}
