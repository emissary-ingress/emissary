
- [getambassador.io/v2/Host.proto](#getambassador.io/v2/Host.proto)
    - [ACMEProviderSpec](#ACMEProviderSpec)
    - [HostSpec](#HostSpec)
    - [HostStatus](#HostStatus)
    - [InsecureRequestPolicy](#InsecureRequestPolicy)
    - [RequestPolicy](#RequestPolicy)
  
    - [HostPhase](#HostPhase)
    - [HostState](#HostState)
    - [HostTLSCertificateSource](#HostTLSCertificateSource)
    - [InsecureRequestAction](#InsecureRequestAction)
  
- [Scalar Value Types](#scalar-value-types)



<a name="getambassador.io/v2/Host.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## getambassador.io/v2/Host.proto


## Attributes


<a name="ACMEProviderSpec"></a>

### ACMEProviderSpec
The acmeProvider element in a Host defines how Ambassador should handle TLS certificates:

```yaml
acmeProvider:
authority: url-to-provider
email: email-of-registrant
tlsSecret:
name: secret-name
```

In general, `email-of-registrant` is mandatory when using ACME: it should be a valid email address that will reach someone responsible for certificate management.
ACME stores certificates in Kubernetes secrets. The name of the secret can be set using the `tlsSecret` element; if not supplied, a name will be automatically generated from the `hostname` and `email`.
If the authority is not supplied, the Let’s Encrypt production environment is assumed.
**If the authority is the literal string “none”, TLS certificate management will be disabled.** You’ll need to manually create a TLSContext to use for your host in order to use HTTPS.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authority | [string](#string) |  | Specifies who to talk ACME with to get certs. Defaults to Let's Encrypt; if "none", do not try to do TLS for this Host. |
| email | [string](#string) |  |  |
| privateKeySecret | [k8s.io.api.core.v1.LocalObjectReference](#k8s.io.api.core.v1.LocalObjectReference) |  |  |
| registration | [string](#string) |  | This is normally set automatically |






<a name="HostSpec"></a>

### HostSpec
The custom `Host` resource defines how the Ambassador Edge Stack will be visible to the outside world. It collects all the following information in a single configuration resource:

The hostname by which Ambassador will be reachable
How Ambassador should handle TLS certificates
How Ambassador should handle secure and insecure requests
Which resources to examine for further configuration

A minimal Host resource, using Let’s Encrypt to handle TLS, would be:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
name: minimal-host
spec:
hostname: host.example.com
acmeProvider:
email: julian@example.com
```

This Host tells Ambassador to expect to be reached at `host.example.com`, and to manage TLS certificates using Let’s Encrypt, registering as `julian@example.com`. Since it doesn’t specify otherwise, requests using cleartext will be automatically redirected to use HTTPS, and Ambassador will not search for any specific further configuration resources related to this Host.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ambassador_id | [string](#string) | repeated | Common to all Ambassador objects (and optional). |
| generation | [int32](#int32) |  | Common to all Ambassador objects (and optional). |
| hostname | [string](#string) |  | Hostname by which the Ambassador can be reached. |
| selector | [k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector](#k8s.io.apimachinery.pkg.apis.meta.v1.LabelSelector) |  | Selector by which we can find further configuration. Defaults to hostname=$hostname |
| acmeProvider | [ACMEProviderSpec](#getambassador.io.v2.ACMEProviderSpec) |  | Specifies who to talk ACME with to get certs. Defaults to Let's Encrypt; if "none", do not try to do TLS for this Host.

The acmeProvider element in a Host defines how Ambassador should handle TLS certificates |
| tlsSecret | [k8s.io.api.core.v1.LocalObjectReference](#k8s.io.api.core.v1.LocalObjectReference) |  | Name of the Kubernetes secret into which to save generated certificates. Defaults to $hostname |
| requestPolicy | [RequestPolicy](#getambassador.io.v2.RequestPolicy) |  | Request policy definition. |






<a name="HostStatus"></a>

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






<a name="InsecureRequestPolicy"></a>

### InsecureRequestPolicy
The `insecure-action` can be one of:

`Redirect` (the default): redirect to HTTPS
`Route`: go ahead and route as normal; this will allow handling HTTP requests normally
`Reject`: reject the request with a 400 response

The `additionalPort` element tells Ambassador to listen on the specified `insecure-port` and treat any request arriving on that port as insecure. **By default, `additionalPort` will be set to 8080 for any `Host` using TLS.** To disable this redirection entirely, set `additionalPort` explicitly to `-1`:

```yaml
requestPolicy:
insecure:
additionalPort: -1   # This is how to disable the default redirection from 8080.
```

Some special cases to be aware of here:

**Case matters in the actions:** you must use e.g. `Reject`, not `reject`.
The `X-Forwarded-Proto` header is honored when determining whether a request is secure or insecure. For more information, see "Load Balancers, the `Host` Resource, and `X-Forwarded-Proto`" below.
ACME challenges with prefix `/.well-known/acme-challenge/` are always forced to be considered insecure, since they are not supposed to arrive over HTTPS.
Ambassador Edge Stack provides native handling of ACME challenges. If you are using this support, Ambassador will automatically arrange for insecure ACME challenges to be handled correctly. If you are handling ACME yourself - as you must when running Ambassador Open Source - you will need to supply appropriate Host resources and Mappings to correctly direct ACME challenges to your ACME challenge handler.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [InsecureRequestAction](#getambassador.io.v2.InsecureRequestAction) |  | What action should be taken for an insecure request? |
| additionalPort | [int32](#int32) |  | Is there an additional insecure port we should listen on? |






<a name="RequestPolicy"></a>

### RequestPolicy
A **secure** request arrives via HTTPS; an **insecure** request does not. By default, secure requests will be routed and insecure requests will be redirected (using an HTTP 301 response) to HTTPS. The behavior of insecure requests can be overridden using the `requestPolicy` element of a Host:

```yaml
requestPolicy:
insecure:
action: insecure-action
additionalPort: insecure-port
```


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| insecure | [InsecureRequestPolicy](#getambassador.io.v2.InsecureRequestPolicy) |  | How shall we handle insecure requests? |








<a name="HostPhase"></a>

### HostPhase


| Name | Number | Description |
| ---- | ------ | ----------- |
| NA | 0 |  |
| DefaultsFilled | 1 |  |
| ACMEUserPrivateKeyCreated | 2 |  |
| ACMEUserRegistered | 3 |  |
| ACMECertificateChallenge | 4 |  |



<a name="HostState"></a>

### HostState


| Name | Number | Description |
| ---- | ------ | ----------- |
| Initial | 0 | The default value is the "zero" value, and it would be great if "Pending" could be the default value; but it's Important that the "zero" value be able to be shown as empty/omitted from display, and we really do want `kubectl get hosts` to say "Pending" in the "STATE" column, and not leave the column empty. |
| Pending | 1 |  |
| Ready | 2 |  |
| Error | 3 |  |



<a name="HostTLSCertificateSource"></a>

### HostTLSCertificateSource


| Name | Number | Description |
| ---- | ------ | ----------- |
| Unknown | 0 |  |
| None | 1 |  |
| Other | 2 |  |
| ACME | 3 |  |



<a name="InsecureRequestAction"></a>

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

