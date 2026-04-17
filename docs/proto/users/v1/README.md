# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [users/v1/types.proto](#users_v1_types-proto)
    - [Gender](#users-v1-Gender)

- [users/v1/events.proto](#users_v1_events-proto)
    - [UserHumanAdded](#users-v1-UserHumanAdded)
    - [UserHumanEmailChanged](#users-v1-UserHumanEmailChanged)
    - [UserHumanEmailVerified](#users-v1-UserHumanEmailVerified)
    - [UserHumanProfileChanged](#users-v1-UserHumanProfileChanged)

- [Scalar Value Types](#scalar-value-types)



<a name="users_v1_types-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## users/v1/types.proto





<a name="users-v1-Gender"></a>

### Gender
Gender mirrors zitadel/zitadel internal/domain.Gender.

| Name | Number | Description |
| ---- | ------ | ----------- |
| GENDER_UNSPECIFIED | 0 |  |
| GENDER_FEMALE | 1 |  |
| GENDER_MALE | 2 |  |
| GENDER_DIVERSE | 3 |  |










<a name="users_v1_events-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## users/v1/events.proto



<a name="users-v1-UserHumanAdded"></a>

### UserHumanAdded
UserHumanAdded mirrors zitadel/zitadel
internal/repository/user.HumanAddedEvent.
Emitted for Zitadel event type &#34;user.human.added&#34;.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_name | [string](#string) |  |  |
| first_name | [string](#string) |  |  |
| last_name | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| email | [string](#string) |  |  |
| preferred_language | [string](#string) |  |  |
| gender | [Gender](#users-v1-Gender) |  |  |
| nick_name | [string](#string) | optional |  |






<a name="users-v1-UserHumanEmailChanged"></a>

### UserHumanEmailChanged
UserHumanEmailChanged mirrors HumanEmailChangedEvent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |






<a name="users-v1-UserHumanEmailVerified"></a>

### UserHumanEmailVerified
UserHumanEmailVerified mirrors HumanEmailVerifiedEvent. Marker event
— Zitadel sends no event_payload on the wire.






<a name="users-v1-UserHumanProfileChanged"></a>

### UserHumanProfileChanged
UserHumanProfileChanged mirrors HumanProfileChangedEvent.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| first_name | [string](#string) |  |  |
| last_name | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| preferred_language | [string](#string) |  |  |
| gender | [Gender](#users-v1-Gender) |  |  |
| nick_name | [string](#string) | optional |  |
| updated_fields | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |















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
