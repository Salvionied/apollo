package TransactionWitnessSet

import (
	"github.com/Salvionied/apollo/serialization/NativeScript"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/VerificationKeyWitness"
	"github.com/Salvionied/cbor/v2"
)

type normaltws struct {
	VkeyWitnesses      []VerificationKeyWitness.VerificationKeyWitness `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript                     `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []any                                           `cbor:"2,keyasint,omitempty"`
	PlutusV1Script     []PlutusData.PlutusV1Script                     `cbor:"3,keyasint,omitempty"`
	PlutusV2Script     []PlutusData.PlutusV2Script                     `cbor:"6,keyasint,omitempty"`
	PlutusData         *PlutusData.PlutusIndefArray                    `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer                             `cbor:"5,keyasint,omitempty"`
}
type TransactionWitnessSet struct {
	VkeyWitnesses      []VerificationKeyWitness.VerificationKeyWitness `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript                     `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []any                                           `cbor:"2,keyasint,omitempty"`
	PlutusV1Script     []PlutusData.PlutusV1Script                     `cbor:"3,keyasint,omitempty"`
	PlutusV2Script     []PlutusData.PlutusV2Script                     `cbor:"6,keyasint,omitempty"`
	PlutusData         PlutusData.PlutusIndefArray                     `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer                             `cbor:"5,keyasint,omitempty"`
}

type WithRedeemerNoScripts struct {
	VkeyWitnesses      []VerificationKeyWitness.VerificationKeyWitness `cbor:"0,keyasint,omitempty"`
	NativeScripts      []NativeScript.NativeScript                     `cbor:"1,keyasint,omitempty"`
	BootstrapWitnesses []any                                           `cbor:"2,keyasint,omitempty"`
	PlutusV1Script     []PlutusData.PlutusV1Script                     `cbor:"3,keyasint"`
	PlutusV2Script     []PlutusData.PlutusV2Script                     `cbor:"6,keyasint,omitempty"`
	PlutusData         *PlutusData.PlutusIndefArray                    `cbor:"4,keyasint,omitempty"`
	Redeemer           []Redeemer.Redeemer                             `cbor:"5,keyasint,omitempty"`
}

func (tws *TransactionWitnessSet) MarshalCBOR() ([]byte, error) {
	if len(tws.PlutusV1Script) == 0 && len(tws.Redeemer) > 0 && len(tws.PlutusData) == 0 {
		return cbor.Marshal(WithRedeemerNoScripts{
			VkeyWitnesses:      tws.VkeyWitnesses,
			NativeScripts:      tws.NativeScripts,
			BootstrapWitnesses: tws.BootstrapWitnesses,
			PlutusV1Script:     tws.PlutusV1Script,
			PlutusV2Script:     tws.PlutusV2Script,
			PlutusData:         nil,
			Redeemer:           tws.Redeemer,
		})
	}
	return cbor.Marshal(normaltws{
		VkeyWitnesses:      tws.VkeyWitnesses,
		NativeScripts:      tws.NativeScripts,
		BootstrapWitnesses: tws.BootstrapWitnesses,
		PlutusV1Script:     tws.PlutusV1Script,
		PlutusV2Script:     tws.PlutusV2Script,
		PlutusData:         &tws.PlutusData,
		Redeemer:           tws.Redeemer,
	})

}
