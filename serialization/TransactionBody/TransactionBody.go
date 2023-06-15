package TransactionBody

import (
	"log"

	"github.com/SundaeSwap-finance/apollo/serialization"
	"github.com/SundaeSwap-finance/apollo/serialization/Certificate"
	"github.com/SundaeSwap-finance/apollo/serialization/MultiAsset"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionInput"
	"github.com/SundaeSwap-finance/apollo/serialization/TransactionOutput"
	"github.com/SundaeSwap-finance/apollo/serialization/Withdrawal"

	"github.com/Salvionied/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

type TransactionBody struct {
	Inputs            []TransactionInput.TransactionInput   `cbor:"0,keyasint"`
	Outputs           []TransactionOutput.TransactionOutput `cbor:"1,keyasint"`
	Fee               int64                                 `cbor:"2,keyasint"`
	Ttl               int64                                 `cbor:"3,keyasint,omitempty"`
	Certificates      *Certificate.Certificates             `cbor:"4,keyasint,omitempty"`
	Withdrawals       []*Withdrawal.Withdrawal              `cbor:"5,keyasint,omitempty"`
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

func (tx *TransactionBody) Hash() []byte {
	bytes, err := cbor.Marshal(tx)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := blake2b.New(32, nil)
	if err != nil {
		return nil
	}
	_, err = hash.Write(bytes)
	if err != nil {
		return nil
	}
	return hash.Sum(nil)

}

func (tx *TransactionBody) Id() serialization.TransactionId {
	return serialization.TransactionId{tx.Hash()}
}
