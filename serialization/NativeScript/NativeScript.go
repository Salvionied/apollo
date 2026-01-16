package NativeScript

import (
	"errors"

	"github.com/Salvionied/apollo/v2/serialization"

	"github.com/blinklabs-io/gouroboros/cbor"
	"golang.org/x/crypto/blake2b"
)

type ScriptTag int

const (
	ScriptPubKey ScriptTag = iota
	ScriptAll
	ScriptAny
	ScriptNofK
	InvalidBefore
	InvalidHereafter
)

func NewScriptPubKey(keyHash []byte) NativeScript {
	return NativeScript{Tag: ScriptPubKey, KeyHash: keyHash}
}

func NewScriptAll(nativeScripts []NativeScript) NativeScript {
	return NativeScript{Tag: ScriptAll, NativeScripts: nativeScripts}
}

func NewScriptAny(nativeScripts []NativeScript) NativeScript {
	return NativeScript{Tag: ScriptAny, NativeScripts: nativeScripts}
}

func NewScriptNofK(nativeScripts []NativeScript, noK int) NativeScript {
	return NativeScript{Tag: ScriptNofK, NativeScripts: nativeScripts, NoK: noK}
}

func NewInvalidBefore(before int64) NativeScript {
	return NativeScript{Tag: InvalidBefore, Before: before}
}

func NewInvalidHereafter(after int64) NativeScript {
	return NativeScript{Tag: InvalidHereafter, After: after}
}

type NativeScript struct {
	Tag           ScriptTag
	KeyHash       []byte
	NativeScripts []NativeScript
	NoK           int
	Before        int64
	After         int64
}

type SerialScripts struct {
	cbor.StructAsArray
	Tag           ScriptTag
	NativeScripts []NativeScript
}

type SerialNok struct {
	Tag           ScriptTag
	NoK           int
	NativeScripts []NativeScript
}

/*
*

	MarshalCBOR serializes the SerialNok into a CBOR-encoded byte slice.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if serialization fails.
*/
func (s *SerialNok) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{s.Tag, s.NoK, s.NativeScripts})
}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into a SerialNok.

	Params:
		value ([]byte): The CBOR-encoded data to be deserialized.

	Returns:
		error: An error if deserialization fails.
*/
func (s *SerialNok) UnmarshalCBOR(value []byte) error {
	var arr []any
	_, err := cbor.Decode(value, &arr)
	if err != nil {
		return err
	}
	if arr == nil {
		return errors.New("cbor.Decode returned nil arr")
	}
	s.Tag = ScriptTag(arr[0].(uint64))
	s.NoK = int(arr[1].(uint64))
	scriptsArr := arr[2].([]any)
	s.NativeScripts = make([]NativeScript, len(scriptsArr))
	for i, v := range scriptsArr {
		scriptArr := v.([]any)
		scriptBytes, _ := cbor.Encode(scriptArr)
		if err := s.NativeScripts[i].UnmarshalCBOR(scriptBytes); err != nil {
			return err
		}
	}
	return nil
}

type SerialInt struct {
	Tag   ScriptTag
	Value int64
}

/*
*

	MarshalCBOR serializes the SerialInt into a CBOR-encoded byte slice.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if serialization fails.
*/
func (s *SerialInt) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{s.Tag, s.Value})
}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into a SerialInt.

	Params:
		value ([]byte): The CBOR-encoded data to be deserialized.

	Returns:
		error: An error if deserialization fails.
*/
func (s *SerialInt) UnmarshalCBOR(value []byte) error {
	var arr []any
	_, err := cbor.Decode(value, &arr)
	if err != nil {
		return err
	}
	if arr == nil {
		return errors.New("cbor.Decode returned nil arr")
	}
	s.Tag = ScriptTag(arr[0].(uint64))
	s.Value = int64(arr[1].(uint64))
	return nil
}

type SerialHash struct {
	Tag   ScriptTag
	Value []byte
}

/*
*

	MarshalCBOR serializes the SerialHash into a CBOR-encoded byte slice.

	Returns:
		[]byte: The CBOR-encoded byte slice.
		error: An error if serialization fails.
*/
func (s *SerialHash) MarshalCBOR() ([]byte, error) {
	return cbor.Encode([]any{s.Tag, s.Value})
}

/*
*

	UnmarshalCBOR deserializes a CBOR-encoded byte slice into a SerialHash.

	Params:
		value ([]byte): The CBOR-encoded data to be deserialized.

	Returns:
		error: An error if deserialization fails.
*/
func (s *SerialHash) UnmarshalCBOR(value []byte) error {
	var arr []any
	_, err := cbor.Decode(value, &arr)
	if err != nil {
		return err
	}
	if arr == nil {
		return errors.New("cbor.Decode returned nil arr")
	}
	s.Tag = ScriptTag(arr[0].(uint64))
	s.Value = arr[1].([]byte)
	return nil
}

/*
*

	Hash computes the script hash for the NativeScript.

	Returns:
		serialization.ScriptHash: The computed script hash.
		error: An error if the hashing fails.
*/
func (ns NativeScript) Hash() (serialization.ScriptHash, error) {
	bytes, err := cbor.Encode(ns)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	finalbytes := make([]byte, 0, 1+len(bytes))
	finalbytes = append(finalbytes, 0)
	finalbytes = append(finalbytes, bytes...)
	hash, err := blake2b.New(28, nil)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	_, err = hash.Write(finalbytes)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
	ret := serialization.ScriptHash{}
	copy(ret[:], hash.Sum(nil))
	return ret, nil
}

/*
*

	UnmarshalCBOR decodes the CBOR-encoded data and populates
	the NativeScript fields.

	Params:
		value []byte: The CBOR-encoded data to encode.

	Returns:
		error: An error if decoding fails, nil otherwise.
*/
func (ns *NativeScript) UnmarshalCBOR(value []byte) error {
	var tmp any
	_, err := cbor.Decode(value, &tmp)
	if err != nil {
		return err
	}
	tmpSlice, ok := tmp.([]any)
	if !ok {
		return errors.New("invalid CBOR structure")
	}
	tag, _ := tmpSlice[0].(uint64)
	switch int(tag) {
	case 0:
		tmp := new(SerialHash)
		_, err := cbor.Decode(value, &tmp)
		ns.KeyHash = tmp.Value
		ns.Tag = tmp.Tag
		return err
	case 1:
		tmp := new(SerialScripts)
		_, err := cbor.Decode(value, &tmp)
		ns.Tag = tmp.Tag
		ns.NativeScripts = tmp.NativeScripts
		return err
	case 2:
		tmp := new(SerialScripts)
		_, err := cbor.Decode(value, &tmp)
		ns.Tag = tmp.Tag
		ns.NativeScripts = tmp.NativeScripts
		return err
	case 3:
		tmp := new(SerialNok)
		_, err := cbor.Decode(value, &tmp)
		ns.NativeScripts = tmp.NativeScripts
		ns.Tag = tmp.Tag
		ns.NoK = tmp.NoK
		return err
	case 4:
		tmp := new(SerialInt)
		_, err := cbor.Decode(value, &tmp)

		ns.Tag = tmp.Tag
		ns.Before = tmp.Value
		return err
	case 5:
		tmp := new(SerialInt)
		_, err := cbor.Decode(value, &tmp)
		ns.Tag = tmp.Tag
		ns.After = tmp.Value
		return err
	default:
		return nil
	}
}

/*
*

	MarshalCBOR encodes the NativeScript into CBOR format.

	Returns:
		[]uint8: The CBOR-encoded data.
		error: An error if encoding fails, nil otherwise.
*/
func (ns *NativeScript) MarshalCBOR() ([]uint8, error) {
	switch ns.Tag {
	case 0:
		return cbor.Encode(SerialHash{Tag: ns.Tag, Value: ns.KeyHash})
	case 1:
		return cbor.Encode(
			SerialScripts{Tag: ns.Tag, NativeScripts: ns.NativeScripts},
		)
	case 2:
		return cbor.Encode(
			SerialScripts{Tag: ns.Tag, NativeScripts: ns.NativeScripts},
		)
	case 3:
		return cbor.Encode(
			SerialNok{
				Tag:           ns.Tag,
				NoK:           ns.NoK,
				NativeScripts: ns.NativeScripts,
			},
		)
	case 4:
		return cbor.Encode(SerialInt{Tag: ns.Tag, Value: ns.Before})
	case 5:
		return cbor.Encode(SerialInt{Tag: ns.Tag, Value: ns.After})

	default:
		return make([]uint8, 0), nil
	}
}
