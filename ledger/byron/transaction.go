package byron

import "Salvionied/apollo/ledger/common"

type TxWithWitness struct {
	_         struct{} `cbor:",toarray"`
	Tx        Transaction
	Witnesses []TxWitness
}

type SpentTx struct {
	_    struct{} `cbor:",toarray"`
	TxId common.Hash
	Idx  uint32
}
type TxIn struct {
	_       struct{} `cbor:",toarray"`
	Id      uint8
	SpentTx SpentTx `cbor:"24,keyasint"`
}

type Address struct {
	_   struct{} `cbor:",toarray"`
	Val []byte   `cbor:"24,keyasint"`
	Id  uint64
}

type TxOut struct {
	_       struct{} `cbor:",toarray"`
	Address Address
	Coin    uint64
}

type Transaction struct {
	_       struct{} `cbor:",toarray"`
	Inputs  []TxIn
	Outputs []TxOut
	Attribs interface{}
}

type TxWitness struct {
	_  struct{} `cbor:",toarray"`
	Id uint8
	//TODO IMPLEMENT THIS
	// twit = [0, #6.24(bytes .cbor ([pubkey, signature]))]
	// / [1, #6.24(bytes .cbor ([[u16, bytes], [u16, bytes]]))]
	// / [2, #6.24(bytes .cbor ([pubkey, signature]))]
	// / [u8 .gt 2, encoded-cbor]
	Val interface{}
}
