package TransactionBody

import (
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Withdrawal"

	"github.com/blinklabs-io/gouroboros/cbor"
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
	bytes, err := cbor.Encode(tx)
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
	return cbor.Encode(cborBody)
}

/*
func (tb *TransactionBody) UnmarshalCBOR(data []byte) error {
	var decoded interface{}
	_, err := cbor.Decode(data, &decoded)
	if err != nil {
		return err
	}

	if m, ok := decoded.(map[interface{}]interface{}); ok {
		for k, v := range m {
			if _, ok := k.([]interface{}); ok {
				continue
			}
			var key = k
			if karr, ok := k.([]interface{}); ok && len(karr) == 2 {
				if tag, ok := karr[0].(uint64); ok && tag == 0 {
					key = karr[1]
				}
			}
			switch key {
			case 0:
				// inputs
				if arr, ok := v.([]interface{}); ok {
					cleanV := make([]interface{}, len(arr))
					for i, elem := range arr {
						if vm, ok := elem.(map[interface{}]interface{}); ok {
							cleanElem := make(map[interface{}]interface{})
							for vk, vv := range vm {
								if _, ok := vk.([]interface{}); !ok {
									cleanElem[vk] = vv
								}
							}
							cleanV[i] = cleanElem
						} else {
							cleanV[i] = elem
						}
					}
					v = cleanV
				}
				inputsBytes, err := cbor.Encode(v)
				if err != nil {
					return err
				}
				_, err = cbor.Decode(inputsBytes, &tb.Inputs)
				if err != nil {
					return err
				}
			case 1:
				// outputs
				if arr, ok := v.([]interface{}); ok {
					cleanV := make([]interface{}, len(arr))
					for i, elem := range arr {
						if vm, ok := elem.(map[interface{}]interface{}); ok {
							cleanElem := make(map[interface{}]interface{})
							for vk, vv := range vm {
								if _, ok := vk.([]interface{}); !ok {
									cleanElem[vk] = vv
								}
							}
							cleanV[i] = cleanElem
						} else {
							cleanV[i] = elem
						}
					}
					v = cleanV
				}
				outputsBytes, err := cbor.Encode(v)
				if err != nil {
					return err
				}
				_, err = cbor.Decode(outputsBytes, &tb.Outputs)
				if err != nil {
					return err
				}
			case 2:
				// fee
				fee, ok := v.(uint64)
				if !ok {
					return errors.New("invalid fee")
				}
				tb.Fee = int64(fee)
			case 3:
				// ttl
				ttl, ok := v.(uint64)
				if !ok {
					return errors.New("invalid ttl")
				}
				tb.Ttl = int64(ttl)
			case 4:
				// certificates
				if v != nil {
					certBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					tb.Certificates = &Certificate.Certificates{}
					_, err = cbor.Decode(certBytes, tb.Certificates)
					if err != nil {
						return err
					}
				}
			case 5:
				// withdrawals
				if v != nil {
					withBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					tb.Withdrawals = &Withdrawal.Withdrawal{}
					_, err = cbor.Decode(withBytes, tb.Withdrawals)
					if err != nil {
						return err
					}
				}
			case 6:
				// update proposals
				if v != nil {
					tb.UpdateProposals = v.([]any)
				}
			case 7:
				// auxiliary data hash
				if v != nil {
					hash, ok := v.([]byte)
					if !ok {
						return errors.New("invalid auxiliary data hash")
					}
					tb.AuxiliaryDataHash = hash
				}
			case 8:
				// validity start
				validity, ok := v.(uint64)
				if !ok {
					return errors.New("invalid validity start")
				}
				tb.ValidityStart = int64(validity)
			case 9:
				// mint
				if v != nil {
					mintBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					_, err = cbor.Decode(mintBytes, &tb.Mint)
					if err != nil {
						return err
					}
				}
			case 11:
				// script data hash
				if v != nil {
					hash, ok := v.([]byte)
					if !ok {
						return errors.New("invalid script data hash")
					}
					tb.ScriptDataHash = hash
				}
			case 13:
				// collateral
				if v != nil {
					collBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					_, err = cbor.Decode(collBytes, &tb.Collateral)
					if err != nil {
						return err
					}
				}
			case 14:
				// required signers
				if v != nil {
					signers, ok := v.([]serialization.PubKeyHash)
					if !ok {
						return errors.New("invalid required signers")
					}
					tb.RequiredSigners = signers
				}
			case 15:
				// network id
				if v != nil {
					nid, ok := v.([]byte)
					if !ok {
						return errors.New("invalid network id")
					}
					tb.NetworkId = nid
				}
			case 16:
				// collateral return
				if v != nil {
					returnBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					tb.CollateralReturn = &TransactionOutput.TransactionOutput{}
					_, err = cbor.Decode(returnBytes, tb.CollateralReturn)
					if err != nil {
						return err
					}
				}
			case 17:
				// total collateral
				total, ok := v.(int)
				if !ok {
					return errors.New("invalid total collateral")
				}
				tb.TotalCollateral = total
			case 18:
				// reference inputs
				if v != nil {
					refBytes, err := cbor.Encode(v)
					if err != nil {
						return err
					}
					_, err = cbor.Decode(refBytes, &tb.ReferenceInputs)
					if err != nil {
						return err
					}
				}
			default:
				// ignore
			}
		}
	} else {
		return errors.New("invalid transaction body CBOR")
	}
	return nil
}
*/
