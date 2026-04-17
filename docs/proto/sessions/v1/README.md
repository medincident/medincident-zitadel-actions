# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [sessions/v1/types.proto](#sessions_v1_types-proto)
    - [HeaderValues](#sessions-v1-HeaderValues)
    - [UserAgent](#sessions-v1-UserAgent)
    - [UserAgent.HeadersEntry](#sessions-v1-UserAgent-HeadersEntry)

- [sessions/v1/events.proto](#sessions_v1_events-proto)
    - [SessionAdded](#sessions-v1-SessionAdded)
    - [SessionUserChecked](#sessions-v1-SessionUserChecked)

- [Scalar Value Types](#scalar-value-types)



<a name="sessions_v1_types-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## sessions/v1/types.proto



<a name="sessions-v1-HeaderValues"></a>

### HeaderValues
HeaderValues wraps the repeated-value half of a multi-valued HTTP
header. Proto3 maps cannot have a repeated value type directly, so
we wrap.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| values | [string](#string) | repeated |  |






<a name="sessions-v1-UserAgent"></a>

### UserAgent
UserAgent mirrors zitadel/zitadel internal/domain.UserAgent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ip | [string](#string) |  |  |
| headers | [UserAgent.HeadersEntry](#sessions-v1-UserAgent-HeadersEntry) | repeated |  |
| fingerprint_id | [string](#string) | optional |  |
| description | [string](#string) | optional |  |






<a name="sessions-v1-UserAgent-HeadersEntry"></a>

### UserAgent.HeadersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [HeaderValues](#sessions-v1-HeaderValues) |  |  |















<a name="sessions_v1_events-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## sessions/v1/events.proto



<a name="sessions-v1-SessionAdded"></a>

### SessionAdded
SessionAdded mirrors zitadel/zitadel
internal/repository/session.AddedEvent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_agent | [UserAgent](#sessions-v1-UserAgent) | optional |  |






<a name="sessions-v1-SessionUserChecked"></a>

### SessionUserChecked
SessionUserChecked mirrors
internal/repository/session.UserCheckedEvent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |
| user_resource_owner | [string](#string) |  |  |
| checked_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| preferred_language | [string](#string) | optional |  |















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
