package shelley

import "github.com/github.com/salvionied/apollo/ledger/common"

type Transaction struct {
	TransactionInputs  []TransactionInput  `cbor:"0,keyasint"`
	TransactionOutputs []TransactionOutput `cbor:"1,keyasint"`
	Fee                uint64              `cbor:"2,keyasint"`
	Ttl                uint64              `cbor:"3,keyasint"`
	Certificates       []interface{}       `cbor:"4,keyasint,omitempty"`
	Withdrawals        []interface{}       `cbor:"5,keyasint,omitempty"`
	Updates            []interface{}       `cbor:"6,keyasint,omitempty"`
	MetadataHash       []byte              `cbor:"7,keyasint,omitempty"`
}

type TransactionInput struct {
	_    struct{} `cbor:",toarray"`
	TxId common.Hash
	Idx  uint32
}

type Address []byte

type TransactionOutput struct {
	_       struct{} `cbor:",toarray"`
	Address Address
	Coin    uint64
}
