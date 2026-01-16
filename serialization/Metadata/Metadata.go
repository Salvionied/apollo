package Metadata

import (
	"maps"

	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/NativeScript"

	"github.com/blinklabs-io/gouroboros/cbor"
)

type MinimalMetadata map[string]any

type PoliciesMetadata map[string]MinimalMetadata

type TagMetadata map[string]any

type Metadata map[int]any

type ShelleyMaryMetadata struct {
	Metadata      Metadata                    `cbor:",omitempty"`
	NativeScripts []NativeScript.NativeScript `cbor:",omitempty"`
}

/*
*

	MarshalCBOR marshals a ShelleyMaryMetadata instance into CBOR-encoded data.

	Returns:
		([]byte, error): The CBOR-encoded data and an error if marshaling fails,
						 nil otherwise.
*/
func (smm *ShelleyMaryMetadata) MarshalCBOR() ([]byte, error) {
	if len(smm.NativeScripts) > 0 {
		return cbor.Encode(smm)
	} else {
		return cbor.Encode(smm.Metadata)
	}
}

type AlonzoMetadata struct {
	Metadata      Metadata                    `cbor:"0,keyasint,omitempty"`
	NativeScripts []NativeScript.NativeScript `cbor:"1,keyasint,omitempty"`
	PlutusScripts []uint8                     `cbor:"2,keyasint,omitempty"`
}

type AuxiliaryData struct {
	_basicMeta   Metadata
	_ShelleyMeta ShelleyMaryMetadata
	_AlonzoMeta  AlonzoMetadata
}

/*
*

	SetBasicMetadata sets the basic metadata for the AuxiliaryData.
*/
func (ad *AuxiliaryData) SetBasicMetadata(value Metadata) {
	ad._basicMeta = value
}

/*
*

	SetAlonzoMetadata sets the Alonzo metadata for the AuxiliaryData.
*/
func (ad *AuxiliaryData) SetAlonzoMetadata(value AlonzoMetadata) {
	ad._AlonzoMeta = value
}

/*
*

	SetShelleyMetadata sets the Shelley metadata for the AuxiliaryData.
*/
func (ad *AuxiliaryData) SetShelleyMetadata(value ShelleyMaryMetadata) {
	if ad._ShelleyMeta.Metadata == nil {
		ad._ShelleyMeta = value
	} else {
		currentMetadata := ad._ShelleyMeta.Metadata
		maps.Copy(currentMetadata, value.Metadata)

	}

}

/*
*

	IsEmpty returns true if the AuxiliaryData contains no metadata.
	Callers can use this to decide whether to include auxiliary data
	in a transaction.

	Returns:
		bool: True if no metadata is present.
*/
func (ad *AuxiliaryData) IsEmpty() bool {
	return len(ad._basicMeta) == 0 &&
		len(ad._AlonzoMeta.Metadata) == 0 &&
		len(ad._AlonzoMeta.NativeScripts) == 0 &&
		len(ad._AlonzoMeta.PlutusScripts) == 0 &&
		len(ad._ShelleyMeta.Metadata) == 0 &&
		len(ad._ShelleyMeta.NativeScripts) == 0
}

/*
*

	Hash computes computes the has of the AuxiliaryData.

	Returns:
		[]byte: The computed hash or nil if all metadata fileds are empty.
*/
func (ad *AuxiliaryData) Hash() []byte {
	marshaled, _ := cbor.Encode(ad)
	hash, err := serialization.Blake2bHash(marshaled)
	if err != nil {
		return nil
	}
	return hash
}

/*
*

	UnmarshalCBOR deserializes the AuxiliaryData from a CBOR-encoded byte slice.

	Params:
		value []byte: The CBOR-encoded data to deserialize.

	Returns:
		error: An error if deserialization fails.
*/
func (ad *AuxiliaryData) UnmarshalCBOR(value []byte) error {
	_, err_shelley := cbor.Decode(value, &ad._ShelleyMeta)
	if err_shelley != nil {
		_, err_alonzo := cbor.Decode(value, &ad._AlonzoMeta)
		if err_alonzo != nil {
			_, err_basic_meta := cbor.Decode(value, &ad._basicMeta)
			if err_basic_meta != nil {
				return err_basic_meta
			}
		}
	}
	return nil
}

/*
*

	MarshalCBOR serializes the AUxiliaryData to a CBOR byte slice.

	Returns:
		[]byte: The CBOR-serialized AuxiliaryData.
		error: An error if serialization fails.
*/
func (ad *AuxiliaryData) MarshalCBOR() ([]byte, error) {
	if len(ad._basicMeta) != 0 {
		return cbor.Encode(ad._basicMeta)
	}
	if len(ad._AlonzoMeta.Metadata) != 0 ||
		len(ad._AlonzoMeta.NativeScripts) != 0 ||
		len(ad._AlonzoMeta.PlutusScripts) != 0 {
		return cbor.Encode(ad._AlonzoMeta)
	}
	if len(ad._ShelleyMeta.Metadata) == 0 &&
		len(ad._ShelleyMeta.NativeScripts) == 0 {
		// Return empty map - valid CBOR encoding for empty auxiliary data
		return cbor.Encode(map[int]any{})
	}
	return cbor.Encode(ad._ShelleyMeta)
}
