package alonzo

import (
	"Salvionied/apollo/ledger/shelley"
	"Salvionied/apollo/serialization/NativeScript"
	"Salvionied/apollo/serialization/PlutusData"
	"Salvionied/apollo/serialization/Redeemer"
)

type Block struct {
	_     struct{} `cbor:",toarray"`
	Id    uint
	Block AlonzoBlock
}

type AlonzoBlock struct {
	_                      struct{} `cbor:",toarray"`
	Header                 AlonzoBlockHeader
	TransactionBodies      []Transaction
	TransactionWitnessSets []TransactionWitnessSet
	AuxiliaryDataSets      map[uint]map[int]interface{}
	InvalidTransactions    []uint
}

type AlonzoBlockHeader shelley.ShelleyBlockHeader

type TransactionWitnessSet struct {
	_                  struct{}                    `cbor:",toarray"`
	VkeyWitnesses      []shelley.VkeyWitness       `cbor:"0,keyasint,omitempty"`
	MultiSigScripts    []NativeScript.NativeScript `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []shelley.BootstrapWitness  `cbor:"2,keyasint,omitempty"`
	PlutusScript       []byte                      `cbor:"3,keyasint,omitempty"`
	PlutusData         []PlutusData.PlutusData     `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer         `cbor:"5,keyasint,omitempty"`
}
