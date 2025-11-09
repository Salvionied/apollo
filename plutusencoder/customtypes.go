package plutusencoder

import (
	"errors"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/PlutusData"
)

// SUPPORT FOR ASSETAMOUNTS AND ASSETIDS
// SAMPLE CBOR
// a140a1401a05f5e100

type PlutusMarshaler interface {
	ToPlutusData() (PlutusData.PlutusData, error)
	FromPlutusData(pd PlutusData.PlutusData, res any) error
}

type Asset map[serialization.CustomBytes]map[serialization.CustomBytes]int64

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
		inner := map[serialization.CustomBytes]int64{}
		for k, v := range innerval {
			x, ok := v.Value.(int64)
			if !ok {
				y, ok := v.Value.(uint64)

				if !ok {
					panic("error: Int Data field is not int64")
				}
				x = int64(y)
			}

			inner[k] = x
		}
		assets[key] = inner
	}
	return assets
}

func GetAddressPlutusData(
	address Address.Address,
) (*PlutusData.PlutusData, error) {
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
		return nil, errors.New("error: Pointer Addresses are not supported")
	}
}

func DecodePlutusAddress(
	data PlutusData.PlutusData,
	network byte,
) (Address.Address, error) {
	if data.PlutusDataType != PlutusData.PlutusArray || data.TagNr != 121 {
		return Address.Address{}, errors.New("error: Invalid Address Data")
	}
	var isIndef bool
	switch data.Value.(type) {
	case PlutusData.PlutusDefArray:
		isIndef = false
	case PlutusData.PlutusIndefArray:
		isIndef = true
	default:
		return Address.Address{}, errors.New("error: Invalid Address Data")
	}
	if isIndef {
		arr := data.Value.(PlutusData.PlutusIndefArray)
		if len(arr) < 1 || len(arr) > 2 {
			return Address.Address{}, errors.New("error: Invalid Address Data")
		}
		pkh := arr[0].Value.(PlutusData.PlutusIndefArray)[0].Value.([]byte)
		is_script := arr[0].TagNr == 122
		skh := []byte{}
		skh_exists := len(arr) == 2
		is_skh_script := false
		if skh_exists {
			if len(arr[1].Value.(PlutusData.PlutusIndefArray)) == 0 {
				skh_exists = false
			} else {
				is_skh_script = arr[1].Value.(PlutusData.PlutusIndefArray)[0].TagNr == 122
				skh = arr[1].Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].Value.(PlutusData.PlutusIndefArray)[0].Value.([]byte)
			}
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
		arr := data.Value.(PlutusData.PlutusDefArray)
		if len(arr) < 1 || len(arr) > 2 {
			return Address.Address{}, errors.New("error: Invalid Address Data")
		}
		pkh := arr[0].Value.(PlutusData.PlutusDefArray)[0].Value.([]byte)
		is_script := arr[0].TagNr == 122
		skh := []byte{}
		skh_exists := len(arr) == 2
		is_skh_script := false
		if skh_exists {
			if len(arr[1].Value.(PlutusData.PlutusDefArray)) == 0 {
				skh_exists = false
			} else {
				is_skh_script = arr[1].Value.(PlutusData.PlutusDefArray)[0].TagNr == 122
				skh = arr[1].Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].Value.(PlutusData.PlutusDefArray)[0].Value.([]byte)
			}
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
