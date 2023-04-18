package alonzo

import (
	"Salvionied/apollo/ledger/common"
	"Salvionied/apollo/ledger/shelley"
	"Salvionied/apollo/serialization"
	"Salvionied/apollo/serialization/Address"
	"Salvionied/apollo/serialization/MultiAsset"
	"Salvionied/apollo/serialization/Value"
)

type Transaction struct {
	_             struct{} `cbor:",toarray"`
	TxBody        TransactionBody
	TxWitness     TransactionWitnessSet
	Validity      bool
	AuxiliaryData interface{}
}

type TransactionBody struct {
	Inputs                []shelley.TransactionInput   `cbor:"0,keyasint"`
	Outputs               []TxOut                      `cbor:"1,keyasint"`
	Fee                   uint64                       `cbor:"2,keyasint"`
	Ttl                   uint64                       `cbor:"3,keyasint"`
	Certificates          []interface{}                `cbor:"4,keyasint,omitempty"`
	Withdrawals           []interface{}                `cbor:"5,keyasint,omitempty"`
	Updates               []interface{}                `cbor:"6,keyasint,omitempty"`
	AuxiliaryDataHash     []byte                       `cbor:"7,keyasint,omitempty"`
	ValidityIntervalStart uint64                       `cbor:"8,keyasint,omitempty"`
	Mint                  MultiAsset.MultiAsset[int64] `cbor:"9,keyasint,omitempty"`
	ScriptDataHash        serialization.ScriptHash     `cbor:"11,keyasint,omitempty"`
	Collateral            []shelley.TransactionInput   `cbor:"13,keyasint,omitempty"`
	RequiredSigners       []common.PubKeyHash          `cbor:"14,keyasint,omitempty"`
	NetworkId             uint                         `cbor:"15,keyasint,omitempty"`
}

type TxOut struct {
	_         struct{} `cbor:",toarray"`
	Addr      Address.Address
	Amount    Value.Value
	DatumHash common.Hash `cbor:"omitempty"`
}
