package Metadata

import (
	"github.com/Salvionied/apollo/serialization/NativeScript"

	"github.com/Salvionied/apollo/serialization"

	"github.com/fxamacker/cbor/v2"
)

type MinimalMetadata map[string]any

type PoliciesMetadata map[string]MinimalMetadata

type TagMetadata map[string]any

type Metadata map[int]any

type ShelleyMaryMetadata struct {
	_             struct{}                    `cbor:",toarray,omitempty"`
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
	enc, _ := cbor.EncOptions{Sort: cbor.SortLengthFirst}.EncMode()
	if len(smm.NativeScripts) > 0 {
		return enc.Marshal(smm)
	} else {
		return enc.Marshal(smm.Metadata)
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
		for key, val := range value.Metadata {
			currentMetadata[key] = val
		}

	}

}

/*
*

	Hash computes computes the has of the AuxiliaryData.

	Returns:
		[]byte: The computed hash or nil if all metadata fileds are empty.
*/
func (ad *AuxiliaryData) Hash() []byte {
	if len(ad._basicMeta) != 0 || len(ad._ShelleyMeta.Metadata) != 0 || len(ad._AlonzoMeta.Metadata) != 0 {
		marshaled, _ := cbor.Marshal(ad)
		hash, err := serialization.Blake2bHash(marshaled)
		if err != nil {
			return nil
		}
		return hash
	} else {
		return nil
	}
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
	err_shelley := cbor.Unmarshal(value, &ad._ShelleyMeta)
	if err_shelley != nil {
		err_basic_meta := cbor.Unmarshal(value, &ad._basicMeta)
		if err_basic_meta != nil {
			return err_basic_meta
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
	enc, _ := cbor.EncOptions{Sort: cbor.SortLengthFirst}.EncMode()
	if len(ad._basicMeta) != 0 {
		return enc.Marshal(ad._basicMeta)
	}
	if len(ad._AlonzoMeta.Metadata) != 0 || len(ad._AlonzoMeta.NativeScripts) != 0 || len(ad._AlonzoMeta.PlutusScripts) != 0 {
		return enc.Marshal(ad._AlonzoMeta)
	}
	if len(ad._ShelleyMeta.Metadata) == 0 && len(ad._ShelleyMeta.NativeScripts) == 0 {
		return enc.Marshal(nil)
	}
	return enc.Marshal(ad._ShelleyMeta)
}
