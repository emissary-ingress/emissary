package v2

import (
	"encoding/json"
	"fmt"
	"strings"
	unsafe "unsafe"

	"github.com/datawire/ambassador/v2/pkg/api/getambassador.io/v3alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conversion "k8s.io/apimachinery/pkg/conversion"
)

func Convert_v2_AddedHeader_To_v3alpha1_AddedHeader(in *AddedHeader, out *v3alpha1.AddedHeader, s conversion.Scope) error {
	if (in.Bool != nil) && (*in.Bool) {
		return errors.New("impossible: AddedHeader has boolean value")
	}

	if in.String != nil {
		if *in.String == "" {
			return errors.New("impossible: AddedHeader has empty string value")
		}

		out.Value = *in.String
	} else {
		// OK, UntypedDicts are awful. The keys are strings, but the values are
		// json.RawMessages. Here, the allowed values are "append" with a bool value,
		// and "value" with a string value.

		if in.Object != nil {
			rawAppend, found := (*in.Object).Values["append"]

			if found && (len(rawAppend.raw) > 0) {
				err := json.Unmarshal(rawAppend.raw, &out.Append)

				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("v2 AddedHeader %#v has invalid 'append'", in.Object))
				}
			}

			rawValue, found := (*in.Object).Values["value"]

			if found && (len(rawValue.raw) > 0) {
				err := json.Unmarshal(rawValue.raw, &out.Value)

				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("v2 AddedHeader %#v has invalid 'value'", in.Object))
				}
			}
		}
	}

	return nil
}

func Convert_v3alpha1_AddedHeader_To_v2_AddedHeader(in *v3alpha1.AddedHeader, out *AddedHeader, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 AddedHeader back to v2 AddedHeader")
}

func Convert_v2_AuthServiceSpec_To_v3alpha1_AuthServiceSpec(in *AuthServiceSpec, out *v3alpha1.AuthServiceSpec, s conversion.Scope) error {
	err := autoConvert_v2_AuthServiceSpec_To_v3alpha1_AuthServiceSpec(in, out, s)

	if err != nil {
		return err
	}

	if in.AddAuthHeaders != nil {
		in, out := &in.AddAuthHeaders, &out.AddAuthHeaders
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			if val.Bool != nil {
				return fmt.Errorf("impossible: AuthServiceSpec.AddAuthHeaders[%q] has boolean value", key)
			}
			(*out)[key] = *val.String
		}
	} else {
		out.AddAuthHeaders = nil
	}

	outTLS, outService := convertTLS(in.TLS, in.AuthService)

	out.AuthService = outService

	if outTLS != "" {
		out.TLS = outTLS
	}

	return nil
}

func Convert_v3alpha1_AuthServiceSpec_To_v2_AuthServiceSpec(in *v3alpha1.AuthServiceSpec, out *AuthServiceSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 AuthService back to v2 AuthService")
}

func Convert_v2_CORS_To_v3alpha1_CORS(in *CORS, out *v3alpha1.CORS, s conversion.Scope) error {
	out.Methods = *(*[]string)(unsafe.Pointer(&in.Methods))
	out.Headers = *(*[]string)(unsafe.Pointer(&in.Headers))
	out.Credentials = in.Credentials
	out.ExposedHeaders = *(*[]string)(unsafe.Pointer(&in.ExposedHeaders))
	out.MaxAge = in.MaxAge

	out.Origins = make([]string, 0)

	if in.Origins.String != nil {
		out.Origins = append(out.Origins, *in.Origins.String)
	} else if in.Origins.ListOfStrings != nil {
		out.Origins = append(out.Origins, *in.Origins.ListOfStrings...)
	}

	return nil
}

func Convert_v3alpha1_CORS_To_v2_CORS(in *v3alpha1.CORS, out *CORS, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 CORS back to v2 CORS")
}

func Convert_v2_HostSpec_To_v3alpha1_HostSpec(in *HostSpec, out *v3alpha1.HostSpec, s conversion.Scope) error {
	err := autoConvert_v2_HostSpec_To_v3alpha1_HostSpec(in, out, s)

	if err != nil {
		return err
	}

	out.DeprecatedSelector = (*metav1.LabelSelector)(unsafe.Pointer(in.Selector))

	return nil
}

func Convert_v3alpha1_HostSpec_To_v2_HostSpec(in *v3alpha1.HostSpec, out *HostSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 Host back to v2 Host")
}

func Convert_v3alpha1_LogServiceSpec_To_v2_LogServiceSpec(in *v3alpha1.LogServiceSpec, out *LogServiceSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 LogServiceSpec back to v2 LogServiceSpec")
}

func Convert_v2_MappingLabelSpecifier_To_v3alpha1_MappingLabelSpecifier(in *MappingLabelSpecifier, out *v3alpha1.MappingLabelSpecifier, s conversion.Scope) error {
	if (in.String != nil) && (*in.String != "") {
		switch *in.String {
		case "source_cluster":
			out.SourceCluster = &v3alpha1.MappingLabelSpecifier_SourceCluster{Key: "source_cluster"}
		case "destination_cluster":
			out.DestinationCluster = &v3alpha1.MappingLabelSpecifier_DestinationCluster{Key: "destination_cluster"}
		case "remote_address":
			out.RemoteAddress = &v3alpha1.MappingLabelSpecifier_RemoteAddress{Key: "remote_address"}
		default:
			out.GenericKey = &v3alpha1.MappingLabelSpecifier_GenericKey{Key: "generic_key", Value: *in.String}
		}
	} else if len(in.Header) == 1 {
		tooMany := false

		for k, v := range in.Header {
			if tooMany {
				return fmt.Errorf("v2 MappingLabelSpecifier: too many headers specified")
			}

			tooMany = true
			out.RequestHeaders = &v3alpha1.MappingLabelSpecifier_RequestHeaders{
				Key:        k,
				HeaderName: v.Header,
			}

			if v.OmitIfNotPresent != nil {
				out.RequestHeaders.OmitIfNotPresent = v.OmitIfNotPresent
			}
		}
	} else if in.Generic != nil {
		out.GenericKey = &v3alpha1.MappingLabelSpecifier_GenericKey{Key: "generic_key", Value: *&in.Generic.GenericKey}
	}

	return nil
}

func Convert_v3alpha1_MappingLabelSpecifier_To_v2_MappingLabelSpecifier(in *v3alpha1.MappingLabelSpecifier, out *MappingLabelSpecifier, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 MappingLabelSpecifier back to v2 MappingLabelSpecifier")
}

func Convert_v2_MappingLabelGroupsArray_To_v3alpha1_MappingLabelGroupsArray(in *MappingLabelGroupsArray, out *v3alpha1.MappingLabelGroupsArray, s conversion.Scope) error {
	// I don't really understand why this can't be autogenerated but whatever.
	if len(*in) > 0 {
		outLabelGroupsArray := make([]v3alpha1.MappingLabelGroup, len(*in))

		for _, inLabelGroup := range *in {
			// A MappingLabelGroup (inLabelGroup) is a map[string]MappingLabelsArray.
			outLabelGroup := make(map[string]v3alpha1.MappingLabelsArray, 0)

			for key, inLabelsArray := range inLabelGroup {
				outLabelsArray := make([]v3alpha1.MappingLabelSpecifier, len(inLabelsArray))

				for _, inLabel := range inLabelsArray {
					outLabel := v3alpha1.MappingLabelSpecifier{}
					err := Convert_v2_MappingLabelSpecifier_To_v3alpha1_MappingLabelSpecifier(&inLabel, &outLabel, s)
					if err != nil {
						return err
					}

					outLabelsArray = append(outLabelsArray, outLabel)
				}

				outLabelGroup[key] = outLabelsArray
			}

			outLabelGroupsArray = append(outLabelGroupsArray, outLabelGroup)
		}

		*out = outLabelGroupsArray
	}

	return nil
}

func Convert_v3alpha1_MappingLabelGroupsArray_To_v2_MappingLabelGroupsArray(in *v3alpha1.MappingLabelGroupsArray, out *MappingLabelGroupsArray, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 MappingLabelGroupsArray back to v2 MappingLabelGroupsArray")
}

func Convert_v2_MappingSpec_To_v3alpha1_MappingSpec(in *MappingSpec, out *v3alpha1.MappingSpec, s conversion.Scope) error {
	err := autoConvert_v2_MappingSpec_To_v3alpha1_MappingSpec(in, out, s)

	if err != nil {
		return err
	}

	outTLS, outService := convertTLS(in.TLS, in.Service)

	out.Service = outService

	if outTLS != "" {
		out.TLS = outTLS
	}

	out.DeprecatedHost = in.Host
	out.DeprecatedHostRegex = in.HostRegex

	return nil
}

func Convert_v3alpha1_MappingSpec_To_v2_MappingSpec(in *v3alpha1.MappingSpec, out *MappingSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 MappingSpec back to v2 MappingSpec")
}

func Convert_v2_RateLimitServiceSpec_To_v3alpha1_RateLimitServiceSpec(in *RateLimitServiceSpec, out *v3alpha1.RateLimitServiceSpec, s conversion.Scope) error {
	err := autoConvert_v2_RateLimitServiceSpec_To_v3alpha1_RateLimitServiceSpec(in, out, s)

	if err != nil {
		return err
	}

	outTLS, outService := convertTLS(in.TLS, in.Service)

	out.Service = outService

	if outTLS != "" {
		out.TLS = outTLS
	}

	return nil
}

func Convert_v3alpha1_RateLimitServiceSpec_To_v2_RateLimitServiceSpec(in *v3alpha1.RateLimitServiceSpec, out *RateLimitServiceSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 RateLimitServiceSpec back to v2 RateLimitServiceSpec")
}

func Convert_v2_TCPMappingSpec_To_v3alpha1_TCPMappingSpec(in *TCPMappingSpec, out *v3alpha1.TCPMappingSpec, s conversion.Scope) error {
	err := autoConvert_v2_TCPMappingSpec_To_v3alpha1_TCPMappingSpec(in, out, s)

	if err != nil {
		return err
	}

	outTLS, outService := convertTLS(in.TLS, in.Service)

	out.Service = outService

	if outTLS != "" {
		out.TLS = outTLS
	}

	return nil
}

func Convert_v3alpha1_TCPMappingSpec_To_v2_TCPMappingSpec(in *v3alpha1.TCPMappingSpec, out *TCPMappingSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 TCPMappingSpec back to v2 TCPMappingSpec")
}

func Convert_v3alpha1_TracingServiceSpec_To_v2_TracingServiceSpec(in *v3alpha1.TracingServiceSpec, out *TracingServiceSpec, s conversion.Scope) error {
	return errors.New("will not convert from v3alpha1 TracingServiceSpec back to v2 TracingServiceSpec")
}

func convertTLS(inTLS *BoolOrString, inService string) (string, string) {
	outTLS := ""
	outService := inService

	if inTLS == nil {
		return outTLS, outService
	}

	if (inTLS.Bool != nil) && (*inTLS.Bool) {
		if strings.HasPrefix(inService, "http://") {
			outService = "https://" + inService[len("http://"):]
		} else if !strings.HasPrefix(inService, "https://") {
			// This looks way too general, I know, but it's correct -- if they
			// have some weird scheme that's neither http nor https, let the
			// Emissary code reject it.
			outService = "https://" + inService
		}
	} else {
		outTLS = *inTLS.String
	}

	return outTLS, outService
}
