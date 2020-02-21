// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package conversion_test

import (
	"reflect"
	"testing"

	pstruct "github.com/gogo/protobuf/types"

	v2 "github.com/datawire/ambassador/pkg/api/envoy/api/v2"
	core "github.com/datawire/ambassador/pkg/api/envoy/api/v2/core"
	"github.com/datawire/ambassador/pkg/envoy-control-plane/conversion"
)

func TestConversion(t *testing.T) {
	pb := &v2.DiscoveryRequest{
		VersionInfo: "test",
		Node:        &core.Node{Id: "proxy"},
	}
	st, err := conversion.MessageToStruct(pb)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	pbst := map[string]*pstruct.Value{
		"version_info": &pstruct.Value{Kind: &pstruct.Value_StringValue{StringValue: "test"}},
		"node": &pstruct.Value{Kind: &pstruct.Value_StructValue{StructValue: &pstruct.Struct{
			Fields: map[string]*pstruct.Value{
				"id": &pstruct.Value{Kind: &pstruct.Value_StringValue{StringValue: "proxy"}},
			},
		}}},
	}
	if !reflect.DeepEqual(st.Fields, pbst) {
		t.Errorf("MessageToStruct(%v) => got %v, want %v", pb, st.Fields, pbst)
	}

	out := &v2.DiscoveryRequest{}
	err = conversion.StructToMessage(st, out)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if !reflect.DeepEqual(pb, out) {
		t.Errorf("StructToMessage(%v) => got %v, want %v", st, out, pb)
	}

	if _, err = conversion.MessageToStruct(nil); err == nil {
		t.Error("MessageToStruct(nil) => got no error")
	}

	if err = conversion.StructToMessage(nil, &v2.DiscoveryRequest{}); err == nil {
		t.Error("StructToMessage(nil) => got no error")
	}
}
