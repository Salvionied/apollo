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

func DecodePlutusAddress(data PlutusData.PlutusData, network byte) (Address.Address, error) {
	if data.PlutusDataType != PlutusData.PlutusArray && data.TagNr != 121 && len(data.Value.(PlutusData.PlutusIndefArray)) != 2 {
		return Address.Address{}, fmt.Errorf("error: Invalid Address Data")
	}
	var isIndef bool
	switch data.Value.(type) {
	case PlutusData.PlutusDefArray:
		isIndef = false
	case PlutusData.PlutusIndefArray:
		isIndef = true
	default:
		return Address.Address{}, fmt.Errorf("error: Invalid Address Data")
	}
	if isIndef {
		pkh := data.Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].Value.([]byte)
		is_script := data.Value.(PlutusData.PlutusIndefArray)[0].TagNr == 122
		skh := []byte{}
		skh_exists := data.Value.(PlutusData.PlutusIndefArray)[1].TagNr == 121
		is_skh_script := false
		if skh_exists {
			is_skh_script = data.Value.(PlutusData.PlutusIndefArray)[1].Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].TagNr == 122
			skh = data.Value.(PlutusData.PlutusIndefArray)[1].Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].Value.([]byte)
		}
		var addrType byte
		if is_script {
			if skh_exists {
				if is_skh_script {
					addrType = Address.SCRIPT_SCRIPT
				} else {
					addrType = Address.SCRIPT_KEY
				}
			} else {
				addrType = Address.SCRIPT_NONE
			}
		} else {
			if skh_exists {
				if is_skh_script {
					addrType = Address.KEY_SCRIPT
				} else {
					addrType = Address.KEY_KEY
				}
			} else {
				addrType = Address.KEY_NONE
			}
		}
		hrp := Address.ComputeHrp(addrType, network)
		header := addrType<<4 | network
		addr := Address.Address{
			PaymentPart: pkh,
			StakingPart: skh,
			AddressType: addrType,
			Network:     network,
			HeaderByte:  header,
			Hrp:         hrp}
		return addr, nil
	} else {
		pkh := data.Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].Value.([]byte)
		is_script := data.Value.(PlutusData.PlutusDefArray)[0].TagNr == 122
		skh := []byte{}
		skh_exists := data.Value.(PlutusData.PlutusDefArray)[1].TagNr == 121
		is_skh_script := false
		if skh_exists {
			is_skh_script = data.Value.(PlutusData.PlutusDefArray)[1].Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].TagNr == 122
			skh = data.Value.(PlutusData.PlutusDefArray)[1].Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].Value.([]byte)
		}
		var addrType byte
		if is_script {
			if skh_exists {
				if is_skh_script {
					addrType = Address.SCRIPT_SCRIPT
				} else {
					addrType = Address.SCRIPT_KEY
				}
			} else {
				addrType = Address.SCRIPT_NONE
			}
		} else {
			if skh_exists {
				if is_skh_script {
					addrType = Address.KEY_SCRIPT
				} else {
					addrType = Address.KEY_KEY
				}
			} else {
				addrType = Address.KEY_NONE
			}
		}
		hrp := Address.ComputeHrp(addrType, network)
		header := addrType<<4 | network
		addr := Address.Address{
			PaymentPart: pkh,
			StakingPart: skh,
			AddressType: addrType,
			Network:     network,
			HeaderByte:  header,
			Hrp:         hrp}
		return addr, nil
	}
}
