package plutusencoder

import (
	"fmt"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/PlutusData"
)

// SUPPORT FOR ASSETAMOUNTS AND ASSETIDS
// SAMPLE CBOR
// a140a1401a05f5e100

type PlutusMarshaler interface {
	ToPlutusData() (PlutusData.PlutusData, error)
	FromPlutusData(pd PlutusData.PlutusData, res interface{}) error
}

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

// / ADDRESS SUPPORT
func DecodePlutusAsset(pd PlutusData.PlutusData) Asset {
	assets := Asset{}
	val, _ := pd.Value.(map[serialization.CustomBytes]PlutusData.PlutusData)
	for key, asset := range val {
		innerval, _ := asset.Value.(map[serialization.CustomBytes]PlutusData.PlutusData)
		inner := map[serialization.CustomBytes]uint64{}
		for k, v := range innerval {
			inner[k] = v.Value.(uint64)
		}
		assets[key] = inner
	}
	return assets
}
func GetAddressPlutusData(address Address.Address) (*PlutusData.PlutusData, error) {
	switch address.AddressType {
	case Address.KEY_KEY:
		return &PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusArray,
			Value: PlutusData.PlutusIndefArray{
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          0,
							Value:          address.PaymentPart,
							PlutusDataType: PlutusData.PlutusBytes,
						},
					},
				},
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          121,
							PlutusDataType: PlutusData.PlutusArray,
							Value: PlutusData.PlutusIndefArray{
								PlutusData.PlutusData{
									TagNr:          121,
									PlutusDataType: PlutusData.PlutusArray,
									Value: PlutusData.PlutusIndefArray{
										PlutusData.PlutusData{
											TagNr:          0,
											Value:          address.StakingPart,
											PlutusDataType: PlutusData.PlutusBytes},
									},
								},
							},
						},
					},
				},
			},
		}, nil

	case Address.SCRIPT_KEY:
		return &PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusArray,
			Value: PlutusData.PlutusIndefArray{
				PlutusData.PlutusData{
					TagNr:          122,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          0,
							Value:          address.PaymentPart,
							PlutusDataType: PlutusData.PlutusBytes,
						},
					},
				},
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          121,
							PlutusDataType: PlutusData.PlutusArray,
							Value: PlutusData.PlutusIndefArray{
								PlutusData.PlutusData{
									TagNr:          121,
									PlutusDataType: PlutusData.PlutusArray,
									Value: PlutusData.PlutusIndefArray{
										PlutusData.PlutusData{
											TagNr:          0,
											Value:          address.StakingPart,
											PlutusDataType: PlutusData.PlutusBytes},
									},
								},
							},
						},
					},
				},
			},
		}, nil
	case Address.KEY_SCRIPT:
		return &PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusArray,
			Value: PlutusData.PlutusIndefArray{
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          0,
							Value:          address.PaymentPart,
							PlutusDataType: PlutusData.PlutusBytes,
						},
					},
				},
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          121,
							PlutusDataType: PlutusData.PlutusArray,
							Value: PlutusData.PlutusIndefArray{
								PlutusData.PlutusData{
									TagNr:          122,
									PlutusDataType: PlutusData.PlutusArray,
									Value: PlutusData.PlutusIndefArray{
										PlutusData.PlutusData{
											TagNr:          0,
											Value:          address.StakingPart,
											PlutusDataType: PlutusData.PlutusBytes},
									},
								},
							},
						},
					},
				},
			},
		}, nil
	case Address.SCRIPT_SCRIPT:
		return &PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusArray,
			Value: PlutusData.PlutusIndefArray{
				PlutusData.PlutusData{
					TagNr:          122,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          0,
							Value:          address.PaymentPart,
							PlutusDataType: PlutusData.PlutusBytes,
						},
					},
				},
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          121,
							PlutusDataType: PlutusData.PlutusArray,
							Value: PlutusData.PlutusIndefArray{
								PlutusData.PlutusData{
									TagNr:          122,
									PlutusDataType: PlutusData.PlutusArray,
									Value: PlutusData.PlutusIndefArray{
										PlutusData.PlutusData{
											TagNr:          0,
											Value:          address.StakingPart,
											PlutusDataType: PlutusData.PlutusBytes},
									},
								},
							},
						},
					},
				},
			},
		}, nil
	case Address.KEY_NONE:
		return &PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusArray,
			Value: PlutusData.PlutusIndefArray{
				PlutusData.PlutusData{
					TagNr:          121,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          0,
							Value:          address.PaymentPart,
							PlutusDataType: PlutusData.PlutusBytes,
						},
					},
				},
				PlutusData.PlutusData{
					TagNr:          122,
					PlutusDataType: PlutusData.PlutusArray,
					Value:          PlutusData.PlutusIndefArray{},
				},
			},
		}, nil
	case Address.SCRIPT_NONE:
		return &PlutusData.PlutusData{
			TagNr:          121,
			PlutusDataType: PlutusData.PlutusArray,
			Value: PlutusData.PlutusIndefArray{
				PlutusData.PlutusData{
					TagNr:          122,
					PlutusDataType: PlutusData.PlutusArray,
					Value: PlutusData.PlutusIndefArray{
						PlutusData.PlutusData{
							TagNr:          0,
							Value:          address.PaymentPart,
							PlutusDataType: PlutusData.PlutusBytes,
						},
					},
				},
				PlutusData.PlutusData{
					TagNr:          122,
					PlutusDataType: PlutusData.PlutusArray,
					Value:          PlutusData.PlutusIndefArray{},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("error: Pointer Addresses are not supported")
	}
}
