package PlutusData_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/blinklabs-io/gouroboros/cbor"
)

func TestSerializeAndDeserializePlutusData(t *testing.T) {
	cborHex := "18e9"
	decoded_cbor, _ := hex.DecodeString(cborHex)
	var pd PlutusData.PlutusData
	_, err := cbor.Decode(decoded_cbor, &pd)
	if err != nil {
		t.Error("Failed unmarshaling", err)
	}
	marshaled, _ := cbor.Encode(pd)
	if hex.EncodeToString(marshaled) != cborHex {
		t.Error(
			"Invalid marshaling",
			hex.EncodeToString(marshaled),
			"Expected",
			cborHex,
		)
	}
}

// Helper function used by other tests in repo; kept minimal and corrected.
func GetMinSwapPlutusData() PlutusData.PlutusData {
	SkhStruct := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		TagNr:          121,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusArray,
				TagNr:          121,
				Value: PlutusData.PlutusIndefArray{
					PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusArray,
						TagNr:          121,
						Value: PlutusData.PlutusIndefArray{
							PlutusData.PlutusData{
								PlutusDataType: PlutusData.PlutusBytes,
								TagNr:          0,
								Value:          []byte{},
							},
						},
					},
				},
			},
		},
	}

	pkhStruct := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		TagNr:          121,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusArray,
				TagNr:          121,
				Value: PlutusData.PlutusIndefArray{
					PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusBytes,
						TagNr:          0,
						Value:          []byte{},
					},
				},
			},
			SkhStruct,
		},
	}

	policy_bytes := []byte{}
	asset_bytes := []byte{}
	AssetStruct := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		Value: PlutusData.PlutusIndefArray{
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusBytes,
				TagNr:          0,
				Value:          policy_bytes,
			},
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusBytes,
				TagNr:          0,
				Value:          asset_bytes,
			},
		},
		TagNr: 121,
	}
	BuyOrderStruct := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		Value: PlutusData.PlutusIndefArray{
			AssetStruct,
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusInt,
				TagNr:          0,
				Value:          int64(0),
			},
		},
		TagNr: 121,
	}

	Fee := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusInt,
		TagNr:          0,
		Value:          int64(2000000),
	}
	Bribe := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusInt,
		TagNr:          0,
		Value:          int64(0),
	}

	FullStruct := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		Value: PlutusData.PlutusIndefArray{
			pkhStruct,
			pkhStruct,
			PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusArray,
				TagNr:          122,
				Value:          []PlutusData.PlutusData{},
			},
			BuyOrderStruct,
			Bribe,
			Fee,
		},
		TagNr: 121,
	}

	return FullStruct
}
