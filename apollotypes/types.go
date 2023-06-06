package apollotypes

import (
	"github.com/salvionied/apollo/serialization"
	serAddress "github.com/salvionied/apollo/serialization/Address"
	"github.com/salvionied/apollo/serialization/Key"
	"github.com/salvionied/apollo/serialization/Transaction"
	"github.com/salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/salvionied/apollo/serialization/VerificationKeyWitness"
	"github.com/salvionied/apollo/txBuilding/Backend/Base"
)

type Wallet interface {
	GetAddress() *serAddress.Address
	SignTx(tx Transaction.Transaction) TransactionWitnessSet.TransactionWitnessSet
	PkeyHash() serialization.PubKeyHash
	//SignMessage(address serAddress.Address, message []uint8) []uint8
}

type GenericWallet struct {
	SigningKey           Key.SigningKey
	VerificationKey      Key.VerificationKey
	Address              serAddress.Address
	StakeSigningKey      Key.StakeSigningKey
	StakeVerificationKey Key.StakeVerificationKey
}

func (gw *GenericWallet) PkeyHash() serialization.PubKeyHash {
	res, _ := gw.VerificationKey.Hash()
	return res
}

func (gw *GenericWallet) GetAddress() *serAddress.Address {
	return &gw.Address
}

func (wallet *GenericWallet) SignTx(tx Transaction.Transaction) TransactionWitnessSet.TransactionWitnessSet {
	witness_set := tx.TransactionWitnessSet
	txHash := tx.TransactionBody.Hash()
	signature := wallet.SigningKey.Sign(txHash)
	witness_set.VkeyWitnesses = append(witness_set.VkeyWitnesses, VerificationKeyWitness.VerificationKeyWitness{Vkey: wallet.VerificationKey, Signature: signature})
	return witness_set
}

type Backend Base.ChainContext

type Address serAddress.Address
