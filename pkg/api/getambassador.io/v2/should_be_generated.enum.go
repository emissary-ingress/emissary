package v2

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
	HostState_Initial = HostState(iota)
	HostState_Pending
	HostState_Ready
	HostState_Error
)

var (
	hostState_name = map[HostState]string{
		0: "Initial",
		1: "Pending",
		2: "Ready",
		3: "Error",
	}

	hostState_value = map[string]HostState{
		"Initial": 0,
		"Pending": 1,
		"Ready":   2,
		"Error":   3,
	}
)

func (o HostState) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

func (o HostState) String() string {
	return hostState_name[o]
}

func (o *HostState) UnmarshalJSON(data []byte) error {
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
	HostPhase_NA = HostPhase(iota)
	HostPhase_DefaultsFilled
	HostPhase_ACMEUserPrivateKeyCreated
	HostPhase_ACMEUserRegistered
	HostPhase_ACMECertificateChallenge
)

var (
	hostPhase_name = map[HostPhase]string{
		HostPhase_NA:                        "NA",
		HostPhase_DefaultsFilled:            "DefaultsFilled",
		HostPhase_ACMEUserPrivateKeyCreated: "ACMEUserPrivateKeyCreated",
		HostPhase_ACMEUserRegistered:        "ACMEUserRegistered",
		HostPhase_ACMECertificateChallenge:  "ACMECertificateChallenge",
	}

	hostPhase_value = map[string]HostPhase{
		"NA":                        HostPhase_NA,
		"DefaultsFilled":            HostPhase_DefaultsFilled,
		"ACMEUserPrivateKeyCreated": HostPhase_ACMEUserPrivateKeyCreated,
		"ACMEUserRegistered":        HostPhase_ACMEUserRegistered,
		"ACMECertificateChallenge":  HostPhase_ACMECertificateChallenge,
	}
)

func (o HostPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.String())
}

func (o HostPhase) String() string {
	return hostPhase_name[o]
}

func (o *HostPhase) UnmarshalJSON(data []byte) error {
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
	PreviewURLType_Path = "path"
)

const (
	HostTLSCertificateSource_Unknown = "Unknown"
	HostTLSCertificateSource_None    = "None"
	HostTLSCertificateSource_Other   = "Other"
	HostTLSCertificateSource_ACME    = "ACME"
)
