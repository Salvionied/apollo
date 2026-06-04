package plutusencoder

import (
	"testing"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/PlutusData"
)

func TestDecodePlutusAssetERejectsInvalidQuantity(t *testing.T) {
	pd := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusMap,
		Value: map[serialization.CustomBytes]PlutusData.PlutusData{
			serialization.NewCustomBytes("policy"): {
				PlutusDataType: PlutusData.PlutusMap,
				Value: map[serialization.CustomBytes]PlutusData.PlutusData{
					serialization.NewCustomBytes("asset"): {
						PlutusDataType: PlutusData.PlutusBytes,
						Value:          []byte("not an int"),
					},
				},
			},
		},
	}

	if _, err := DecodePlutusAssetE(pd); err == nil {
		t.Fatal("expected invalid asset quantity error")
	}
}

func TestDecodePlutusAssetCompatibilityWrapperDoesNotPanic(t *testing.T) {
	pd := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusBytes,
		Value:          []byte("not a map"),
	}

	_ = DecodePlutusAsset(pd)
}
