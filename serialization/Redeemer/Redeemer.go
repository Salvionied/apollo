package Redeemer

import (
	"fmt"

	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/blinklabs-io/gouroboros/cbor"
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

type ExecutionUnits []int64

/*
*

	Clone creates a deep copy of the ExecutionUnits.

	Returns:
		ExecutionUnits: A new ExecutionUnits instance with the same values.
*/
func (ex *ExecutionUnits) Clone() ExecutionUnits {
	return ExecutionUnits{ex.Mem(), ex.Steps()}
}

/*
*

	Sum adds the memory and step of another ExecutionUnits to
	the current instance.

	Params:
		other ExecutionUnits: The ExecutionUnits to add.
*/
func (eu *ExecutionUnits) Sum(other ExecutionUnits) {
	if len(other) == 0 {
		return
	}
	eu.SetMem(eu.Mem() + other.Mem())
	eu.SetSteps(eu.Steps() + other.Steps())
}

func (eu ExecutionUnits) Mem() int64 {
	if len(eu) > 0 {
		return eu[0]
	}
	return 0
}

func (eu ExecutionUnits) Steps() int64 {
	if len(eu) > 1 {
		return eu[1]
	}
	return 0
}

func (eu *ExecutionUnits) SetMem(m int64) {
	if len(*eu) < 1 {
		*eu = append(*eu, 0)
	}
	(*eu)[0] = m
}

func (eu *ExecutionUnits) SetSteps(s int64) {
	if len(*eu) < 2 {
		for len(*eu) < 2 {
			*eu = append(*eu, 0)
		}
	}
	(*eu)[1] = s
}

// TODO
type Redeemer struct {
	Tag     RedeemerTag
	Index   int
	Data    PlutusData.PlutusData
	ExUnits ExecutionUnits
}

func (r Redeemer) MarshalCBOR() ([]byte, error) {
	dataBytes, err := r.Data.MarshalCBOR()
	if err != nil {
		return nil, err
	}
	return cbor.Encode(
		[]interface{}{
			r.Tag,
			r.Index,
			cbor.RawMessage(dataBytes),
			r.ExUnits.Mem(),
			r.ExUnits.Steps(),
		},
	)
}

func (r *Redeemer) UnmarshalCBOR(data []byte) error {
	var arr []interface{}
	_, err := cbor.Decode(data, &arr)
	if err != nil {
		return err
	}
	if len(arr) < 4 || len(arr) > 5 {
		return fmt.Errorf("invalid redeemer array length: %d", len(arr))
	}
	r.Tag = RedeemerTag(arr[0].(uint64))
	r.Index = int(arr[1].(uint64))
	// arr[2] is the PlutusData, decode it
	dataBytes, err := cbor.Encode(arr[2])
	if err != nil {
		return err
	}
	err = r.Data.UnmarshalCBOR(dataBytes)
	if err != nil {
		return err
	}
	if len(arr) == 5 {
		r.ExUnits = ExecutionUnits{0, 0}
		r.ExUnits.SetMem(int64(arr[3].(uint64)))
		r.ExUnits.SetSteps(int64(arr[4].(uint64)))
	} else {
		// len == 4, arr[3] is exunits
		exBytes, err := cbor.Encode(arr[3])
		if err != nil {
			return err
		}
		_, err = cbor.Decode(exBytes, &r.ExUnits)
		if err != nil {
			return err
		}
	}
	return nil
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
