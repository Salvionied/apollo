package Redeemer

import "github.com/SundaeSwap-finance/apollo/serialization/PlutusData"

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

func (r Redeemer) Clone() Redeemer {
	return Redeemer{
		Tag:     r.Tag,
		Index:   r.Index,
		Data:    r.Data.Clone(),
		ExUnits: r.ExUnits.Clone(),
	}
}
