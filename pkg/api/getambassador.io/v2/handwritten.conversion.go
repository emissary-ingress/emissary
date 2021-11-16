// This file is ultimately authored by a human, you can ask the build system to generate the
// nescessary signatures for you by running (in the project root)
//
//    make $PWD/pkg/api/getambassador.io/v2/handwritten.conversion.scaffold.go
//
// You can then diff `handwritten.conversion.go` and `handwritten.conversion.scaffold.go` to make
// sure you have all of the functions that conversion-gen thinks you need.

package v2

import (
	"k8s.io/apimachinery/pkg/conversion"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

// These first few functions are written of our own human initiative.

func Convert_string_To_v2_BoolOrString(in *string, out *BoolOrString, s conversion.Scope) error {
	*out = BoolOrString{
		String: in,
	}
	return nil
}

func Convert_v2_MappingLabelGroupsArray_To_v3alpha1_MappingLabelGroupsArray(in *MappingLabelGroupsArray, out *v3alpha1.MappingLabelGroupsArray, s conversion.Scope) error {
	// TODO
	return nil
}
func Convert_v3alpha1_MappingLabelGroupsArray_To_v2_MappingLabelGroupsArray(in *v3alpha1.MappingLabelGroupsArray, out *MappingLabelGroupsArray, s conversion.Scope) error {
	// TODO
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// The remaining functions are all filled out from `handwritten.conversion.scaffold.go` (see the
// comment at the top of the file).  I like to leave in the "WARNING" and "INFO" comments that
// `handwritten.conversion.scaffold.go` has, so that I can (1) compare the comments and the code,
// and make sure the code does everything the comments mention, and (2) compare this file against
// `handwritten.conversion.scaffold.go` to make sure all the comments are there.

func Convert_v2_AddedHeader_To_v3alpha1_AddedHeader(in *AddedHeader, out *v3alpha1.AddedHeader, s conversion.Scope) error {
	if err := autoConvert_v2_AddedHeader_To_v3alpha1_AddedHeader(in, out, s); err != nil {
		return err
	}
	// WARNING: in.Shorthand requires manual conversion: does not exist in peer-type
	// WARNING: in.Full requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v3alpha1_AddedHeader_To_v2_AddedHeader(in *v3alpha1.AddedHeader, out *AddedHeader, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_AddedHeader_To_v2_AddedHeader(in, out, s); err != nil {
		return err
	}
	// WARNING: in.Value requires manual conversion: does not exist in peer-type
	// WARNING: in.Append requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v2_AuthServiceSpec_To_v3alpha1_AuthServiceSpec(in *AuthServiceSpec, out *v3alpha1.AuthServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_AuthServiceSpec_To_v3alpha1_AuthServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (*./pkg/api/getambassador.io/v2.BoolOrString vs string)
	return nil
}

func Convert_v3alpha1_AuthServiceSpec_To_v2_AuthServiceSpec(in *v3alpha1.AuthServiceSpec, out *AuthServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_AuthServiceSpec_To_v2_AuthServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (string vs *./pkg/api/getambassador.io/v2.BoolOrString)
	// WARNING: in.StatsName requires manual conversion: does not exist in peer-type
	// WARNING: in.CircuitBreakers requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v2_CORS_To_v3alpha1_CORS(in *CORS, out *v3alpha1.CORS, s conversion.Scope) error {
	if err := autoConvert_v2_CORS_To_v3alpha1_CORS(in, out, s); err != nil {
		return err
	}
	// WARNING: in.Origins requires manual conversion: inconvertible types (*./pkg/api/getambassador.io/v2.OriginList vs []string)
	return nil
}

func Convert_v3alpha1_CORS_To_v2_CORS(in *v3alpha1.CORS, out *CORS, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_CORS_To_v2_CORS(in, out, s); err != nil {
		return err
	}
	// WARNING: in.Origins requires manual conversion: inconvertible types ([]string vs *./pkg/api/getambassador.io/v2.OriginList)
	return nil
}

func Convert_v2_HostSpec_To_v3alpha1_HostSpec(in *HostSpec, out *v3alpha1.HostSpec, s conversion.Scope) error {
	if err := autoConvert_v2_HostSpec_To_v3alpha1_HostSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.Selector requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v3alpha1_HostSpec_To_v2_HostSpec(in *v3alpha1.HostSpec, out *HostSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_HostSpec_To_v2_HostSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.DeprecatedSelector requires manual conversion: does not exist in peer-type
	// WARNING: in.MappingSelector requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v3alpha1_LogServiceSpec_To_v2_LogServiceSpec(in *v3alpha1.LogServiceSpec, out *LogServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_LogServiceSpec_To_v2_LogServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.StatsName requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v2_MappingLabelSpecifier_To_v3alpha1_MappingLabelSpecifier(in *MappingLabelSpecifier, out *v3alpha1.MappingLabelSpecifier, s conversion.Scope) error {
	if err := autoConvert_v2_MappingLabelSpecifier_To_v3alpha1_MappingLabelSpecifier(in, out, s); err != nil {
		return err
	}
	// WARNING: in.String requires manual conversion: does not exist in peer-type
	// WARNING: in.Header requires manual conversion: does not exist in peer-type
	// WARNING: in.Generic requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v3alpha1_MappingLabelSpecifier_To_v2_MappingLabelSpecifier(in *v3alpha1.MappingLabelSpecifier, out *MappingLabelSpecifier, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_MappingLabelSpecifier_To_v2_MappingLabelSpecifier(in, out, s); err != nil {
		return err
	}
	// WARNING: in.SourceCluster requires manual conversion: does not exist in peer-type
	// WARNING: in.DestinationCluster requires manual conversion: does not exist in peer-type
	// WARNING: in.RequestHeaders requires manual conversion: does not exist in peer-type
	// WARNING: in.RemoteAddress requires manual conversion: does not exist in peer-type
	// WARNING: in.GenericKey requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v2_MappingSpec_To_v3alpha1_MappingSpec(in *MappingSpec, out *v3alpha1.MappingSpec, s conversion.Scope) error {
	if err := autoConvert_v2_MappingSpec_To_v3alpha1_MappingSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (*./pkg/api/getambassador.io/v2.BoolOrString vs string)
	// WARNING: in.UseWebsocket requires manual conversion: does not exist in peer-type
	// WARNING: in.Host requires manual conversion: does not exist in peer-type
	// WARNING: in.HostRegex requires manual conversion: does not exist in peer-type
	// INFO: in.Headers opted out of conversion generation
	// INFO: in.QueryParameters opted out of conversion generation
	return nil
}

func Convert_v3alpha1_MappingSpec_To_v2_MappingSpec(in *v3alpha1.MappingSpec, out *MappingSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_MappingSpec_To_v2_MappingSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (string vs *./pkg/api/getambassador.io/v2.BoolOrString)
	// WARNING: in.DeprecatedUseWebsocket requires manual conversion: does not exist in peer-type
	// WARNING: in.DeprecatedHost requires manual conversion: does not exist in peer-type
	// WARNING: in.DeprecatedHostRegex requires manual conversion: does not exist in peer-type
	// WARNING: in.Hostname requires manual conversion: does not exist in peer-type
	// WARNING: in.StatsName requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v2_RateLimitServiceSpec_To_v3alpha1_RateLimitServiceSpec(in *RateLimitServiceSpec, out *v3alpha1.RateLimitServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_RateLimitServiceSpec_To_v3alpha1_RateLimitServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (*./pkg/api/getambassador.io/v2.BoolOrString vs string)
	return nil
}

func Convert_v3alpha1_RateLimitServiceSpec_To_v2_RateLimitServiceSpec(in *v3alpha1.RateLimitServiceSpec, out *RateLimitServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_RateLimitServiceSpec_To_v2_RateLimitServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (string vs *./pkg/api/getambassador.io/v2.BoolOrString)
	// WARNING: in.StatsName requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v2_TCPMappingSpec_To_v3alpha1_TCPMappingSpec(in *TCPMappingSpec, out *v3alpha1.TCPMappingSpec, s conversion.Scope) error {
	if err := autoConvert_v2_TCPMappingSpec_To_v3alpha1_TCPMappingSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (*./pkg/api/getambassador.io/v2.BoolOrString vs string)
	return nil
}

func Convert_v3alpha1_TCPMappingSpec_To_v2_TCPMappingSpec(in *v3alpha1.TCPMappingSpec, out *TCPMappingSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_TCPMappingSpec_To_v2_TCPMappingSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TLS requires manual conversion: inconvertible types (string vs *./pkg/api/getambassador.io/v2.BoolOrString)
	// WARNING: in.StatsName requires manual conversion: does not exist in peer-type
	return nil
}

func Convert_v3alpha1_TracingServiceSpec_To_v2_TracingServiceSpec(in *v3alpha1.TracingServiceSpec, out *TracingServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_TracingServiceSpec_To_v2_TracingServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.StatsName requires manual conversion: does not exist in peer-type
	return nil
}
