package Redeemer

import (
	"fmt"

	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
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

type ExecutionUnits struct {
	cbor.StructAsArray
	Mem   uint64
	Steps uint64
}

/*
*

	Clone creates a deep copy of the ExecutionUnits.

	Returns:
		ExecutionUnits: A new ExecutionUnits instance with the same values.
*/
func (ex ExecutionUnits) Clone() ExecutionUnits {
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
		[]any{
			r.Tag,
			r.Index,
			cbor.RawMessage(dataBytes),
			r.ExUnits,
		},
	)
}

// UnmarshalCBOR deserializes a CBOR-encoded byte slice into a Redeemer.
// Expected format: [tag, index, data, [mem, steps]]
func (r *Redeemer) UnmarshalCBOR(data []byte) error {
	var arr []any
	_, err := cbor.Decode(data, &arr)
	if err != nil {
		return err
	}
	if len(arr) != 4 {
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
	// arr[3] is exunits [mem, steps]
	exBytes, err := cbor.Encode(arr[3])
	if err != nil {
		return err
	}
	_, err = cbor.Decode(exBytes, &r.ExUnits)
	if err != nil {
		return err
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
