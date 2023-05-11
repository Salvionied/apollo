package shelleymary

import "github.com/github.com/salvionied/apollo/ledger/shelley"

type ShelleyMaryBlock struct {
	_     struct{} `cbor:",toarray"`
	Id    uint
	Block Block
}

type Block struct {
	_                      struct{} `cbor:",toarray"`
	Header                 ShelleyMaryBlockHeader
	TransactionBodies      []Transaction
	TransactionWitnessSets []TransactionWitnessSet
	AuxiliaryDataSets      map[uint]map[int]interface{}
}

type ShelleyMaryBlockHeader shelley.ShelleyBlockHeader

type TransactionWitnessSet shelley.TransactionWitnessSet
