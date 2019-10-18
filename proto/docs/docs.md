# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [AuthService.proto](#AuthService.proto)
    - [AuthService](#.AuthService)
    - [AuthService.AuthIncludeBody](#.AuthService.AuthIncludeBody)
    - [AuthService.AuthStatusOnError](#.AuthService.AuthStatusOnError)
  
    - [AuthService.AuthProtos](#.AuthService.AuthProtos)
  
  
  

- [Mapping.proto](#Mapping.proto)
    - [Mapping](#.Mapping)
    - [Mapping.CORS](#.Mapping.CORS)
    - [Mapping.CircuitBreaker](#.Mapping.CircuitBreaker)
    - [Mapping.HeadersEntry](#.Mapping.HeadersEntry)
    - [Mapping.Labels](#.Mapping.Labels)
    - [Mapping.Labels.LabelsEntry](#.Mapping.Labels.LabelsEntry)
    - [Mapping.LoadBalancer](#.Mapping.LoadBalancer)
    - [Mapping.LoadBalancer.Cookie](#.Mapping.LoadBalancer.Cookie)
    - [Mapping.RegexHeadersEntry](#.Mapping.RegexHeadersEntry)
    - [Mapping.RetryPolicy](#.Mapping.RetryPolicy)
  
    - [Mapping.CircuitBreaker.Priorities](#.Mapping.CircuitBreaker.Priorities)
    - [Mapping.LoadBalancer.Policy](#.Mapping.LoadBalancer.Policy)
  
  
  

- [Scalar Value Types](#scalar-value-types)



<a name="AuthService.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## AuthService.proto
AuthService defines an external authentication service that Ambassador
will use to check whether incoming requests should be allowed to 
continue. The AuthService is particularly powerful because it can 
modify many things about the request in flight.


<a name=".AuthService"></a>

### AuthService



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiVersion | [string](#string) |  | Common to all Ambassador objects. |
| kind | [string](#string) |  |  |
| name | [string](#string) |  |  |
| ambassador_id | [string](#string) | repeated |  |
| generation | [int32](#int32) |  | generally not used by humans! |
| auth_service | [string](#string) |  | This is the service to talk to. |
| path_prefix | [string](#string) |  | This is prefixed to the path of every request. |
| tls | [bool](#bool) |  |  |
| tls_context | [string](#string) |  |  |
| proto | [AuthService.AuthProtos](#AuthService.AuthProtos) |  |  |
| timeout_ms | [int32](#int32) |  |  |
| allowed_request_headers | [string](#string) | repeated |  |
| allowed_authorization_headers | [string](#string) | repeated |  |
| allow_request_body | [bool](#bool) |  |  |
| add_linkerd_headers | [bool](#bool) |  |  |
| failure_mode_allow | [bool](#bool) |  |  |
| include_body | [AuthService.AuthIncludeBody](#AuthService.AuthIncludeBody) |  |  |
| status_on_error | [AuthService.AuthStatusOnError](#AuthService.AuthStatusOnError) |  |  |






<a name=".AuthService.AuthIncludeBody"></a>

### AuthService.AuthIncludeBody



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| max_bytes | [int32](#int32) |  |  |
| allow_partial | [bool](#bool) |  |  |






<a name=".AuthService.AuthStatusOnError"></a>

### AuthService.AuthStatusOnError
Why isn&#39;t this just an int??


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int32](#int32) |  |  |





 


<a name=".AuthService.AuthProtos"></a>

### AuthService.AuthProtos


| Name | Number | Description |
| ---- | ------ | ----------- |
| HTTP | 0 |  |
| GRPC | 1 |  |


 

 

 



<a name="Mapping.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## Mapping.proto



<a name=".Mapping"></a>

### Mapping



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apiVersion | [string](#string) |  |  |
| kind | [string](#string) |  |  |
| name | [string](#string) |  |  |
| ambassador_id | [string](#string) | repeated |  |
| generation | [int32](#int32) |  |  |
| prefix | [string](#string) |  |  |
| prefix_regex | [bool](#bool) |  |  |
| service | [string](#string) |  |  |
| add_request_headers | [string](#string) | repeated |  |
| add_response_headers | [string](#string) | repeated |  |
| add_linkerd_headers | [bool](#bool) |  |  |
| auto_host_rewrite | [bool](#bool) |  |  |
| case_sensitive | [bool](#bool) |  |  |
| enable_ipv4 | [bool](#bool) |  |  |
| enable_ipv6 | [bool](#bool) |  |  |
| circuit_breakers | [Mapping.CircuitBreaker](#Mapping.CircuitBreaker) | repeated |  |
| cors | [Mapping.CORS](#Mapping.CORS) |  |  |
| retry_policy | [Mapping.RetryPolicy](#Mapping.RetryPolicy) |  |  |
| grpc | [bool](#bool) |  |  |
| host_redirect | [bool](#bool) |  |  |
| host_rewrite | [string](#string) |  |  |
| method | [string](#string) |  |  |
| method_regex | [bool](#bool) |  |  |
| outlier_detection | [string](#string) |  |  |
| path_redirect | [string](#string) |  |  |
| priority | [string](#string) |  |  |
| precedence | [int32](#int32) |  |  |
| remove_request_headers | [string](#string) | repeated |  |
| remove_response_headers | [string](#string) | repeated |  |
| resolver | [string](#string) |  |  |
| rewrite | [string](#string) |  |  |
| shadow | [bool](#bool) |  |  |
| connect_timeout_ms | [int32](#int32) |  |  |
| cluster_idle_timeout_ms | [int32](#int32) |  |  |
| timeout_ms | [int32](#int32) |  |  |
| idle_timeout_ms | [int32](#int32) |  |  |
| tls_context | [string](#string) |  |  |
| tls | [bool](#bool) |  |  |
| use_websocket | [bool](#bool) |  |  |
| weight | [int32](#int32) |  |  |
| bypass_auth | [bool](#bool) |  |  |
| host | [string](#string) |  |  |
| host_regex | [bool](#bool) |  |  |
| headers | [Mapping.HeadersEntry](#Mapping.HeadersEntry) | repeated |  |
| regex_headers | [Mapping.RegexHeadersEntry](#Mapping.RegexHeadersEntry) | repeated |  |
| labels | [Mapping.Labels](#Mapping.Labels) |  |  |
| load_balancer | [Mapping.LoadBalancer](#Mapping.LoadBalancer) |  |  |






<a name=".Mapping.CORS"></a>

### Mapping.CORS



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| origins | [string](#string) | repeated |  |
| methods | [string](#string) | repeated |  |
| headers | [string](#string) | repeated |  |
| exposed_headers | [string](#string) | repeated |  |
| credentials | [bool](#bool) |  |  |
| max_age | [string](#string) |  |  |






<a name=".Mapping.CircuitBreaker"></a>

### Mapping.CircuitBreaker



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| priority | [Mapping.CircuitBreaker.Priorities](#Mapping.CircuitBreaker.Priorities) |  |  |
| max_connections | [int32](#int32) |  |  |
| max_pending_requests | [int32](#int32) |  |  |
| max_requests | [int32](#int32) |  |  |
| max_retries | [int32](#int32) |  |  |






<a name=".Mapping.HeadersEntry"></a>

### Mapping.HeadersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name=".Mapping.Labels"></a>

### Mapping.Labels



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [Mapping.Labels.LabelsEntry](#Mapping.Labels.LabelsEntry) | repeated |  |






<a name=".Mapping.Labels.LabelsEntry"></a>

### Mapping.Labels.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [google.protobuf.Any](#google.protobuf.Any) |  |  |






<a name=".Mapping.LoadBalancer"></a>

### Mapping.LoadBalancer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [Mapping.LoadBalancer.Policy](#Mapping.LoadBalancer.Policy) |  |  |
| cookie | [Mapping.LoadBalancer.Cookie](#Mapping.LoadBalancer.Cookie) |  |  |
| header | [string](#string) |  |  |
| source_ip | [bool](#bool) |  |  |






<a name=".Mapping.LoadBalancer.Cookie"></a>

### Mapping.LoadBalancer.Cookie



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| path | [string](#string) |  |  |
| ttl | [string](#string) |  |  |






<a name=".Mapping.RegexHeadersEntry"></a>

### Mapping.RegexHeadersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name=".Mapping.RetryPolicy"></a>

### Mapping.RetryPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| retry_on | [string](#string) |  | Legal values: 5xx, gateway-error, connect-failure, retriable-4xx, refused-stream, retriable-status-codes

Technically this could be an enum, but... ew. |
| num_retries | [int32](#int32) |  |  |
| per_try_timeout | [string](#string) |  |  |





 


<a name=".Mapping.CircuitBreaker.Priorities"></a>

### Mapping.CircuitBreaker.Priorities


| Name | Number | Description |
| ---- | ------ | ----------- |
| default | 0 |  |
| high | 1 |  |



<a name=".Mapping.LoadBalancer.Policy"></a>

### Mapping.LoadBalancer.Policy


| Name | Number | Description |
| ---- | ------ | ----------- |
| round_robin | 0 |  |
| ring_hash | 1 |  |
| maglev | 2 |  |
| least_request | 3 |  |


 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

