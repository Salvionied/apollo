package Redeemer

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/fxamacker/cbor/v2"
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

/*
*

	Clone creates a deep copy of the ExecutionUnits.

	Returns:
		ExecutionUnits: A new ExecutionUnits instance with the same values.
*/
func (ex *ExecutionUnits) Clone() ExecutionUnits {
	return ExecutionUnits{
		Mem:   ex.Mem,
		Steps: ex.Steps,
	}
}

/*
*

	Sum adds the memory and step of another ExecutionUnits to
	the current instance.

	Params:
		other ExecutionUnits: The ExecutionUnits to add.
*/
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

// TODO: add UnmarshalCBOR for round-trip support.
type Redeemers struct {
	Redeemers []Redeemer
}

type RedeemerKey struct {
	_     struct{} `cbor:",toarray"`
	Tag   RedeemerTag
	Index int
}

type RedeemerValue struct {
	_       struct{} `cbor:",toarray"`
	Data    PlutusData.PlutusData
	ExUnits ExecutionUnits
}

func cborMapHeader(length int) ([]byte, error) {
	if length <= 0x17 {
		return []byte{0xa0 + byte(length)}, nil
	} else if length <= 0xff {
		return []byte{0xb8, byte(length)}, nil
	} else if length <= 0xffff {
		return []byte{
			0xb9,
			byte((length >> 8) & 0xff),
			byte(length & 0xff),
		}, nil
	}
	return nil, fmt.Errorf(
		"cbor map length %d exceeds maximum supported",
		length,
	)
}

func (r *Redeemers) MarshalCBOR() ([]byte, error) {
	sorted := make([]Redeemer, len(r.Redeemers))
	copy(sorted, r.Redeemers)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Tag != sorted[j].Tag {
			return sorted[i].Tag < sorted[j].Tag
		}
		return sorted[i].Index < sorted[j].Index
	})

	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := em.NewEncoder(&buf)
	for _, item := range sorted {
		err := enc.Encode(RedeemerKey{
			Tag:   item.Tag,
			Index: item.Index,
		})
		if err != nil {
			return nil, err
		}
		err = enc.Encode(RedeemerValue{
			Data:    item.Data,
			ExUnits: item.ExUnits,
		})
		if err != nil {
			return nil, err
		}
	}

	m := buf.Bytes()
	header, err := cborMapHeader(len(sorted))
	if err != nil {
		return nil, err
	}
	res := make([]byte, 0, len(header)+len(m))
	res = append(res, header...)
	res = append(res, m...)
	return res, nil
}

/*
*

	Clone creates a deep copy of the Redeemer.

	Returns:
		Redeemer: A new Redeemer instance with the same values.
*/
func (r Redeemer) Clone() Redeemer {
	return Redeemer{
		Tag:     r.Tag,
		Index:   r.Index,
		Data:    r.Data.Clone(),
		ExUnits: r.ExUnits.Clone(),
	}
}
