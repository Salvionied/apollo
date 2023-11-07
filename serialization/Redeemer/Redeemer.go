package Redeemer

import "github.com/Salvionied/apollo/serialization/PlutusData"

type RedeemerTag int

const (
	SPEND RedeemerTag = iota
	MINT
	CERT
	REWARD
)

var RdeemerTagNames = map[RedeemerTag]string{
	0: "spend",
	1: "mint",
	2: "cert",
	3: "reward",
}

type ExecutionUnits struct {
	_     struct{} `cbor:",toarray"`
	Mem   int64
	Steps int64
}

/**
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

/**
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

/**
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
