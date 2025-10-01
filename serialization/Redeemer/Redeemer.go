package Redeemer

import (
	"bytes"
	"github.com/SundaeSwap-finance/apollo/serialization/PlutusData"
	"github.com/Salvionied/cbor/v2"
)

type RedeemerTag int

const (
	SPEND RedeemerTag = iota
	MINT
	CERT
	REWARD
)

// See https://ogmios.dev/mini-protocols/local-tx-submission/#evaluatetx
var RedeemerTagNames = map[RedeemerTag]string{
	0: "spend",
	1: "mint",
	2: "certificate",
	3: "withdrawal",
}

type ExecutionUnits struct {
	_     struct{} `cbor:",toarray"`
	Mem   int64
	Steps int64
}

func (ex *ExecutionUnits) Clone() ExecutionUnits {
	return ExecutionUnits{
		Mem:   ex.Mem,
		Steps: ex.Steps,
	}
}

func (eu *ExecutionUnits) Sum(other ExecutionUnits) {
	eu.Mem += other.Mem
	eu.Steps += other.Steps
}

// TODO
type Redeemer struct {
	_       struct{} `cbor:",toarray"`
	Tag     RedeemerTag
	Index   int
	Data    PlutusData.PlutusData
	ExUnits ExecutionUnits
}

type Redeemers struct {
	Redeemers []Redeemer
}

type RedeemerKey struct {
	_       struct{} `cbor:",toarray"`
	Tag     RedeemerTag
	Index   int
}

type RedeemerValue struct {
	_       struct{} `cbor:",toarray"`
	Data    PlutusData.PlutusData
	ExUnits ExecutionUnits
}

func cborMapHeader(length int) []byte {
	if length <= 0x17 {
		return []byte{0xa0 + byte(length)}
	} else if length <= 0xff {
		return []byte{0xb8, byte(length)}
	} else if length <= 0xffff {
		return []byte{0xb9, byte((length >> 8) & 0xff), byte(length & 0xff)}
	} else {
		panic("apollo/serialization/Redeemer/Redeemer: Very long cbor map header")
	}
}

func (r* Redeemers) MarshalCBOR() ([]byte, error) {
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)
	for _, item := range r.Redeemers {
//		// key
//		if err := enc.StartIndefiniteArray(); err != nil {
//			return nil, err
//		}
//		if err := enc.Encode(item.Tag); err != nil {
//			return nil, err
//		}
//		if err := enc.Encode(item.Index); err != nil {
//			return nil, err
//		}
//		if err := enc.EndIndefinite(); err != nil {
//			return nil, err
//		}
//
//		// value
//		if err := enc.StartIndefiniteArray(); err != nil {
//			return nil, err
//		}
//		if err := enc.Encode(item.Data); err != nil {
//			return nil, err
//		}
//		if err := enc.Encode(item.ExUnits); err != nil {
//			return nil, err
//		}
//		if err := enc.EndIndefinite(); err != nil {
//			return nil, err
//		}
                if err := enc.Encode(RedeemerKey{Tag: item.Tag, Index: item.Index}); err != nil {
			return nil, err
		}
		if err := enc.Encode(RedeemerValue{Data: item.Data, ExUnits: item.ExUnits}); err != nil {
			return nil, err
		}
	}

	m := buf.Bytes()
	res := make([]byte, 0)
	res = append(res, cborMapHeader(len(r.Redeemers))...)
	res = append(res, m...)
	return res, nil
}

func (r Redeemer) Clone() Redeemer {
	return Redeemer{
		Tag:     r.Tag,
		Index:   r.Index,
		Data:    r.Data.Clone(),
		ExUnits: r.ExUnits.Clone(),
	}
}
