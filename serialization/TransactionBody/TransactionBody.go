package TransactionBody

import (
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Withdrawal"

	"github.com/fxamacker/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

type TransactionBody struct {
	Inputs            []TransactionInput.TransactionInput   `cbor:"0,keyasint"`
	Outputs           []TransactionOutput.TransactionOutput `cbor:"1,keyasint"`
	Fee               int64                                 `cbor:"2,keyasint"`
	Ttl               int64                                 `cbor:"3,keyasint,omitempty"`
	Certificates      *Certificate.Certificates             `cbor:"4,keyasint,omitempty"`
	Withdrawals       *Withdrawal.Withdrawal                `cbor:"5,keyasint,omitempty"`
	UpdateProposals   []any                                 `cbor:"6,keyasint,omitempty"`
	AuxiliaryDataHash []byte                                `cbor:"7,keyasint,omitempty"`
	ValidityStart     int64                                 `cbor:"8,keyasint,omitempty"`
	Mint              MultiAsset.MultiAsset[int64]          `cbor:"9,keyasint,omitempty"`
	ScriptDataHash    []byte                                `cbor:"11,keyasint,omitempty"`
	Collateral        []TransactionInput.TransactionInput   `cbor:"13,keyasint,omitempty"`
	RequiredSigners   []serialization.PubKeyHash            `cbor:"14,keyasint,omitempty"`
	NetworkId         []byte                                `cbor:"15,keyasint,omitempty"`
	CollateralReturn  *TransactionOutput.TransactionOutput  `cbor:"16,keyasint,omitempty"`
	TotalCollateral   int                                   `cbor:"17,keyasint,omitempty"`
	ReferenceInputs   []TransactionInput.TransactionInput   `cbor:"18,keyasint,omitempty"`
}

type CborBody struct {
	Inputs            []TransactionInput.TransactionInput   `cbor:"0,keyasint"`
	Outputs           []TransactionOutput.TransactionOutput `cbor:"1,keyasint"`
	Fee               int64                                 `cbor:"2,keyasint"`
	Ttl               int64                                 `cbor:"3,keyasint,omitempty"`
	Certificates      *Certificate.Certificates             `cbor:"4,keyasint,omitempty"`
	Withdrawals       *Withdrawal.Withdrawal                `cbor:"5,keyasint,omitempty"`
	UpdateProposals   []any                                 `cbor:"6,keyasint,omitempty"`
	AuxiliaryDataHash []byte                                `cbor:"7,keyasint,omitempty"`
	ValidityStart     int64                                 `cbor:"8,keyasint,omitempty"`
	Mint              MultiAsset.MultiAsset[int64]          `cbor:"9,keyasint,omitempty"`
	ScriptDataHash    []byte                                `cbor:"11,keyasint,omitempty"`
	Collateral        []TransactionInput.TransactionInput   `cbor:"13,keyasint,omitempty"`
	RequiredSigners   []serialization.PubKeyHash            `cbor:"14,keyasint,omitempty"`
	NetworkId         []byte                                `cbor:"15,keyasint,omitempty"`
	CollateralReturn  *TransactionOutput.TransactionOutput  `cbor:"16,keyasint,omitempty"`
	TotalCollateral   int                                   `cbor:"17,keyasint,omitempty"`
	ReferenceInputs   []TransactionInput.TransactionInput   `cbor:"18,keyasint,omitempty"`
}

func (tx *TransactionBody) Hash() ([]byte, error) {
	bytes, err := cbor.Marshal(tx)
	if err != nil {
		return nil, err
	}
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil, err
	}
	_, err = hash.Write(bytes)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil

}

func (tx *TransactionBody) Id() (serialization.TransactionId, error) {
	bytes, err := tx.Hash()
	if err != nil {
		return serialization.TransactionId{}, err
	}
	return serialization.TransactionId{Payload: bytes}, nil
}

func (tx *TransactionBody) MarshalCBOR() ([]byte, error) {
	cborBody := CborBody{
		Inputs:            tx.Inputs,
		Outputs:           tx.Outputs,
		Fee:               tx.Fee,
		Ttl:               tx.Ttl,
		Certificates:      tx.Certificates,
		Withdrawals:       tx.Withdrawals,
		UpdateProposals:   tx.UpdateProposals,
		AuxiliaryDataHash: tx.AuxiliaryDataHash,
		ValidityStart:     tx.ValidityStart,
		Mint:              tx.Mint,
		ScriptDataHash:    tx.ScriptDataHash,
		Collateral:        tx.Collateral,
		RequiredSigners:   tx.RequiredSigners,
		NetworkId:         tx.NetworkId,
		CollateralReturn:  tx.CollateralReturn,
		TotalCollateral:   tx.TotalCollateral,
		ReferenceInputs:   tx.ReferenceInputs,
	}
	em, _ := cbor.CanonicalEncOptions().EncMode()
	return em.Marshal(cborBody)
}
