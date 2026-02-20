package TransactionBody

import (
	"errors"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Certificate"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Withdrawal"
	apolloCbor "github.com/Salvionied/apollo/serialization/cbor"

	"github.com/fxamacker/cbor/v2"
	"golang.org/x/crypto/blake2b"
)

// TransactionInputSet is a wrapper around []TransactionInput that handles
// CBOR tag 258 (set) which may wrap the inputs in Conway+ era transactions.
type TransactionInputSet struct {
	items  []TransactionInput.TransactionInput
	useTag bool // Whether the set was wrapped in tag 258 during decoding
}

// Items returns a defensive copy of the transaction inputs to prevent external
// mutation of the internal slice. Returns an empty slice (not nil) when empty.
func (s *TransactionInputSet) Items() []TransactionInput.TransactionInput {
	result := make([]TransactionInput.TransactionInput, len(s.items))
	copy(result, s.items)
	return result
}

// SetItems sets the transaction inputs using a defensive copy.
// This prevents the caller from mutating the internal slice after setting it.
func (s *TransactionInputSet) SetItems(
	items []TransactionInput.TransactionInput,
) {
	if items == nil {
		s.items = nil
		return
	}
	s.items = make([]TransactionInput.TransactionInput, len(items))
	copy(s.items, items)
}

// UnmarshalCBOR decodes CBOR data that may or may not be wrapped in tag 258.
func (s *TransactionInputSet) UnmarshalCBOR(data []byte) error {
	// Try to decode as a tag first
	var tag apolloCbor.RawTag
	if err := apolloCbor.Decode(data, &tag); err == nil {
		// It's a tag - check if it's tag 258 (set)
		if tag.Number != apolloCbor.CborTagSet {
			return errors.New("unexpected CBOR tag type for transaction inputs")
		}
		data = []byte(tag.Content)
		s.useTag = true
	}

	// Decode as array of transaction inputs
	var inputs []TransactionInput.TransactionInput
	if err := apolloCbor.Decode(data, &inputs); err != nil {
		return err
	}
	s.items = inputs
	return nil
}

// MarshalCBOR encodes the transaction inputs, optionally wrapping in tag 258.
func (s *TransactionInputSet) MarshalCBOR() ([]byte, error) {
	if s.useTag {
		// Wrap in tag 258
		tag := cbor.Tag{
			Number:  apolloCbor.CborTagSet,
			Content: s.items,
		}
		return cbor.Marshal(tag)
	}
	return cbor.Marshal(s.items)
}

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
	TotalCollateral   int64                                 `cbor:"17,keyasint,omitempty"`
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
	TotalCollateral   int64                                 `cbor:"17,keyasint,omitempty"`
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
	// CanonicalEncOptions().EncMode() is not expected to fail with default
	// options, but we still check the error defensively in case the cbor
	// library behavior changes in future versions.
	em, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, err
	}
	return em.Marshal(cborBody)
}

// UnmarshalCBOR deserializes CBOR data into a TransactionBody.
// It handles CBOR tag 258 (set) that may wrap inputs, collateral, and reference inputs.
func (tx *TransactionBody) UnmarshalCBOR(data []byte) error {
	// Decode the map preserving raw bytes for each field value.
	// This is critical because re-encoding Plutus data through Go interfaces
	// can fail when the data contains maps with tag-based keys.
	rawMap, err := apolloCbor.DecodeMapToRaw(data)
	if err != nil {
		return errors.New("decode map: " + err.Error())
	}

	// Field 0: Inputs (may be wrapped in tag 258)
	if rawBytes, ok := rawMap[0]; ok {
		// Check for and unwrap tag 258
		unwrapped, _, err := apolloCbor.UnwrapSetTag(rawBytes)
		if err != nil {
			return errors.New("inputs unwrap: " + err.Error())
		}
		if err := apolloCbor.Decode(unwrapped, &tx.Inputs); err != nil {
			return errors.New("inputs decode: " + err.Error())
		}
	}

	// Field 1: Outputs - decode using our custom decoder
	if rawBytes, ok := rawMap[1]; ok {
		if err := apolloCbor.Decode(rawBytes, &tx.Outputs); err != nil {
			return errors.New("outputs decode: " + err.Error())
		}
	}

	// Field 2: Fee
	if rawBytes, ok := rawMap[2]; ok {
		var fee uint64
		if err := apolloCbor.Decode(rawBytes, &fee); err != nil {
			return errors.New("fee decode: " + err.Error())
		}
		tx.Fee = int64(fee)
	}

	// Field 3: TTL
	if rawBytes, ok := rawMap[3]; ok {
		var ttl uint64
		if err := apolloCbor.Decode(rawBytes, &ttl); err != nil {
			return errors.New("ttl decode: " + err.Error())
		}
		tx.Ttl = int64(ttl)
	}

	// Field 4: Certificates
	if rawBytes, ok := rawMap[4]; ok {
		tx.Certificates = &Certificate.Certificates{}
		if err := apolloCbor.Decode(rawBytes, tx.Certificates); err != nil {
			return errors.New("certificates decode: " + err.Error())
		}
	}

	// Field 5: Withdrawals
	if rawBytes, ok := rawMap[5]; ok {
		tx.Withdrawals = &Withdrawal.Withdrawal{}
		if err := apolloCbor.Decode(rawBytes, tx.Withdrawals); err != nil {
			return errors.New("withdrawals decode: " + err.Error())
		}
	}

	// Field 6: UpdateProposals
	if rawBytes, ok := rawMap[6]; ok {
		var proposals []any
		if err := apolloCbor.Decode(rawBytes, &proposals); err != nil {
			return errors.New("update proposals decode: " + err.Error())
		}
		tx.UpdateProposals = proposals
	}

	// Field 7: AuxiliaryDataHash
	if rawBytes, ok := rawMap[7]; ok {
		var hash []byte
		if err := apolloCbor.Decode(rawBytes, &hash); err != nil {
			return errors.New("aux data hash decode: " + err.Error())
		}
		tx.AuxiliaryDataHash = hash
	}

	// Field 8: ValidityStart
	if rawBytes, ok := rawMap[8]; ok {
		var vs uint64
		if err := apolloCbor.Decode(rawBytes, &vs); err != nil {
			return errors.New("validity start decode: " + err.Error())
		}
		tx.ValidityStart = int64(vs)
	}

	// Field 9: Mint
	if rawBytes, ok := rawMap[9]; ok {
		if err := apolloCbor.Decode(rawBytes, &tx.Mint); err != nil {
			return errors.New("mint decode: " + err.Error())
		}
	}

	// Field 11: ScriptDataHash
	if rawBytes, ok := rawMap[11]; ok {
		var hash []byte
		if err := apolloCbor.Decode(rawBytes, &hash); err != nil {
			return errors.New("script data hash decode: " + err.Error())
		}
		tx.ScriptDataHash = hash
	}

	// Field 13: Collateral (may be wrapped in tag 258)
	if rawBytes, ok := rawMap[13]; ok {
		unwrapped, _, err := apolloCbor.UnwrapSetTag(rawBytes)
		if err != nil {
			return errors.New("collateral unwrap: " + err.Error())
		}
		if err := apolloCbor.Decode(unwrapped, &tx.Collateral); err != nil {
			return errors.New("collateral decode: " + err.Error())
		}
	}

	// Field 14: RequiredSigners
	if rawBytes, ok := rawMap[14]; ok {
		if err := apolloCbor.Decode(rawBytes, &tx.RequiredSigners); err != nil {
			return errors.New("required signers decode: " + err.Error())
		}
	}

	// Field 15: NetworkId
	if rawBytes, ok := rawMap[15]; ok {
		var netId []byte
		if err := apolloCbor.Decode(rawBytes, &netId); err != nil {
			return errors.New("network id decode: " + err.Error())
		}
		tx.NetworkId = netId
	}

	// Field 16: CollateralReturn
	if rawBytes, ok := rawMap[16]; ok {
		tx.CollateralReturn = &TransactionOutput.TransactionOutput{}
		if err := apolloCbor.Decode(rawBytes, tx.CollateralReturn); err != nil {
			return errors.New("collateral return decode: " + err.Error())
		}
	}

	// Field 17: TotalCollateral
	if rawBytes, ok := rawMap[17]; ok {
		var tc uint64
		if err := apolloCbor.Decode(rawBytes, &tc); err != nil {
			return errors.New("total collateral decode: " + err.Error())
		}
		tx.TotalCollateral = int64(tc)
	}

	// Field 18: ReferenceInputs (may be wrapped in tag 258)
	if rawBytes, ok := rawMap[18]; ok {
		unwrapped, _, err := apolloCbor.UnwrapSetTag(rawBytes)
		if err != nil {
			return errors.New("reference inputs unwrap: " + err.Error())
		}
		if err := apolloCbor.Decode(unwrapped, &tx.ReferenceInputs); err != nil {
			return errors.New("reference inputs decode: " + err.Error())
		}
	}

	return nil
}
