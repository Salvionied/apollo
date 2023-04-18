package babbage

import (
	"Salvionied/apollo/ledger/common"
	"Salvionied/apollo/serialization"
	"Salvionied/apollo/serialization/Address"
	"Salvionied/apollo/serialization/MultiAsset"
	"Salvionied/apollo/serialization/Value"
)

type Transaction struct {
	_             struct{} `cbor:",toarray"`
	TxBody        TransactionBody
	TxWitnessSet  TransactionWitnessSet
	Validity      bool
	AuxiliaryData interface{}
}

type TransactionBody struct {
	Inputs                []TransactionInput           `cbor:"0,keyasint"`
	Outputs               []TransactionOutput          `cbor:"1,keyasint"`
	Fee                   uint64                       `cbor:"2,keyasint"`
	Ttl                   uint64                       `cbor:"3,keyasint"`
	Certificates          []interface{}                `cbor:"4,keyasint"`
	Withdrawals           []interface{}                `cbor:"5,keyasint"`
	Updates               []interface{}                `cbor:"6,keyasint"`
	AuxiliaryDataHash     common.Hash                  `cbor:"7,keyasint"`
	ValidityIntervalStart uint64                       `cbor:"8,keyasint"`
	Mint                  MultiAsset.MultiAsset[int64] `cbor:"9,keyasint"`
	ScriptDataHash        serialization.ScriptHash     `cbor:"11,keyasint"`
	Collateral            []TransactionInput           `cbor:"13,keyasint"`
	RequiredSigners       []serialization.PubKeyHash   `cbor:"14,keyasint"`
	NetworkId             uint                         `cbor:"15,keyasint"`
	CollateralReturn      TransactionOutput            `cbor:"16,keyasint"`
	TotalCollateral       uint64                       `cbor:"17,keyasint"`
	ReferenceInputs       []TransactionInput           `cbor:"18,keyasint"`
}

type TransactionInput struct {
	_     struct{} `cbor:",toarray"`
	TxId  common.Hash
	Index uint
}

type DatumOptionKey uint

const (
	DatumOptionKeyHash DatumOptionKey = iota
	DatumOptionKeyScript
)

type DatumOption struct {
	Key DatumOptionKey
	Val []byte
}

type TransactionOutput struct {
	Address     Address.Address `cbor:"0,keyasint"`
	Amount      Value.Value     `cbor:"1,keyasint"`
	DatumOption `cbor:"2,keyasint,omitempty"`
	ScriptRef   []byte `cbor:"3,keyasint,omitempty"`
}
