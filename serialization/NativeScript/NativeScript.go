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
