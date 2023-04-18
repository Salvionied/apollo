package shelleymary

import (
	"Salvionied/apollo/ledger/common"
	"Salvionied/apollo/ledger/shelley"
	"Salvionied/apollo/serialization/Address"
	"Salvionied/apollo/serialization/MultiAsset"
	"Salvionied/apollo/serialization/Value"
)

type Transaction struct {
	_                     struct{} `cbor:",toarray"`
	Body                  TransactionBody
	TransactionWitnessSet shelley.TransactionWitnessSet
	AuxiliaryData         interface{}
}

type TransactionBody struct {
	_                     struct{}                     `cbor:",toarray"`
	TransactionInputs     []TransactionInput           `cbor:"0,keyasint"`
	TransactionOutputs    []TransactionOutput          `cbor:"1,keyasint"`
	Fee                   uint64                       `cbor:"2,keyasint"`
	Ttl                   uint64                       `cbor:"3,keyasint"`
	Certificates          []interface{}                `cbor:"4,keyasint,omitempty"`
	Withdrawals           []interface{}                `cbor:"5,keyasint,omitempty"`
	Updates               []interface{}                `cbor:"6,keyasint,omitempty"`
	MetadataHash          []byte                       `cbor:"7,keyasint,omitempty"`
	ValidityIntervalStart uint64                       `cbor:"8,keyasint,omitempty"`
	Mint                  MultiAsset.MultiAsset[int64] `cbor:"9,keyasint,omitempty"`
}

type TransactionInput struct {
	_     struct{} `cbor:",toarray"`
	TxId  common.Hash
	Index uint
}

type TransactionOutput struct {
	_      struct{} `cbor:",toarray"`
	Addr   Address.Address
	Amount Value.Value
}
