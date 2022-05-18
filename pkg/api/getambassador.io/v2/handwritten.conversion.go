// This file is ultimately authored by a human, you can ask the build system to generate the
// necessary signatures for you by running (in the project root)
//
//    make $PWD/pkg/api/getambassador.io/v2/handwritten.conversion.scaffold.go
//
// You can then diff `handwritten.conversion.go` and `handwritten.conversion.scaffold.go` to make
// sure you have all of the functions that conversion-gen thinks you need.

package v2

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/conversion"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
)

// These first few functions are written of our own human initiative.

func convert_v2_TLS_To_v3alpha1_TLS(
	inTLS *BoolOrString,
	inSvc string,

	outTLS *string,
	outSvc *string,
	outExplicit **v3alpha1.V2ExplicitTLS,
) {
	// First parse the input.
	// This parsing logic mimics ircluster.py as closely as possible.
	originateTLS := false
	var bareSvc, explicitScheme string
	switch {
	case strings.HasPrefix(strings.ToLower(inSvc), "https://"):
		bareSvc = inSvc[len("https://"):]
		explicitScheme = inSvc[:len("https://")]
		originateTLS = true
	case strings.HasPrefix(strings.ToLower(inSvc), "http://"):
		bareSvc = inSvc[len("http://"):]
		explicitScheme = inSvc[:len("http://")]
	default:
		bareSvc = inSvc
	}
	var ctxName string
	switch {
	case inTLS != nil && inTLS.String != nil && *inTLS.String != "":
		originateTLS = true
		ctxName = *inTLS.String
	case inTLS != nil && inTLS.Bool != nil && *inTLS.Bool:
		originateTLS = *inTLS.Bool
	}
	// OK, now re-serialize that in v3alpha1 syntax.
	if ctxName != "" {
		*outTLS = *inTLS.String
		*outSvc = inSvc
		*outExplicit = &v3alpha1.V2ExplicitTLS{}
	} else {
		// 1. outTLS
		*outTLS = ""
		var outExplicitTLS string
		switch {
		case inTLS == nil:
			outExplicitTLS = ""
		case inTLS.Bool != nil && *inTLS.Bool:
			outExplicitTLS = "bool:true"
		case inTLS.Bool != nil && !*inTLS.Bool:
			outExplicitTLS = "bool:false"
		case inTLS.String != nil && *inTLS.String == "":
			outExplicitTLS = "string"
		case *inTLS == BoolOrString{}:
			outExplicitTLS = "null"
		}

		// 2. outSvc
		//
		//                  | explicitScheme="https://" | explicitScheme="http://" |
		//  | originateTLS  | inSvc                     | force-https://           |
		//  | !originateTLS | not possible              | inSvc                    |
		//
		// Because an https:// scheme forces originateTLS=true, the bottom-left corner isn't
		// possible.
		var outExplicitScheme *string
		schemeIsHTTPS := strings.ToLower(explicitScheme) == "https://"
		if schemeIsHTTPS == originateTLS {
			// cover the diagonal
			*outSvc = inSvc
		} else {
			// cover the top-right
			outExplicitScheme = &explicitScheme
			*outSvc = "https://" + bareSvc
		}

		// 3. outExplicit
		if outExplicitTLS == "" && outExplicitScheme == nil {
			*outExplicit = nil
		} else {
			*outExplicit = &v3alpha1.V2ExplicitTLS{
				TLS:           outExplicitTLS,
				ServiceScheme: outExplicitScheme,
			}
		}
	}
}

func convert_v3alpha1_TLS_To_v2_TLS(
	inTLS string,
	inSvc string,
	inExplicit *v3alpha1.V2ExplicitTLS,

	outTLS **BoolOrString,
	outSvc *string,
) {
	if inExplicit == nil {
		inExplicit = &v3alpha1.V2ExplicitTLS{}
	}
	// First parse the input.
	// This parsing logic mimics ircluster.py as closely as possible.
	originateTLS := false
	var bareSvc string
	switch {
	case strings.HasPrefix(strings.ToLower(inSvc), "https://"):
		bareSvc = inSvc[len("https://"):]
		originateTLS = true
	case strings.HasPrefix(strings.ToLower(inSvc), "http://"):
		bareSvc = inSvc[len("http://"):]
	default:
		bareSvc = inSvc
	}
	if inTLS != "" {
		originateTLS = true
	}
	// OK, now re-serialize that in v2 syntax.
	tlsIsTruthy := false
	if inTLS != "" {
		*outTLS = &BoolOrString{
			String: &inTLS,
		}
		tlsIsTruthy = true
	} else {
		switch inExplicit.TLS {
		case "null":
			*outTLS = &BoolOrString{}
		case "bool:false":
			val := false
			*outTLS = &BoolOrString{
				Bool: &val,
			}
		case "bool:true":
			if originateTLS {
				val := true
				*outTLS = &BoolOrString{
					Bool: &val,
				}
				tlsIsTruthy = true
			}
		case "string":
			val := ""
			*outTLS = &BoolOrString{
				String: &val,
			}
		}
	}
	if tlsIsTruthy {
		// .tls overrides the .service scheme, so in this case the explicit scheme can be
		// whatever the user wants.
		if inExplicit.ServiceScheme != nil {
			*outSvc = *inExplicit.ServiceScheme + bareSvc
		} else {
			*outSvc = inSvc
		}
	} else {
		// If !tlsIsTruthy, then the schema is what determines originate TLS; which means
		// that `strings.HasPrefix(strings.ToLower(inSvc), "https://") == originateTLS`.
		if (inExplicit.ServiceScheme != nil) && ((strings.ToLower(*inExplicit.ServiceScheme) == "https://") == originateTLS) {
			*outSvc = *inExplicit.ServiceScheme + bareSvc
		} else { // if strings.HasPrefix(strings.ToLower(inSvc), "https://") == originateTLS { // `if` clause unnecessary, see above
			*outSvc = inSvc
		}
	}
}

const ambassadorIDMangle = "--apiVersion-v3alpha1-only--"

func Convert_v2_AmbassadorID_To_v3alpha1_AmbassadorID(in *AmbassadorID, out *v3alpha1.AmbassadorID, s conversion.Scope) error {
	if *in == nil {
		*out = nil
		return nil
	}
	list := make(v3alpha1.AmbassadorID, 0, len(*in))
	for _, item := range *in {
		list = append(list, strings.TrimPrefix(item, ambassadorIDMangle))
	}
	*out = list
	return nil
}

// Pass DisableManglingAmbassadorID as the context to scheme.Convert() to disable mangling
// ambassador_id when converting from v3alpha1 to v2.
type DisableManglingAmbassadorID struct{}

func Convert_v3alpha1_AmbassadorID_To_v2_AmbassadorID(in *v3alpha1.AmbassadorID, out *AmbassadorID, s conversion.Scope) error {
	mangle := true
	if meta := s.Meta(); meta != nil {
		if _, disable := meta.Context.(DisableManglingAmbassadorID); disable {
			mangle = false
		}
	}

	if *in == nil {
		if mangle {
			in = &v3alpha1.AmbassadorID{"default"}
		} else {
			*out = nil
			return nil
		}
	}
	list := make(AmbassadorID, 0, len(*in))
	for _, item := range *in {
		item = strings.TrimPrefix(item, ambassadorIDMangle)
		if mangle {
			item = ambassadorIDMangle + item
		}
		list = append(list, item)
	}
	*out = list
	return nil
}

func Convert_string_To_v2_BoolOrString(in *string, out *BoolOrString, s conversion.Scope) error {
	*out = BoolOrString{
		String: in,
	}
	return nil
}

func Convert_string_To_Pointer_v2_BoolOrString(in *string, out **BoolOrString, s conversion.Scope) error {
	if *in != "" {
		*out = &BoolOrString{
			String: in,
		}
	} else {
		*out = nil
	}
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
	switch {
	// WARNING: in.Shorthand requires manual conversion: does not exist in peer-type
	case in.Shorthand != nil:
		*out = v3alpha1.AddedHeader{
			Value:            *in.Shorthand,
			V2Representation: "string",
		}
	// WARNING: in.Full requires manual conversion: does not exist in peer-type
	case in.Full != nil:
		*out = v3alpha1.AddedHeader{
			Value:  in.Full.Value,
			Append: in.Full.Append,
		}
	default:
		*out = v3alpha1.AddedHeader{
			V2Representation: "null",
		}
	}
	return nil
}

func Convert_v3alpha1_AddedHeader_To_v2_AddedHeader(in *v3alpha1.AddedHeader, out *AddedHeader, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_AddedHeader_To_v2_AddedHeader(in, out, s); err != nil {
		return err
	}
	// WARNING: in.Value requires manual conversion: does not exist in peer-type
	// WARNING: in.Append requires manual conversion: does not exist in peer-type
	// WARNING: in.V2Representation requires manual conversion: does not exist in peer-type
	switch {
	case in.V2Representation == "string" && in.Append == nil:
		*out = AddedHeader{
			Shorthand: &in.Value,
		}
	case in.V2Representation == "null" && in.Append == nil && in.Value == "":
		*out = AddedHeader{}
	default:
		*out = AddedHeader{
			Full: &AddedHeaderFull{
				Value:  in.Value,
				Append: in.Append,
			},
		}
	}
	return nil
}

func Convert_v2_AuthServiceSpec_To_v3alpha1_AuthServiceSpec(in *AuthServiceSpec, out *v3alpha1.AuthServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_AuthServiceSpec_To_v3alpha1_AuthServiceSpec(in, out, s); err != nil {
		return err
	}
	// INFO: in.TLS opted out of conversion generation via +k8s:conversion-gen=false
	convert_v2_TLS_To_v3alpha1_TLS(
		in.TLS, in.AuthService,
		&out.TLS, &out.AuthService, &out.V2ExplicitTLS)
	return nil
}

func Convert_v3alpha1_AuthServiceSpec_To_v2_AuthServiceSpec(in *v3alpha1.AuthServiceSpec, out *AuthServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_AuthServiceSpec_To_v2_AuthServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.V2ExplicitTLS requires manual conversion: does not exist in peer-type
	convert_v3alpha1_TLS_To_v2_TLS(
		in.TLS, in.AuthService, in.V2ExplicitTLS,
		&out.TLS, &out.AuthService)
	return nil
}

func Convert_v2_CORS_To_v3alpha1_CORS(in *CORS, out *v3alpha1.CORS, s conversion.Scope) error {
	if err := autoConvert_v2_CORS_To_v3alpha1_CORS(in, out, s); err != nil {
		return err
	}

	// INFO: in.Origins opted out of conversion generation via +k8s:conversion-gen=false
	if in.Origins != nil {
		out.Origins = in.Origins.Values
		out.V2CommaSeparatedOrigins = in.Origins.CommaSeparated
	}

	return nil
}

func Convert_v3alpha1_CORS_To_v2_CORS(in *v3alpha1.CORS, out *CORS, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_CORS_To_v2_CORS(in, out, s); err != nil {
		return err
	}
	// INFO: in.Origins opted out of conversion generation via +k8s:conversion-gen=false
	// WARNING: in.V2CommaSeparatedOrigins requires manual conversion: does not exist in peer-type
	if in.Origins != nil {
		out.Origins = &OriginList{
			Values:         in.Origins,
			CommaSeparated: in.V2CommaSeparatedOrigins,
		}
	}

	return nil
}

func Convert_v2_HostSpec_To_v3alpha1_HostSpec(in *HostSpec, out *v3alpha1.HostSpec, s conversion.Scope) error {
	if len(in.DeprecatedAmbassadorID) > 0 {
		in = in.DeepCopy()
		in.AmbassadorID = append(in.AmbassadorID, in.DeprecatedAmbassadorID...)
		in.DeprecatedAmbassadorID = nil
	}

	if err := autoConvert_v2_HostSpec_To_v3alpha1_HostSpec(in, out, s); err != nil {
		return err
	}

	// WARNING: in.DeprecatedAmbassadorID requires manual conversion: does not exist in peer-type
	// (see above)

	return nil
}

func Convert_v3alpha1_HostSpec_To_v2_HostSpec(in *v3alpha1.HostSpec, out *HostSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_HostSpec_To_v2_HostSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.DeprecatedSelector requires manual conversion: does not exist in peer-type
	if in.DeprecatedSelector != nil && out.Selector == nil {
		out.Selector = in.DeprecatedSelector
	}
	return nil
}

func Convert_v2_MappingLabelSpecifier_To_v3alpha1_MappingLabelSpecifier(in *MappingLabelSpecifier, out *v3alpha1.MappingLabelSpecifier, s conversion.Scope) error {
	if err := autoConvert_v2_MappingLabelSpecifier_To_v3alpha1_MappingLabelSpecifier(in, out, s); err != nil {
		return err
	}
	switch {
	// WARNING: in.String requires manual conversion: does not exist in peer-type
	case in.String != nil:
		switch *in.String {
		case "source_cluster":
			out.SourceCluster = &v3alpha1.MappingLabelSpecifier_SourceCluster{
				Key: *in.String,
			}
		case "destination_cluster":
			out.DestinationCluster = &v3alpha1.MappingLabelSpecifier_DestinationCluster{
				Key: *in.String,
			}
		case "remote_address":
			out.RemoteAddress = &v3alpha1.MappingLabelSpecifier_RemoteAddress{
				Key: *in.String,
			}
		default:
			out.GenericKey = &v3alpha1.MappingLabelSpecifier_GenericKey{
				Value:       *in.String,
				V2Shorthand: true,
			}
		}
	// WARNING: in.Header requires manual conversion: does not exist in peer-type
	case in.Header != nil:
		for k, v := range in.Header {
			out.RequestHeaders = &v3alpha1.MappingLabelSpecifier_RequestHeaders{
				Key:              k,
				HeaderName:       v.Header,
				OmitIfNotPresent: v.OmitIfNotPresent,
			}
		}
	// WARNING: in.Generic requires manual conversion: does not exist in peer-type
	case in.Generic != nil:
		out.GenericKey = &v3alpha1.MappingLabelSpecifier_GenericKey{
			Value: in.Generic.GenericKey,
		}
	}
	return nil
}

func Convert_v3alpha1_MappingLabelSpecifier_To_v2_MappingLabelSpecifier(in *v3alpha1.MappingLabelSpecifier, out *MappingLabelSpecifier, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_MappingLabelSpecifier_To_v2_MappingLabelSpecifier(in, out, s); err != nil {
		return err
	}
	switch {
	// WARNING: in.SourceCluster requires manual conversion: does not exist in peer-type
	case in.SourceCluster != nil:
		out.String = &in.SourceCluster.Key
	// WARNING: in.DestinationCluster requires manual conversion: does not exist in peer-type
	case in.DestinationCluster != nil:
		out.String = &in.DestinationCluster.Key
	// WARNING: in.RequestHeaders requires manual conversion: does not exist in peer-type
	case in.RequestHeaders != nil:
		out.Header = MappingLabelSpecHeader{
			in.RequestHeaders.Key: MappingLabelSpecHeaderStruct{
				Header:           in.RequestHeaders.HeaderName,
				OmitIfNotPresent: in.RequestHeaders.OmitIfNotPresent,
			},
		}
	// WARNING: in.RemoteAddress requires manual conversion: does not exist in peer-type
	case in.RemoteAddress != nil:
		out.String = &in.RemoteAddress.Key
	// WARNING: in.GenericKey requires manual conversion: does not exist in peer-type
	case in.GenericKey != nil:
		if in.GenericKey.V2Shorthand && in.GenericKey.Key == "" {
			out.String = &in.GenericKey.Value
		} else {
			out.Generic = &MappingLabelSpecGeneric{
				V3Key:      in.GenericKey.Key,
				GenericKey: in.GenericKey.Value,
			}
		}
	}
	return nil
}

func Convert_v2_MappingSpec_To_v3alpha1_MappingSpec(in *MappingSpec, out *v3alpha1.MappingSpec, s conversion.Scope) error {
	if err := autoConvert_v2_MappingSpec_To_v3alpha1_MappingSpec(in, out, s); err != nil {
		return err
	}

	// INFO: in.TLS opted out of conversion generation via +k8s:conversion-gen=false
	convert_v2_TLS_To_v3alpha1_TLS(
		in.TLS, in.Service,
		&out.TLS, &out.Service, &out.V2ExplicitTLS)

	// INFO: in.Headers opted out of conversion generation via +k8s:conversion-gen=false
	for k, v := range in.Headers {
		switch {
		case v.String != nil:
			if out.Headers == nil {
				out.Headers = make(map[string]string)
			}
			out.Headers[k] = *v.String
		case v.Bool != nil && *v.Bool:
			if out.RegexHeaders == nil {
				out.RegexHeaders = make(map[string]string)
			}
			if _, conflict := out.RegexHeaders[k]; !conflict {
				out.RegexHeaders[k] = ".*"
			}
			out.V2BoolHeaders = append(out.V2BoolHeaders, k)
		}
	}

	// INFO: in.QueryParameters opted out of conversion generation via +k8s:conversion-gen=false
	for k, v := range in.QueryParameters {
		switch {
		case v.String != nil:
			if out.QueryParameters == nil {
				out.QueryParameters = make(map[string]string)
			}
			out.QueryParameters[k] = *v.String
		case v.Bool != nil && *v.Bool:
			if out.RegexQueryParameters == nil {
				out.RegexQueryParameters = make(map[string]string)
			}
			if _, conflict := out.RegexQueryParameters[k]; !conflict {
				out.RegexQueryParameters[k] = ".*"
			}
			out.V2BoolQueryParameters = append(out.V2BoolQueryParameters, k)
		}
	}

	if out.DeprecatedHostRegex != nil && *out.DeprecatedHostRegex {
		out.DeprecatedHost = out.Hostname
		out.Hostname = ""
	} else if out.Hostname == "" {
		out.Hostname = "*"
	} else if out.Hostname == "_skip_mapping_with_empty_host_" {
		out.Hostname = ""
	}

	return nil
}

func Convert_v3alpha1_MappingSpec_To_v2_MappingSpec(in *v3alpha1.MappingSpec, out *MappingSpec, s conversion.Scope) error {
	in = in.DeepCopy()
	if in.Hostname != "" {
		in.DeprecatedHost = ""
		if in.DeprecatedHostRegex != nil {
			v := false
			in.DeprecatedHostRegex = &v
		}
	} else if in.DeprecatedHost == "" && (in.DeprecatedHostRegex == nil || !*in.DeprecatedHostRegex) {
		in.Hostname = "_skip_mapping_with_empty_host_"
	}

	if err := autoConvert_v3alpha1_MappingSpec_To_v2_MappingSpec(in, out, s); err != nil {
		return err
	}

	// WARNING: in.DeprecatedHost requires manual conversion: does not exist in peer-type
	if in.DeprecatedHost != "" {
		out.Host = in.DeprecatedHost
	}
	if (out.HostRegex == nil || !*out.HostRegex) && out.Host == "*" {
		out.Host = ""
	}

	// WARNING: in.V2ExplicitTLS requires manual conversion: does not exist in peer-type
	convert_v3alpha1_TLS_To_v2_TLS(
		in.TLS, in.Service, in.V2ExplicitTLS,
		&out.TLS, &out.Service)

	// WARNING: in.V2BoolHeaders requires manual conversion: does not exist in peer-type
	if out.RegexHeaders != nil {
		for _, name := range in.V2BoolHeaders {
			if re, exists := out.RegexHeaders[name]; exists {
				if out.Headers == nil {
					out.Headers = make(map[string]BoolOrString)
				}
				if _, conflict := out.Headers[name]; !conflict {
					val := true
					out.Headers[name] = BoolOrString{Bool: &val}
					if re == ".*" {
						delete(out.RegexHeaders, name)
					}
				}
			}
		}
	}

	// WARNING: in.V2BoolQueryParameters requires manual conversion: does not exist in peer-type
	if out.RegexQueryParameters != nil {
		for _, name := range in.V2BoolQueryParameters {
			if re, exists := out.RegexQueryParameters[name]; exists {
				if out.QueryParameters == nil {
					out.QueryParameters = make(map[string]BoolOrString)
				}
				if _, conflict := out.QueryParameters[name]; !conflict {
					val := true
					out.QueryParameters[name] = BoolOrString{Bool: &val}
					if re == ".*" {
						delete(out.RegexQueryParameters, name)
					}
				}
			}
		}
	}

	return nil
}

func Convert_v2_RateLimitServiceSpec_To_v3alpha1_RateLimitServiceSpec(in *RateLimitServiceSpec, out *v3alpha1.RateLimitServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_RateLimitServiceSpec_To_v3alpha1_RateLimitServiceSpec(in, out, s); err != nil {
		return err
	}
	// INFO: in.TLS opted out of conversion generation via +k8s:conversion-gen=false
	convert_v2_TLS_To_v3alpha1_TLS(
		in.TLS, in.Service,
		&out.TLS, &out.Service, &out.V2ExplicitTLS)
	return nil
}

func Convert_v3alpha1_RateLimitServiceSpec_To_v2_RateLimitServiceSpec(in *v3alpha1.RateLimitServiceSpec, out *RateLimitServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v3alpha1_RateLimitServiceSpec_To_v2_RateLimitServiceSpec(in, out, s); err != nil {
		return err
	}

	// WARNING: in.V2ExplicitTLS requires manual conversion: does not exist in peer-type
	convert_v3alpha1_TLS_To_v2_TLS(
		in.TLS, in.Service, in.V2ExplicitTLS,
		&out.TLS, &out.Service)

	return nil
}

func Convert_v2_TCPMappingSpec_To_v3alpha1_TCPMappingSpec(in *TCPMappingSpec, out *v3alpha1.TCPMappingSpec, s conversion.Scope) error {
	if err := autoConvert_v2_TCPMappingSpec_To_v3alpha1_TCPMappingSpec(in, out, s); err != nil {
		return err
	}

	// INFO: in.TLS opted out of conversion generation via +k8s:conversion-gen=false
	convert_v2_TLS_To_v3alpha1_TLS(
		in.TLS, in.Service,
		&out.TLS, &out.Service, &out.V2ExplicitTLS)
	// Don't ever set V2ExplicitTLS.ServiceScheme; v2 did not allow schemes for TCPMappings.
	if out.V2ExplicitTLS != nil {
		if out.V2ExplicitTLS.TLS != "" {
			out.V2ExplicitTLS = &v3alpha1.V2ExplicitTLS{
				TLS: out.V2ExplicitTLS.TLS,
			}
		} else {
			out.V2ExplicitTLS = nil
		}
	}

	return nil
}

func Convert_v3alpha1_TCPMappingSpec_To_v2_TCPMappingSpec(in *v3alpha1.TCPMappingSpec, out *TCPMappingSpec, s conversion.Scope) error {
	// Force V2ExplicitTLS.ServiceScheme=strPtr(""); v2 did not allow schemes for TCPMappings.
	in = in.DeepCopy()
	if in.V2ExplicitTLS == nil {
		in.V2ExplicitTLS = &v3alpha1.V2ExplicitTLS{}
	}
	if in.V2ExplicitTLS.ServiceScheme == nil || *in.V2ExplicitTLS.ServiceScheme != "" {
		val := ""
		in.V2ExplicitTLS.ServiceScheme = &val
	}

	if err := autoConvert_v3alpha1_TCPMappingSpec_To_v2_TCPMappingSpec(in, out, s); err != nil {
		return err
	}

	// WARNING: in.V2ExplicitTLS requires manual conversion: does not exist in peer-type
	convert_v3alpha1_TLS_To_v2_TLS(
		in.TLS, in.Service, in.V2ExplicitTLS,
		&out.TLS, &out.Service)

	return nil
}

func Convert_v2_TracingServiceSpec_To_v3alpha1_TracingServiceSpec(in *TracingServiceSpec, out *v3alpha1.TracingServiceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_TracingServiceSpec_To_v3alpha1_TracingServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.TagHeaders requires manual conversion: does not exist in peer-type
	// if only tag_headers are set, translate to custom_tags.
	// if both are set, ignore tag_headers.
	if in.TagHeaders != nil {
		if in.V3CustomTags == nil {
			out.CustomTags = []v3alpha1.TracingCustomTag{}
			for _, tag := range in.TagHeaders {
				out.CustomTags = append(out.CustomTags, v3alpha1.TracingCustomTag{
					Tag: tag,
					Header: &v3alpha1.TracingCustomTagTypeRequestHeader{
						Name: tag,
					},
				})
			}
		}
	}
	return nil
}

func Convert_v3alpha1_TracingServiceSpec_To_v2_TracingServiceSpec(in *v3alpha1.TracingServiceSpec, out *TracingServiceSpec, s conversion.Scope) error {
	in = in.DeepCopy()
	// if only tag_headers are set, translate to custom_tags.
	// if both are set, log a warning and ignore tag_headers.
	if in.DeprecatedTagHeaders != nil {
		if in.CustomTags == nil {
			in.CustomTags = []v3alpha1.TracingCustomTag{}
			for _, tag := range in.DeprecatedTagHeaders {
				in.CustomTags = append(in.CustomTags, v3alpha1.TracingCustomTag{
					Tag: tag,
					Header: &v3alpha1.TracingCustomTagTypeRequestHeader{
						Name: tag,
					},
				})
			}
		} else {
			// TODO: Use dlog logger
			fmt.Printf("CustomTags and TagHeaders cannot be set at the same time in a TracingService. ignoring TagHeaders since it is deprecated.")
		}
	}

	if err := autoConvert_v3alpha1_TracingServiceSpec_To_v2_TracingServiceSpec(in, out, s); err != nil {
		return err
	}
	// WARNING: in.DeprecatedTagHeaders requires manual conversion: does not exist in peer-type
	// see above
	return nil
}
