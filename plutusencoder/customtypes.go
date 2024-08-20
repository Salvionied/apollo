package plutusencoder

import (
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/PlutusData"
)

// SUPPORT FOR ASSETAMOUNTS AND ASSETIDS
// SAMPLE CBOR
// a140a1401a05f5e100

type Asset map[serialization.CustomBytes]map[serialization.CustomBytes]uint64

func GetAssetPlutusData(assets Asset) PlutusData.PlutusData {
	outermap := map[serialization.CustomBytes]PlutusData.PlutusData{}
	for key, asset := range assets {
		inner := map[serialization.CustomBytes]PlutusData.PlutusData{}
		for k, v := range asset {
			inner[k] = PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusInt,
				Value:          v,
			}
		}
		outermap[key] = PlutusData.PlutusData{
			PlutusDataType: PlutusData.PlutusMap,
			Value:          inner,
		}
	}
	assetData := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusMap,
		Value:          outermap,
	}
	return assetData
}

func DecodePlutusAsset(pd PlutusData.PlutusData) Asset {
	assets := Asset{}
	val, _ := pd.Value.(*map[serialization.CustomBytes]PlutusData.PlutusData)
	for key, asset := range *val {
		innerval, _ := asset.Value.(*map[serialization.CustomBytes]PlutusData.PlutusData)
		inner := map[serialization.CustomBytes]uint64{}
		for k, v := range *innerval {
			inner[k] = v.Value.(uint64)
		}
		assets[key] = inner
	}
	return assets
}
