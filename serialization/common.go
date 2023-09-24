package serialization

import (
	"encoding/hex"
	"log"
	"reflect"

	"github.com/Salvionied/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

func Blake2bHash(data []byte) []byte {
	hash, err := blake2b.New(32, nil)
	if err != nil {
		log.Fatal(err)
	}
	_, err = hash.Write(data)
	if err != nil {
		log.Fatal(err)
	}
	return hash.Sum(nil)
}

type ConstrainedBytes struct {
	Payload []byte
}

func (cb *ConstrainedBytes) UnmarshalCBOR(data []byte) error {
	err := cbor.Unmarshal(data, &cb.Payload)
	return err
}

func (cb *ConstrainedBytes) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(cb.Payload)
}

const VERIFICATION_KEY_HASH_SIZE = 28

type TransactionId ConstrainedBytes
type ScriptHash [28]byte

func (sh *ScriptHash) Bytes() []byte {
	return sh[:]
}

type DatumHash ConstrainedBytes

func (dh *DatumHash) Equal(other DatumHash) bool {
	return reflect.DeepEqual(dh.Payload, other.Payload)
}

type ScriptDataHash ConstrainedBytes

type PubKeyHash [28]byte
type CustomBytes struct {
	Value string
	tp    string
}

func (cb CustomBytes) String() string {
	return cb.Value
}

func (cb *CustomBytes) MarshalCBOR() ([]byte, error) {
	// if cb.Value == "40" || cb.Value == "625b5d" {
	// 	return cbor.Marshal(make([]byte, 0))
	// }
	if cb.tp == "string" {
		if cb.Value == "[]" {
			return cbor.Marshal(make([]byte, 0))
		}
		return cbor.Marshal(cb.Value)
	}
	res, err := hex.DecodeString(cb.Value)
	if err != nil {
		return nil, err
	}
	// if len(res) == 0 {
	// 	return cbor.Marshal(make([]byte, 0))
	// }
	return cbor.Marshal(res)

}
func (cb *CustomBytes) UnmarshalCBOR(value []byte) error {
	var res any
	err := cbor.Unmarshal(value, &res)
	if err != nil {
		log.Fatal(err)
		return err
	}
	switch res.(type) {
	case []byte:
		cb.tp = "bytes"
		cb.Value = hex.EncodeToString(res.([]byte))
	case string:
		cb.tp = "string"
		cb.Value = res.(string)
	default:
		log.Fatal("Unknown type")
	}
	return nil
}
