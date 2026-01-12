package serialization

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/blinklabs-io/gouroboros/cbor"
	"golang.org/x/crypto/blake2b"
)

/*
*
Blake2bHash computes the Blake2b hash of the given data.

Params:

	data ([]byte): The data to hash.

Returns:

	[]byte: The Blake2b hash of the data.
	error: An error if the hashing fails.
*/
func Blake2bHash(data []byte) ([]byte, error) {
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil, err
	}
	_, err = hash.Write(data)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

type ConstrainedBytes struct {
	Payload []byte
}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into ConstrainedBytes.

	Params:
		data ([]byte): The CBOR-encoded byte slice.

	Returns:
		error: An error if deserialization fails.
*/
func (cb *ConstrainedBytes) UnmarshalCBOR(data []byte) error {
	_, err := cbor.Decode(data, &cb.Payload)
	return err
}

/*
*

	MarshalCBOR serializes ConstrainedBytes into a CBOR-encoded byte slice.

	Returns:
		[]byte: A CBOR-encoded byte slice representing the ConstrainedBytes.
		error: An error if serialization fails.
*/
func (cb *ConstrainedBytes) MarshalCBOR() ([]byte, error) {
	return cbor.Encode(cb.Payload)
}

const VERIFICATION_KEY_HASH_SIZE = 28

type TransactionId ConstrainedBytes
type ScriptHash [28]byte

/*
*

	Bytes returns the underlying byte slice of a ScriptHash.

	Returns:
		[]byte: The byte slice representation of the ScriptHash.
*/
func (sh *ScriptHash) Bytes() []byte {
	return sh[:]
}

type DatumHash ConstrainedBytes

/*
*

	Equal checks if two DatumHash instances are equal.

	Params:
		other (DatumHash): The other DatumHash to compare to.

	Returns:
		bool: True if the DatumHashes are equal, false otherwise.
*/
func (dh *DatumHash) Equal(other DatumHash) bool {
	return reflect.DeepEqual(dh.Payload, other.Payload)
}

type ScriptDataHash ConstrainedBytes

type PubKeyHash [28]byte
type CustomBytes struct {
	Value string
	tp    string
}

func (cb *CustomBytes) Int() (int, error) {
	if cb.tp == "uint64" {
		value, err := strconv.ParseInt(cb.Value, 10, 64)
		if err != nil {
			return 0, err
		}
		if value < math.MinInt || value > math.MaxInt {
			return 0, errors.New("value out of int range")
		}
		return int(value), nil
	}
	return 0, errors.New("not int")
}

func (cb *CustomBytes) IsInt() bool {
	return cb.tp == "uint64"
}

func NewCustomBytes(value string) CustomBytes {

	return CustomBytes{Value: hex.EncodeToString([]byte(value))}
}

func NewCustomBytesInt(value int) CustomBytes {
	return CustomBytes{Value: strconv.Itoa(value), tp: "uint64"}
}

func NewCustomBytesString(value string) CustomBytes {
	return CustomBytes{Value: value, tp: "string"}
}

/*
*

	String returns the Value's string representation of a CustomBytes.

	Returns:
		string: The Value's string representation of CustomBytes.
*/
func (cb *CustomBytes) String() string {
	if cb.tp == "uint64" {
		return fmt.Sprintf("Int(%s)", cb.Value)
	}
	return fmt.Sprintf("Bytes(%s)", cb.Value)
}
func (cb *CustomBytes) HexString() string {
	return cb.Value
}

/*
*

	MarshalCBOR serializes CustomBytes into a CBOR-encoded byte slice.

	Returns:
		[]byte: A CBOR-encoded byte slice representing the CustomBytes.
		error: An error if serialization fails.
*/
func (cb *CustomBytes) MarshalCBOR() ([]byte, error) {
	// if cb.Value == "40" || cb.Value == "625b5d" {
	// 	return cbor.Encode(make([]byte, 0))
	// }
	if cb.tp == "string" {
		if cb.Value == "[]" {
			return cbor.Encode(make([]byte, 0))
		}
		return cbor.Encode([]byte(cb.Value))
	}
	if cb.tp == "uint64" {
		value, err := strconv.ParseInt(cb.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		return cbor.Encode(value)
	}
	res, err := hex.DecodeString(cb.Value)
	if err != nil {
		return nil, err
	}
	// if len(res) == 0 {
	// 	return cbor.Encode(make([]byte, 0))
	// }
	return cbor.Encode(res)

}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into CustomBytes.

	Params:
		value ([]byte): The CBOR-encoded byte slice.

	Returns:
		error: An error if deserialization fails.
*/
func (cb *CustomBytes) UnmarshalCBOR(value []byte) error {
	var res any
	_, err := cbor.Decode(value, &res)
	if err != nil {
		return err
	}
	switch res := res.(type) {
	case []byte:
		cb.tp = "bytes"
		cb.Value = hex.EncodeToString(res)
	case string:
		cb.tp = "string"
		cb.Value = res
	case uint64:
		cb.tp = "uint64"
		cb.Value = strconv.FormatUint(res, 10)
	default:
		return errors.New("invalid type for customBytes")
	}
	return nil
}
