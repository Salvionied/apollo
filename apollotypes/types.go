package apollotypes

import (
	"bytes"

	"github.com/Salvionied/apollo/serialization"
	serAddress "github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Key"
	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/VerificationKeyWitness"
	"github.com/Salvionied/apollo/txBuilding/Backend/Base"
)

type Wallet interface {
	GetAddress() *serAddress.Address
	SignTx(
		tx Transaction.Transaction,
		usedUtxos []UTxO.UTxO,
	) TransactionWitnessSet.TransactionWitnessSet
	PkeyHash() serialization.PubKeyHash
	SkeyHash() serialization.PubKeyHash
	//SignMessage(address serAddress.Address, message []uint8) []uint8
}

type ExternalWallet struct {
	Address serAddress.Address
}

/*
*

	GetAddress returns the address associated with an external wallet.

	Returns:
		*serAddress.Address: A pointer to the address of the external wallet.
*/
func (ew *ExternalWallet) GetAddress() *serAddress.Address {
	return &ew.Address
}

/*
*

	SignTx signs a transaction using an external wallet.

	Params:
		tx (Transaction.Transaction): The transaction to be signed.

	Returns:


	TransactionWitnessSet.TransactionWitnessSet: The withness set associated with the signed transaction.
*/
func (ew *ExternalWallet) SignTx(
	tx Transaction.Transaction,
	usedUtxos []UTxO.UTxO,
) TransactionWitnessSet.TransactionWitnessSet {
	return tx.TransactionWitnessSet
}

/*
*

	PkeyHash returns the public key hash assoicated with an external wallet.
	It computes and returns the public key hash based on the PaymentPart
	of the wallet's address.

	Returns:
		serialization.PubKeyHash: The public key hash of the external wallet.
*/
func (ew *ExternalWallet) PkeyHash() serialization.PubKeyHash {
	res := serialization.PubKeyHash(ew.Address.PaymentPart)
	return res
}

func (ew *ExternalWallet) SkeyHash() serialization.PubKeyHash {
	res := serialization.PubKeyHash(ew.Address.StakingPart)
	return res
}

type GenericWallet struct {
	SigningKey           Key.SigningKey
	VerificationKey      Key.VerificationKey
	Address              serAddress.Address
	StakeSigningKey      Key.SigningKey
	StakeVerificationKey Key.VerificationKey
}

/*
*

	PkeyHash calculates and returns the public key hash associated with a generic wallet.


		It computes the public key hash by calling the Hash() method on the wallet's VerificationKey.
		Then it returns as a serialization.PubKeyHas type.

		Returns:
	   		serialization.PubKeyHash: The public key hash of the generic wallet.
*/
func (gw *GenericWallet) PkeyHash() serialization.PubKeyHash {
	res, _ := gw.VerificationKey.Hash()
	return res
}

/*
*

	PkeyHash calculates and returns the stake key hash associated with a generic wallet.


		It computes the public key hash by calling the Hash() method on the wallet's StakeVerificationKey.
		Then it returns as a serialization.PubKeyHash type.

		Returns:
	   		serialization.PubKeyHash: The public stake key hash of the generic wallet.
*/
func (gw *GenericWallet) SkeyHash() serialization.PubKeyHash {
	res, _ := gw.StakeVerificationKey.Hash()
	return res
}

/*
*

	GetAddress returns the address associated with a generic wallet.

	Returns:
		*serAddress.Address: A pointer to the address of a generic wallet.
*/
func (gw *GenericWallet) GetAddress() *serAddress.Address {
	return &gw.Address
}

/*
*

	SignTx signs a transaction using a generic wallet and returns the updated TransactionWitnessSet.


	It takes a transaction of type Transaction.Transaction and signs it using the wallet's SigningKey.


	Then it appends the corresponding VerificationKeyWitness to the TransactionWitnessSet and returns
	the updated witness set.

	Parameters:
	   	wallet (*GenericWallet): A pointer to a generic wallet.
		tx (Transaction.Transaction): The transaction to be signed.

	Returns:


	TransactionWitnessSet.TransactionWitnessSet: The updated TransactionWitnessSet after signing the transaction.
*/
func (wallet *GenericWallet) SignTx(
	tx Transaction.Transaction,
	usedUtxos []UTxO.UTxO,
) TransactionWitnessSet.TransactionWitnessSet {
	witness_set := tx.TransactionWitnessSet
	txHash, _ := tx.TransactionBody.Hash()
	if isKeyHashUsedFromUtxos(usedUtxos, wallet.PkeyHash()) ||
		isKeyHashUsedFromTx(tx, wallet.PkeyHash()) {
		signature, _ := wallet.SigningKey.Sign(txHash)

		witness_set.VkeyWitnesses = append(
			witness_set.VkeyWitnesses,
			VerificationKeyWitness.VerificationKeyWitness{
				Vkey:      wallet.VerificationKey,
				Signature: signature,
			},
		)
	}

	if isKeyHashUsedFromUtxos(usedUtxos, wallet.SkeyHash()) ||
		isKeyHashUsedFromTx(tx, wallet.SkeyHash()) {
		signature, _ := wallet.StakeSigningKey.Sign(txHash)

		witness_set.VkeyWitnesses = append(
			witness_set.VkeyWitnesses,
			VerificationKeyWitness.VerificationKeyWitness{
				Vkey:      wallet.StakeVerificationKey,
				Signature: signature,
			},
		)
	}

	return witness_set
}

func isKeyHashUsedFromUtxos(
	usedUtxos []UTxO.UTxO,
	keyHash serialization.PubKeyHash,
) bool {
	for _, utxo := range usedUtxos {
		utxoKeyHash := serialization.PubKeyHash(utxo.Output.GetAddress().PaymentPart)
		if utxoKeyHash == keyHash {
			return true
		}
	}
	return false
}

func isKeyHashUsedFromTx(
	tx Transaction.Transaction,
	keyHash serialization.PubKeyHash,
) bool {
	keyHashBytes := keyHash[:]
	if tx.TransactionBody.Certificates != nil {
		for _, certificate := range *tx.TransactionBody.Certificates {
			if certificate.StakeCredential.KeyHash() == keyHash {
				return true
			}
		}
	}
	if tx.TransactionBody.Withdrawals != nil {
		for withdrawal := range *tx.TransactionBody.Withdrawals {
			withdrawalBytes := withdrawal[1:]
			if bytes.Equal(withdrawalBytes, keyHashBytes) {
				return true
			}
		}
	}
	for _, requiredSigner := range tx.TransactionBody.RequiredSigners {
		if requiredSigner == keyHash {
			return true
		}
	}
	for _, nativeScript := range tx.TransactionWitnessSet.NativeScripts {
		if bytes.Equal(nativeScript.KeyHash, keyHashBytes) {
			return true
		}
	}

	return false
}

type Backend Base.ChainContext

type Address serAddress.Address
