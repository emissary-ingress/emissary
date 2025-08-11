// Copyright 2020 Datawire.  All rights reserved
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

///////////////////////////////////////////////////////////////////////////
// Important: Run "make generate-fast" to regenerate code after modifying
// this file.
///////////////////////////////////////////////////////////////////////////

package v4alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdditionalLogHeaders struct {
	HeaderName     string `json:"headerName,omitempty" v3:"header_name,omitempty"`
	DuringRequest  *bool  `json:"duringRequest,omitempty" v3:"during_request,omitempty"`
	DuringResponse *bool  `json:"duringResponse,omitempty" v3:"during_response,omitempty"`
	DuringTrailer  *bool  `json:"duringTrailer,omitempty" v3:"during_trailer,omitempty"`
}

type DriverConfig struct {
	AdditionalLogHeaders []*AdditionalLogHeaders `json:"additionalLogHeaders,omitempty" v3:"additional_log_headers,omitempty"`
}

// LogServiceSpec defines the desired state of LogService
type LogServiceSpec struct {
	AmbassadorID AmbassadorID `json:"ambassadorID,omitempty" v3:"ambassador_id,omitempty"`

	Service string `json:"service,omitempty"`

	// ProtocolVersion is the envoy api transport protocol version
	//
	// +kubebuilder:validation:Enum={"v2","v3"}
	ProtocolVersion string `json:"protocolVersion,omitempty" v3:"protocol_version,omitempty"`

	// +kubebuilder:validation:Enum={"tcp","http"}
	Driver                string          `json:"driver,omitempty"`
	DriverConfig          *DriverConfig   `json:"driverConfig,omitempty" v3:"driver_config,omitempty"`
	FlushIntervalTime     *SecondDuration `json:"flushIntervalTime,omitempty" v3:"flush_interval_time,omitempty"`
	FlushIntervalByteSize *int            `json:"flushIntervalByteSize,omitempty" v3:"flush_interval_byte_size,omitempty"`

	GRPC *bool `json:"grpc,omitempty"`

	StatsName string `json:"statsName,omitempty" v3:"stats_name,omitempty"`
}

// LogService is the Schema for the logservices API
//
// +kubebuilder:object:root=true
type LogService struct {
	metav1.TypeMeta   `json:""`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LogServiceSpec `json:"spec,omitempty"`
}

// LogServiceList contains a list of LogServices.
//
// +kubebuilder:object:root=true
type LogServiceList struct {
	metav1.TypeMeta `json:""`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LogService{}, &LogServiceList{})
}
