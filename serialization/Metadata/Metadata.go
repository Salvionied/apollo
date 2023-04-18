package Metadata

import (
	"Salvionied/apollo/serialization"
	"Salvionied/apollo/serialization/NativeScript"

	"github.com/Salvionied/cbor/v2"
)

type MinimalMetadata map[string]any

type PoliciesMetadata map[string]MinimalMetadata

type TagMetadata map[string]PoliciesMetadata

type Metadata map[int]TagMetadata

type ShelleyMaryMetadata struct {
	_             struct{}                    `cbor:",toarray,omitempty"`
	Metadata      Metadata                    `cbor:",omitempty"`
	NativeScripts []NativeScript.NativeScript `cbor:",omitempty"`
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

func (ad *AuxiliaryData) SetBasicMetadata(value Metadata) {
	ad._basicMeta = value
}
func (ad *AuxiliaryData) SetAlonzoMetadata(value AlonzoMetadata) {
	ad._AlonzoMeta = value
}
func (ad *AuxiliaryData) SetShelleyMetadata(value ShelleyMaryMetadata) {
	ad._ShelleyMeta = value
}

func (ad *AuxiliaryData) Hash() []byte {
	if len(ad._basicMeta) != 0 || len(ad._ShelleyMeta.Metadata) != 0 || len(ad._AlonzoMeta.Metadata) != 0 {
		marshaled, _ := cbor.Marshal(ad)
		return serialization.Blake2bHash(marshaled)
	} else {
		return nil
	}
}

func (ad *AuxiliaryData) UnmarshalCBOR(value []byte) error {
	err := cbor.Unmarshal(value, &ad._ShelleyMeta)
	if err != nil {
		return err
	}
	return nil
}

func (ad *AuxiliaryData) MarshalCBOR() ([]byte, error) {
	enc, _ := cbor.EncOptions{Sort: cbor.SortCanonical}.EncMode()
	if len(ad._AlonzoMeta.Metadata) != 0 || len(ad._AlonzoMeta.NativeScripts) != 0 || len(ad._AlonzoMeta.PlutusScripts) != 0 {
		return enc.Marshal(ad._AlonzoMeta)
	}
	if len(ad._ShelleyMeta.Metadata) == 0 && len(ad._ShelleyMeta.NativeScripts) == 0 {
		return enc.Marshal(nil)
	}
	return enc.Marshal(ad._ShelleyMeta)
}
