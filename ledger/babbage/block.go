package babbage

import (
	"Salvionied/apollo/ledger/shelley"
	"Salvionied/apollo/serialization/NativeScript"
	"Salvionied/apollo/serialization/PlutusData"
	"Salvionied/apollo/serialization/Redeemer"
)

type Block struct {
	_            struct{} `cbor:",toarray"`
	Id           uint
	BabbageBlock BabbageBlock
}

type BabbageBlock struct {
	_                      struct{} `cbor:",toarray"`
	Header                 shelley.ShelleyBlockHeader
	TransactionBodies      []Transaction
	TransactionWitnessSets []TransactionWitnessSet
	AuxiliaryDataSets      map[uint]map[int]interface{}
	InvalidTransactions    []uint
}

type TransactionWitnessSet struct {
	VkeyWitnesses      []shelley.VkeyWitness       `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []shelley.BootstrapWitness  `cbor:"2,keyasint,omitempty"`
	PlutusScriptV1     []PlutusData.PlutusV1Script `cbor:"3,keyasint,omitempty"`
	PlutusData         []PlutusData.PlutusData     `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer         `cbor:"5,keyasint,omitempty"`
	PlutusScriptV2     []PlutusData.PlutusV2Script `cbor:"6,keyasint,omitempty"`
}
