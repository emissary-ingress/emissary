# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [Host.proto](#Host.proto)
    - [ACMEProviderSpec](#getambassador.io.v2.ACMEProviderSpec)
    - [HostSpec](#getambassador.io.v2.HostSpec)
    - [HostStatus](#getambassador.io.v2.HostStatus)
    - [InsecureRequestPolicy](#getambassador.io.v2.InsecureRequestPolicy)
    - [RequestPolicy](#getambassador.io.v2.RequestPolicy)
  
    - [HostPhase](#getambassador.io.v2.HostPhase)
    - [HostState](#getambassador.io.v2.HostState)
    - [HostTLSCertificateSource](#getambassador.io.v2.HostTLSCertificateSource)
    - [InsecureRequestAction](#getambassador.io.v2.InsecureRequestAction)
  
- [Scalar Value Types](#scalar-value-types)



<a name="Host.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## Host.proto
=== HACK HACK HACK ===

The existence of Host_nojson.proto is because if we bring in all the
k8s.io types that we would want to, the generated *.pb.json.go files try
to import github.com/gogo/protobuf/jsonpb, and that turns out to crash on
some of the k8s.io types. Sigh.

So, instead, we split out the minimal high-level stuff we need in most of
of the Go code into Host_nojson.proto, and leave the more detailed things
with the breaking types here. For more information on this brutality, see
https://github.com/datawire/ambassador/pull/1999#issuecomment-548939518.

=== end hack ===

Host defines a way that an Ambassador will be visible to the
outside world. A minimal Host defines a hostname (of course) by
which the Ambassador will be reachable, but a Host can also
tell an Ambassador how to manage TLS, and which resources to 
examine for further configuration.


<a name="getambassador.io.v2.ACMEProviderSpec"></a>

### ACMEProviderSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | [string](#string) |  | Specifies who to talk ACME with to get certs. Defaults to Let's Encrypt; if "none", do not try to do TLS for this Host. |
| email | [string](#string) |  |  |
| privateKeySecret | [k8s.io.api.core.v1.LocalObjectReference](#k8s.io.api.core.v1.LocalObjectReference) |  |  |
| registration | [string](#string) |  | This is normally set automatically |






<a name="getambassador.io.v2.HostSpec"></a>

### HostSpec
The Host resource will usually be a Kubernetes CRD, but it could
appear in other forms. The HostSpec is the part of the Host resource
that doesn't change, no matter what form it's in -- when it's a CRD,
this is the part in the "spec" dictionary.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ambassador_id | [string](#string) | repeated | Common to all Ambassador objects (and optional). |
| generation | [int32](#int32) |  | Common to all Ambassador objects (and optional). |
| hostname | [string](#string) |  | Hostname by which the Ambassador can be reached. |
| selector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector) |  | Selector by which we can find further configuration. Defaults to hostname=$hostname |
| acmeProvider | [ACMEProviderSpec](#getambassador.io.v2.ACMEProviderSpec) |  | Specifies who to talk ACME with to get certs. Defaults to Let's Encrypt; if "none", do not try to do TLS for this Host. |
| tlsSecret | [k8s.io.api.core.v1.LocalObjectReference](#k8s.io.api.core.v1.LocalObjectReference) |  | Name of the Kubernetes secret into which to save generated certificates. Defaults to $hostname |
| requestPolicy | [RequestPolicy](#getambassador.io.v2.RequestPolicy) |  | Request policy definition. |






<a name="getambassador.io.v2.HostStatus"></a>

### HostStatus



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tlsCertificateSource | [HostTLSCertificateSource](#getambassador.io.v2.HostTLSCertificateSource) |  |  |
| state | [HostState](#getambassador.io.v2.HostState) |  |  |
| phaseCompleted | [HostPhase](#getambassador.io.v2.HostPhase) |  | phaseCompleted and phasePending are valid when state==Pending or state==Error. |
| phasePending | [HostPhase](#getambassador.io.v2.HostPhase) |  |  |
| errorReason | [string](#string) |  | errorReason, errorTimestamp, and errorBackoff are valid when state==Error. |
| errorTimestamp | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| errorBackoff | [google.protobuf.Duration](#google.protobuf.Duration) |  |  |






<a name="getambassador.io.v2.InsecureRequestPolicy"></a>

### InsecureRequestPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [InsecureRequestAction](#getambassador.io.v2.InsecureRequestAction) |  | What action should be taken for an insecure request? |
| additionalPort | [int32](#int32) |  | Is there an additional insecure port we should listen on? |






<a name="getambassador.io.v2.RequestPolicy"></a>

### RequestPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| insecure | [InsecureRequestPolicy](#getambassador.io.v2.InsecureRequestPolicy) |  | How shall we handle insecure requests? |








<a name="getambassador.io.v2.HostPhase"></a>

### HostPhase


| Name | Number | Description |
| ---- | ------ | ----------- |
| NA | 0 |  |
| DefaultsFilled | 1 |  |
| ACMEUserPrivateKeyCreated | 2 |  |
| ACMEUserRegistered | 3 |  |
| ACMECertificateChallenge | 4 |  |



<a name="getambassador.io.v2.HostState"></a>

### HostState


| Name | Number | Description |
| ---- | ------ | ----------- |
| Initial | 0 | The default value is the "zero" value, and it would be great if "Pending" could be the default value; but it's Important that the "zero" value be able to be shown as empty/omitted from display, and we really do want `kubectl get hosts` to say "Pending" in the "STATE" column, and not leave the column empty. |
| Pending | 1 |  |
| Ready | 2 |  |
| Error | 3 |  |



<a name="getambassador.io.v2.HostTLSCertificateSource"></a>

### HostTLSCertificateSource


| Name | Number | Description |
| ---- | ------ | ----------- |
| Unknown | 0 |  |
| None | 1 |  |
| Other | 2 |  |
| ACME | 3 |  |



<a name="getambassador.io.v2.InsecureRequestAction"></a>

### InsecureRequestAction


| Name | Number | Description |
| ---- | ------ | ----------- |
| Redirect | 0 |  |
| Reject | 1 |  |
| Route | 2 |  |










## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

