package shelley

import (
	"github.com/github.com/salvionied/apollo/ledger/common"
	"github.com/github.com/salvionied/apollo/serialization/NativeScript"
)

type ShelleyBlock struct {
	_     struct{} `cbor:",toarray"`
	Id    uint
	Block Block
}

type Block struct {
	_                      struct{} `cbor:",toarray"`
	Header                 ShelleyBlockHeader
	TransactionBodies      []Transaction
	TransactionWitnessSets []TransactionWitnessSet
	TransactionMetadataSet map[int]interface{}
}

type ShelleyBlockHeader struct {
	_             struct{} `cbor:",toarray"`
	HeaderBody    ShelleyBlockHeaderBody
	BodySignature interface{}
}

type ShelleyBlockHeaderBody struct {
	_                    struct{} `cbor:",toarray"`
	BlockNumber          uint64
	Slot                 uint64
	PrevHash             common.Hash
	IssuerVkey           interface{}
	VrfKey               interface{}
	NonceVrf             interface{}
	LeaderVrf            interface{}
	BlockBodySize        uint32
	BlockBodyHash        common.Hash
	OpCertHotVkey        interface{}
	OpCertSequenceNumber uint32
	OpCertKesPeriod      uint32
	OpCertSignature      interface{}
	ProtoMajorVersion    uint64
	ProtoMinorVersion    uint64
}

type VkeyWitness struct {
	_         struct{} `cbor:",toarray"`
	Vkey      common.PubKey
	Signature common.Signature
}

type BootstrapWitness struct {
	_          struct{} `cbor:",toarray"`
	PublicKey  common.PubKey
	Signature  common.Signature
	ChainCode  common.Hash
	Attributes interface{}
}

type TransactionWitnessSet struct {
	VkeyWitnesses      []VkeyWitness               `cbor:"0,keyasint,omitempty"`
	MultiSigScripts    []NativeScript.NativeScript `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []BootstrapWitness          `cbor:"2,keyasint,omitempty"`
}
