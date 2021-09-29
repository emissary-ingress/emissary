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

package v3alpha1

// This file is support code for enum types.  I'm disappointed that
// controller-gen doesn't generate this.
//
// FIXME(lukeshu): Either patch (and PR) controller-gen to generate
// this, or createa separate code-gen tool to genrate it.
//
//  - Constants for the values of the enum
//
//  - For enums that have an 'int' type in Go:
//     - A MarshalJSON() method
//     - An UnmarshalJSON() method
//     - A String() method

import (
	"encoding/json"
)

const (
	AmbassadorHostState_Initial = AmbassadorHostState(iota)
	AmbassadorHostState_Pending
	AmbassadorHostState_Ready
	AmbassadorHostState_Error
)

var (
	hostState_name = map[AmbassadorHostState]string{
		0: "Initial",
		1: "Pending",
		2: "Ready",
		3: "Error",
	}

	hostState_value = map[string]AmbassadorHostState{
		"Initial": 0,
		"Pending": 1,
		"Ready":   2,
		"Error":   3,
	}
)

func (o AmbassadorHostState) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

func (o AmbassadorHostState) String() string {
	return hostState_name[o]
}

func (o *AmbassadorHostState) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = 0
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	*o = hostState_value[str]
	return nil
}

const (
	AmbassadorHostPhase_NA = AmbassadorHostPhase(iota)
	AmbassadorHostPhase_DefaultsFilled
	AmbassadorHostPhase_ACMEUserPrivateKeyCreated
	AmbassadorHostPhase_ACMEUserRegistered
	AmbassadorHostPhase_ACMECertificateChallenge
)

var (
	hostPhase_name = map[AmbassadorHostPhase]string{
		AmbassadorHostPhase_NA:                        "NA",
		AmbassadorHostPhase_DefaultsFilled:            "DefaultsFilled",
		AmbassadorHostPhase_ACMEUserPrivateKeyCreated: "ACMEUserPrivateKeyCreated",
		AmbassadorHostPhase_ACMEUserRegistered:        "ACMEUserRegistered",
		AmbassadorHostPhase_ACMECertificateChallenge:  "ACMECertificateChallenge",
	}

	hostPhase_value = map[string]AmbassadorHostPhase{
		"NA":                        AmbassadorHostPhase_NA,
		"DefaultsFilled":            AmbassadorHostPhase_DefaultsFilled,
		"ACMEUserPrivateKeyCreated": AmbassadorHostPhase_ACMEUserPrivateKeyCreated,
		"ACMEUserRegistered":        AmbassadorHostPhase_ACMEUserRegistered,
		"ACMECertificateChallenge":  AmbassadorHostPhase_ACMECertificateChallenge,
	}
)

func (o AmbassadorHostPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

func (o AmbassadorHostPhase) String() string {
	return hostPhase_name[o]
}

func (o *AmbassadorHostPhase) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = 0
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	*o = hostPhase_value[str]
	return nil
}

const (
	PreviewURLType_Path = "Path"
)

const (
	HostTLSCertificateSource_Unknown = "Unknown"
	HostTLSCertificateSource_None    = "None"
	HostTLSCertificateSource_Other   = "Other"
	HostTLSCertificateSource_ACME    = "ACME"
)
