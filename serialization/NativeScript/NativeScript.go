package NativeScript

import (
	"github.com/Salvionied/apollo/serialization"

	"github.com/Salvionied/cbor/v2"
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
	_             struct{} `cbor:",toarray"`
	Tag           ScriptTag
	NativeScripts []NativeScript
}

type SerialNok struct {
	_             struct{} `cbor:",toarray"`
	Tag           ScriptTag
	NoK           int
	NativeScripts []NativeScript
}

type SerialInt struct {
	_     struct{} `cbor:",toarray"`
	Tag   ScriptTag
	Value int64
}
type SerialHash struct {
	_     struct{} `cbor:",toarray"`
	Tag   ScriptTag
	Value []byte
}

/*
*

	Hash computes the script hash for the NativeScript.

	Returns:
		serialization.ScriptHash: The computed script hash.
		error: An error if the hashing fails.
*/
func (ns NativeScript) Hash() (serialization.ScriptHash, error) {
	finalbytes := []byte{0}
	bytes, err := cbor.Marshal(ns)
	if err != nil {
		return serialization.ScriptHash{}, err
	}
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
	var tmp = make([]any, 0)
	err := cbor.Unmarshal(value, &tmp)
	if err != nil {
		return err
	}
	ok, _ := tmp[0].(uint64)
	switch int(ok) {
	case 0:
		tmp := new(SerialHash)
		err := cbor.Unmarshal(value, &tmp)
		ns.KeyHash = tmp.Value
		ns.Tag = tmp.Tag
		return err
	case 1:
		tmp := new(SerialScripts)
		err := cbor.Unmarshal(value, &tmp)
		ns.Tag = tmp.Tag
		ns.NativeScripts = tmp.NativeScripts
		return err
	case 2:
		tmp := new(SerialScripts)
		err := cbor.Unmarshal(value, &tmp)
		ns.Tag = tmp.Tag
		ns.NativeScripts = tmp.NativeScripts
		return err
	case 3:
		tmp := new(SerialNok)
		err := cbor.Unmarshal(value, &tmp)
		ns.NativeScripts = tmp.NativeScripts
		ns.Tag = tmp.Tag
		ns.NoK = tmp.NoK
		return err
	case 4:
		tmp := new(SerialInt)
		err := cbor.Unmarshal(value, &tmp)

		ns.Tag = tmp.Tag
		ns.Before = tmp.Value
		return err
	case 5:
		tmp := new(SerialInt)
		err := cbor.Unmarshal(value, &tmp)
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
		return cbor.Marshal(SerialHash{Tag: ns.Tag, Value: ns.KeyHash})
	case 1:
		return cbor.Marshal(SerialScripts{Tag: ns.Tag, NativeScripts: ns.NativeScripts})
	case 2:
		return cbor.Marshal(SerialScripts{Tag: ns.Tag, NativeScripts: ns.NativeScripts})
	case 3:
		return cbor.Marshal(SerialNok{Tag: ns.Tag, NoK: ns.NoK, NativeScripts: ns.NativeScripts})
	case 4:
		return cbor.Marshal(SerialInt{Tag: ns.Tag, Value: ns.Before})
	case 5:
		return cbor.Marshal(SerialInt{Tag: ns.Tag, Value: ns.After})

	default:
		return make([]uint8, 0), nil
	}
}
