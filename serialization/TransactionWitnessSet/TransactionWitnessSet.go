package TransactionWitnessSet

import (
	"github.com/Salvionied/apollo/serialization/NativeScript"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/VerificationKeyWitness"
	"github.com/fxamacker/cbor/v2"
)

type normaltws struct {
	VkeyWitnesses      []VerificationKeyWitness.VerificationKeyWitness `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript                     `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []any                                           `cbor:"2,keyasint,omitempty"`
	PlutusV1Script     []PlutusData.PlutusV1Script                     `cbor:"3,keyasint,omitempty"`
	PlutusV2Script     []PlutusData.PlutusV2Script                     `cbor:"6,keyasint,omitempty"`
	PlutusV3Script     []PlutusData.PlutusV3Script                     `cbor:"7,keyasint,omitempty"`
	PlutusData         *PlutusData.PlutusIndefArray                    `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer                             `cbor:"5,keyasint,omitempty"`
}
type TransactionWitnessSet struct {
	VkeyWitnesses      []VerificationKeyWitness.VerificationKeyWitness `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript                     `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []any                                           `cbor:"2,keyasint,omitempty"`
	PlutusV1Script     []PlutusData.PlutusV1Script                     `cbor:"3,keyasint,omitempty"`
	PlutusV2Script     []PlutusData.PlutusV2Script                     `cbor:"6,keyasint,omitempty"`
	PlutusV3Script     []PlutusData.PlutusV3Script                     `cbor:"7,keyasint,omitempty"`
	PlutusData         PlutusData.PlutusIndefArray                     `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer                             `cbor:"5,keyasint,omitempty"`
}

type WithRedeemerNoScripts struct {
	VkeyWitnesses      []VerificationKeyWitness.VerificationKeyWitness `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript                     `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []any                                           `cbor:"2,keyasint,omitempty"`
	PlutusV1Script     []PlutusData.PlutusV1Script                     `cbor:"3,keyasint,"`
	PlutusV2Script     []PlutusData.PlutusV2Script                     `cbor:"6,keyasint,omitempty"`
	PlutusV3Script     []PlutusData.PlutusV3Script                     `cbor:"7,keyasint,omitempty"`
	PlutusData         *PlutusData.PlutusIndefArray                    `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer                             `cbor:"5,keyasint,omitempty"`
}

/*
*

	MarshalCBOR serializes the TransactionWitnessSet to a CBOR byte slice.

	Returns:
		[]byte: The CBOR-serialized TransactionWitnessSet.
		error: An error if serialization fails.
*/
func (tws *TransactionWitnessSet) MarshalCBOR() ([]byte, error) {
	res := normaltws{}
	if len(tws.VkeyWitnesses) > 0 {
		res.VkeyWitnesses = tws.VkeyWitnesses
	}
	if len(tws.NativeScripts) > 0 {
		res.NativeScripts = tws.NativeScripts
	}
	if len(tws.BootstrapWitnesses) > 0 {
		res.BootstrapWitnesses = tws.BootstrapWitnesses
	}
	if len(tws.PlutusV1Script) > 0 {
		res.PlutusV1Script = tws.PlutusV1Script
	}
	if len(tws.PlutusV2Script) > 0 {
		res.PlutusV2Script = tws.PlutusV2Script
	}
	if len(tws.PlutusV3Script) > 0 {
		res.PlutusV3Script = tws.PlutusV3Script
	}
	if len(tws.PlutusData) > 0 {
		res.PlutusData = &tws.PlutusData
	}
	if len(tws.Redeemer) > 0 {
		res.Redeemer = tws.Redeemer
	}
	return cbor.Marshal(res)
}
