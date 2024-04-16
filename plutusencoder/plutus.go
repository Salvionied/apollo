package plutusencoder

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/cbor/v2"
)

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

func MarshalPlutus(v interface{}) (*PlutusData.PlutusData, error) {
	var overallContainer interface{}
	var containerConstr = uint64(0)
	var isMap = false
	var isIndef = true
	types := reflect.TypeOf(v)
	values := reflect.ValueOf(v)
	//get Container type
	ok := types.Kind() == reflect.Struct
	if ok {
		fields, _ := types.FieldByName("_")
		typeOfStruct := fields.Tag.Get("plutusType")
		if typeOfStruct == "Map" {
			isMap = true
		}
		Constr := fields.Tag.Get("plutusConstr")
		if Constr != "" {
			parsedConstr, err := strconv.Atoi(Constr)
			if err != nil {
				return nil, fmt.Errorf("error parsing constructor: %v", err)
			}
			if parsedConstr < 7 {
				containerConstr = 121 + uint64(parsedConstr)
			} else if 7 <= parsedConstr && parsedConstr <= 1400 {
				containerConstr = 1280 + uint64(parsedConstr-7)
			} else {
				return nil, fmt.Errorf("parsedConstr value is above 1400")
			}
		}
		switch typeOfStruct {
		case "IndefList":
			overallContainer = PlutusData.PlutusIndefArray{}
		case "Map":
			overallContainer = map[serialization.CustomBytes]PlutusData.PlutusData{}
			isMap = true
		case "DefList":
			overallContainer = PlutusData.PlutusDefArray{}
			isIndef = false
		default:
			fmt.Println(typeOfStruct)
			return nil, fmt.Errorf("error: unknown type")
		}
		//get fields
		for i := 0; i < types.NumField(); i++ {
			f := types.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag
			name := f.Name
			constr := uint64(0)
			typeOfField := tag.Get("plutusType")
			constrOfField := tag.Get("plutusConstr")
			if constrOfField != "" {
				parsedConstr, err := strconv.Atoi(constrOfField)
				if err != nil {
					return nil, fmt.Errorf("error parsing constructor: %v", err)
				}
				if parsedConstr < 7 {
					constr = 121 + uint64(parsedConstr)
				} else if 7 <= parsedConstr && parsedConstr <= 1400 {
					constr = 1280 + uint64(parsedConstr-7)
				} else {
					return nil, fmt.Errorf("parsedConstr value is above 1400")
				}
			}
			switch typeOfField {
			case "Bool":
				if values.Field(i).Kind() != reflect.Bool {
					return nil, fmt.Errorf("error: Bool field is not bool")
				}
				var boolPD PlutusData.PlutusData
				switch values.Field(i).Bool() {
				case true:
					boolPD = PlutusData.PlutusData{
						TagNr:          122,
						PlutusDataType: PlutusData.PlutusArray,
						Value:          PlutusData.PlutusDefArray{},
					}
				case false:
					boolPD = PlutusData.PlutusData{
						TagNr:          121,
						PlutusDataType: PlutusData.PlutusArray,
						Value:          PlutusData.PlutusDefArray{},
					}
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = boolPD
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), boolPD)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), boolPD)
					}
				}
			case "Bytes":
				if values.Field(i).Kind() != reflect.Slice {
					return nil, fmt.Errorf("error: Bytes field is not a slice")
				}
				pdb := PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusBytes,
					Value:          values.Field(i).Interface().([]byte),
					TagNr:          constr,
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = pdb
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), pdb)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), pdb)
					}
				}
			case "Int":
				if values.Field(i).Kind() != reflect.Int64 {
					return nil, fmt.Errorf("error: Int field is not int64")
				}
				pdi := PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusInt,
					Value:          values.Field(i).Interface().(int64),
					TagNr:          constr,
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)

					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = pdi
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), pdi)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), pdi)
					}
				}
			case "StringBytes":
				if values.Field(i).Kind() != reflect.String {
					return nil, fmt.Errorf("error: StringBytes field is not string")
				}
				pdsb := PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusBytes,
					Value:          []byte(values.Field(i).Interface().(string)),
					TagNr:          constr,
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = pdsb
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), pdsb)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), pdsb)
					}
				}
			case "HexString":
				if values.Field(i).Kind() != reflect.String {
					return nil, fmt.Errorf("error: HexString field is not string")
				}
				hexString, err := hex.DecodeString(values.Field(i).Interface().(string))
				if err != nil {
					return nil, fmt.Errorf("error: HexString field is not string")
				}
				pdsb := PlutusData.PlutusData{
					PlutusDataType: PlutusData.PlutusBytes,
					Value:          hexString,
					TagNr:          constr,
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = pdsb
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), pdsb)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), pdsb)
					}
				}
			case "Address":
				addpd, err := GetAddressPlutusData(values.Field(i).Interface().(Address.Address))
				if err != nil {
					return nil, fmt.Errorf("error marshalling: %v", err)
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = *addpd
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), *addpd)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), *addpd)
					}
				}

			case "IndefList":
				container := PlutusData.PlutusIndefArray{}
				for j := 0; j < values.Field(i).Len(); j++ {
					pd, err := MarshalPlutus(values.Field(i).Index(j).Interface())
					if err != nil {
						return nil, fmt.Errorf("error marshalling: %v", err)
					}
					container = append(container, *pd)
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusArray,
						Value:          container,
						TagNr:          constr,
					}
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), PlutusData.PlutusData{
							PlutusDataType: PlutusData.PlutusArray,
							Value:          container,
							TagNr:          constr,
						})
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), PlutusData.PlutusData{
							PlutusDataType: PlutusData.PlutusArray,
							Value:          container,
							TagNr:          constr,
						})

					}
				}
			case "DefList":
				container := PlutusData.PlutusDefArray{}
				for j := 0; j < values.Field(i).Len(); j++ {
					pd, err := MarshalPlutus(values.Field(i).Index(j).Interface())
					if err != nil {
						return nil, fmt.Errorf("error marshalling: %v", err)
					}
					container = append(container, *pd)
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusArray,
						Value:          container,
						TagNr:          constr,
					}
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), PlutusData.PlutusData{
							PlutusDataType: PlutusData.PlutusArray,
							Value:          container,
							TagNr:          constr,
						})
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), PlutusData.PlutusData{
							PlutusDataType: PlutusData.PlutusArray,
							Value:          container,
							TagNr:          constr,
						})
					}
				}
			case "Map":
				container := map[serialization.CustomBytes]PlutusData.PlutusData{}
				for j := 0; j < values.Field(i).Len(); j++ {
					pd, err := MarshalPlutus(values.Field(i).Index(j).Interface())
					if err != nil {
						return nil, fmt.Errorf("error marshalling: %v", err)
					}
					nameBytes := serialization.NewCustomBytes(values.Field(i).Index(j).Field(0).String())
					container[nameBytes] = *pd
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)
					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = PlutusData.PlutusData{
						PlutusDataType: PlutusData.PlutusMap,
						Value:          container,
						TagNr:          constr,
					}
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), PlutusData.PlutusData{
							PlutusDataType: PlutusData.PlutusMap,
							Value:          container,
							TagNr:          constr,
						})
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), PlutusData.PlutusData{
							PlutusDataType: PlutusData.PlutusMap,
							Value:          container,
							TagNr:          constr,
						})
					}
				}

			default:
				pd, err := MarshalPlutus(values.Field(i).Interface())
				if err != nil {
					return nil, fmt.Errorf("error marshalling: %v", err)
				}
				if isMap {
					nameBytes := serialization.NewCustomBytes(name)

					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = *pd
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), *pd)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), *pd)
					}
				}
			}
		}

	}
	if !ok {
		switch types.Kind() {
		case reflect.String:
			return &PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          []byte(values.Interface().(string)),
				TagNr:          containerConstr,
			}, nil
		case reflect.Int:
			return &PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusInt,
				Value:          values.Interface().(int),
				TagNr:          containerConstr,
			}, nil
		case reflect.Slice:
			return &PlutusData.PlutusData{
				PlutusDataType: PlutusData.PlutusBytes,
				Value:          values.Interface().([]byte),
				TagNr:          containerConstr,
			}, nil
		default:
			return nil, fmt.Errorf("error: unknown type")
		}
	}
	ptype := PlutusData.PlutusArray
	if isMap {
		ptype = PlutusData.PlutusMap
	}
	pd := PlutusData.PlutusData{
		PlutusDataType: ptype,
		Value:          overallContainer,
		TagNr:          containerConstr,
	}
	return &pd, nil
}

func CborUnmarshal(data string, v interface{}, network byte) error {
	decoded, err := hex.DecodeString(data)
	if err != nil {
		return fmt.Errorf("error decoding hex: %v", err)
	}
	pd := PlutusData.PlutusData{}
	err = cbor.Unmarshal(decoded, &pd)
	if err != nil {
		return fmt.Errorf("error unmarshalling: %v", err)
	}
	err = UnmarshalPlutus(&pd, v, network)
	if err != nil {
		return fmt.Errorf("error unmarshalling: %v", err)
	}
	return nil
}

func UnmarshalPlutus(data *PlutusData.PlutusData, v interface{}, network byte) (ret error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in UnmarshalPlutus", r)
			ret = errors.New("error unmarshalling")
		}
	}()

	ret = unmarshalPlutus(data, v, data.TagNr, data.PlutusDataType, network)
	return ret
}

func DecodePlutusAddress(data PlutusData.PlutusData, network byte) Address.Address {
	if data.PlutusDataType != PlutusData.PlutusArray && data.TagNr != 121 && len(data.Value.(PlutusData.PlutusIndefArray)) != 2 {
		return Address.Address{}
	}
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
	return addr
}

func unmarshalPlutus(data *PlutusData.PlutusData, v interface{}, Plutusconstr uint64, PlutusType PlutusData.PlutusType, network byte) error {
	types := reflect.TypeOf(v)
	if types.Kind() != reflect.Ptr {
		return fmt.Errorf("error: v is not a pointer %v", v)
	}
	constr := data.TagNr
	//get Container type
	tps := types.Elem()
	//values := reflect.ValueOf(tps)
	//isStruct := tps.Kind() == reflect.Struct
	ok := tps.Kind() == reflect.Struct
	if ok {
		fields, _ := tps.FieldByName("_")
		switch data.PlutusDataType {
		case PlutusData.PlutusArray:
			if reflect.TypeOf(v).Kind() != reflect.Ptr {
				return fmt.Errorf("error: v is not a pointer")
			}
			if fields.Tag.Get("plutusType") != "IndefList" && fields.Tag.Get("plutusType") != "DefList" && fields.Tag.Get("plutusType") != "" {
				return fmt.Errorf("error: v is not a PlutusList")
			}
			plutusConstr := fields.Tag.Get("plutusConstr")
			if constr != 0 && constr > 1400 && (plutusConstr != fmt.Sprint(constr-121) || plutusConstr != fmt.Sprint(constr-1280)) {
				return fmt.Errorf("error: constructorTag does not match, got %s, expected %d", plutusConstr, constr)
			}

			arrayType := reflect.TypeOf(data.Value).String()
			switch arrayType {
			case "PlutusData.PlutusDefArray":
				plutusValues, ok := data.Value.(PlutusData.PlutusDefArray)
				if !ok {
					return fmt.Errorf("error: value is not a PlutusDefArray")
				}
				for idx, pAEl := range plutusValues {
					if tps.Field(idx+1).Type.String() == "Address.Address" {
						addr := DecodePlutusAddress(pAEl, network)
						reflect.ValueOf(v).Elem().Field(idx + 1).Set(reflect.ValueOf(addr))
						continue
					}
					if tps.Field(idx+1).Type.String() == "bool" {
						if tps.Field(idx+1).Type.String() != "bool" {
							return fmt.Errorf("error: Bool field is not bool")
						}
						reflect.ValueOf(v).Elem().Field(idx + 1).SetBool(pAEl.TagNr == 122)
						continue
					}
					switch pAEl.PlutusDataType {
					case PlutusData.PlutusBytes:
						if tps.Field(idx+1).Type.String() != "[]uint8" {
							if tps.Field(idx+1).Type.String() != "string" {
								return fmt.Errorf("error: Bytes field is not a slice")
							} else {
								if reflect.TypeOf(v).Elem().Field(idx+1).Tag.Get("plutusType") == "HexString" {
									reflect.ValueOf(v).Elem().Field(idx + 1).SetString(hex.EncodeToString(pAEl.Value.([]byte)))
									continue
								}
								reflect.ValueOf(v).Elem().Field(idx + 1).SetString(string(pAEl.Value.([]byte)))
								continue
							}
						}
						reflect.ValueOf(v).Elem().Field(idx + 1).Set(reflect.ValueOf(pAEl.Value))
					case PlutusData.PlutusInt:
						if tps.Field(idx+1).Type.String() != "int64" {
							return fmt.Errorf("error: Int field is not int64")
						}
						x, ok := pAEl.Value.(uint64)
						if !ok {
							return fmt.Errorf("error: Int field is not int64")
						}

						reflect.ValueOf(v).Elem().Field(idx + 1).SetInt(int64(x))
					case PlutusData.PlutusArray:
						if reflect.TypeOf(v).Elem().Field(idx+1).Type.Kind() == reflect.Slice {
							pa, ok := pAEl.Value.(PlutusData.PlutusIndefArray)
							if ok {
								val := reflect.ValueOf(v).Elem().Field(idx + 1)
								val.Grow(len(pa))
								val.SetLen(len(pa))
								for secIdx, arrayElement := range pa {
									err := unmarshalPlutus(&arrayElement, val.Index(secIdx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
									if err != nil {
										return fmt.Errorf("error at index %d.%d: %v:", idx, secIdx, err)
									}
								}
								reflect.ValueOf(v).Elem().Field(idx + 1).Set(val)
							} else {
								pa2, ok := pAEl.Value.(PlutusData.PlutusDefArray)
								if !ok {
									return fmt.Errorf("error: value is not a PlutusArray")
								}
								val2 := reflect.ValueOf(v).Elem().Field(idx + 1)
								val2.Grow(len(pa2))
								val2.SetLen(len(pa2))
								for secIdx, arrayElement := range pa2 {
									err := unmarshalPlutus(&arrayElement, val2.Index(secIdx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
									if err != nil {
										return fmt.Errorf("error at index %d.%d: %v:", idx, secIdx, err)
									}
								}
								reflect.ValueOf(v).Elem().Field(idx + 1).Set(val2)
							}
						} else {
							err := unmarshalPlutus(&pAEl, reflect.ValueOf(v).Elem().Field(idx+1).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
							if err != nil {
								return fmt.Errorf("error at index %d: %v", idx, err)
							}
						}
					case PlutusData.PlutusMap:
						err := unmarshalPlutus(&pAEl, reflect.ValueOf(v).Elem().Field(idx+1).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
						if err != nil {
							return fmt.Errorf("error at index %d: %v", idx, err)
						}
					default:
						return fmt.Errorf("error: unknown type")
					}
				}
			case "PlutusData.PlutusIndefArray":
				plutusValues, ok := data.Value.(PlutusData.PlutusIndefArray)
				if !ok {
					return fmt.Errorf("error: value is not a PlutusIndefArray")
				}
				for idx, pAEl := range plutusValues {
					if tps.Field(idx+1).Type.String() == "Address.Address" {
						addr := DecodePlutusAddress(pAEl, network)
						reflect.ValueOf(v).Elem().Field(idx + 1).Set(reflect.ValueOf(addr))
						continue
					}
					if tps.Field(idx+1).Type.String() == "bool" {
						if tps.Field(idx+1).Type.String() != "bool" {
							return fmt.Errorf("error: Bool field is not bool")
						}
						reflect.ValueOf(v).Elem().Field(idx + 1).SetBool(pAEl.TagNr == 122)
						continue
					}
					switch pAEl.PlutusDataType {
					case PlutusData.PlutusBytes:
						if tps.Field(idx+1).Type.String() != "[]uint8" {
							if tps.Field(idx+1).Type.String() != "string" {
								return fmt.Errorf("error: Bytes field is not a slice")
							} else {
								if reflect.TypeOf(v).Elem().Field(idx+1).Tag.Get("plutusType") == "HexString" {
									reflect.ValueOf(v).Elem().Field(idx + 1).SetString(hex.EncodeToString(pAEl.Value.([]byte)))
									continue
								}
								reflect.ValueOf(v).Elem().Field(idx + 1).SetString(string(pAEl.Value.([]byte)))
								continue
							}
						}
						reflect.ValueOf(v).Elem().Field(idx + 1).Set(reflect.ValueOf(pAEl.Value))
					case PlutusData.PlutusInt:
						if tps.Field(idx+1).Type.String() != "int64" {
							return fmt.Errorf("error: Int field is not int64")
						}
						x, ok := pAEl.Value.(uint64)
						if !ok {
							return fmt.Errorf("error: Int field is not int64")
						}

						reflect.ValueOf(v).Elem().Field(idx + 1).SetInt(int64(x))
					case PlutusData.PlutusArray:
						if reflect.TypeOf(v).Elem().Field(idx+1).Type.Kind() == reflect.Slice {
							pa, ok := pAEl.Value.(PlutusData.PlutusIndefArray)
							if ok {
								val := reflect.ValueOf(v).Elem().Field(idx + 1)
								val.Grow(len(pa))
								val.SetLen(len(pa))
								for secIdx, arrayElement := range pa {
									err := unmarshalPlutus(&arrayElement, val.Index(secIdx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
									if err != nil {
										return fmt.Errorf("error at index %d.%d: %v:", idx, secIdx, err)
									}
								}
								reflect.ValueOf(v).Elem().Field(idx + 1).Set(val)
							} else {
								pa2, ok := pAEl.Value.(PlutusData.PlutusDefArray)
								if !ok {
									return fmt.Errorf("error: value is not a PlutusArray")
								}
								val2 := reflect.ValueOf(v).Elem().Field(idx + 1)
								val2.Grow(len(pa2))
								val2.SetLen(len(pa2))
								for secIdx, arrayElement := range pa2 {
									err := unmarshalPlutus(&arrayElement, val2.Index(secIdx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
									if err != nil {
										return fmt.Errorf("error at index %d.%d: %v:", idx, secIdx, err)
									}
								}
								reflect.ValueOf(v).Elem().Field(idx + 1).Set(val2)
							}
						} else {
							err := unmarshalPlutus(&pAEl, reflect.ValueOf(v).Elem().Field(idx+1).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
							if err != nil {
								return fmt.Errorf("error at index %d: %v", idx, err)
							}
						}
					case PlutusData.PlutusMap:
						err := unmarshalPlutus(&pAEl, reflect.ValueOf(v).Elem().Field(idx+1).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
						if err != nil {
							return fmt.Errorf("error at index %d: %v", idx, err)
						}
					default:
						return fmt.Errorf("error: unknown type")
					}
				}
			default:
				return fmt.Errorf("error: unknown type")
			}
		case PlutusData.PlutusMap:
			values, ok := data.Value.(*map[serialization.CustomBytes]PlutusData.PlutusData)
			if !ok {
				return fmt.Errorf("error: value is not a PlutusMap")
			}
			for idxStringHex, pAEl := range *values {
				idxBytes, _ := hex.DecodeString(idxStringHex.String())
				idx := string(idxBytes)
				field, ok := tps.FieldByName(idx)
				if !ok {
					return fmt.Errorf("error: field %s does not exist", idx)
				}
				switch field.Type.String() {
				case "Address.Address":
					addr := DecodePlutusAddress(pAEl, network)
					reflect.ValueOf(v).Elem().FieldByName(idx).Set(reflect.ValueOf(addr))
					continue
				case "bool":
					reflect.ValueOf(v).Elem().FieldByName(idx).SetBool(pAEl.TagNr == 122)
					continue
				case "[]uint8":
					if pAEl.PlutusDataType != PlutusData.PlutusBytes {
						return fmt.Errorf("error: Bytes field is not a slice")
					}
					reflect.ValueOf(v).Elem().FieldByName(idx).Set(reflect.ValueOf(pAEl.Value))
				case "string":
					if pAEl.PlutusDataType != PlutusData.PlutusBytes {
						return fmt.Errorf("error: Bytes field is not a slice")
					}
					tp, _ := reflect.TypeOf(v).Elem().FieldByName(idx)
					if tp.Tag.Get("plutusType") == "HexString" {
						reflect.ValueOf(v).Elem().FieldByName(string(idx)).SetString(hex.EncodeToString(pAEl.Value.([]byte)))
						continue
					}
					reflect.ValueOf(v).Elem().FieldByName(idx).SetString(string(pAEl.Value.([]byte)))
				case "int64":
					if pAEl.PlutusDataType != PlutusData.PlutusInt {
						return fmt.Errorf("error: Int field is not int64")
					}
					x, ok := pAEl.Value.(uint64)
					if !ok {
						return fmt.Errorf("error: Int field is not int64")
					}
					reflect.ValueOf(v).Elem().FieldByName(idx).SetInt(int64(x))
				default:
					switch pAEl.PlutusDataType {
					case PlutusData.PlutusArray:
						tp, _ := reflect.TypeOf(v).Elem().FieldByName(idx)

						switch tp.Tag.Get("plutusType") {
						case "IndefList":
							pa, ok := pAEl.Value.(PlutusData.PlutusIndefArray)
							if !ok {
								return fmt.Errorf("error: value is not a PlutusArray")
							}
							val := reflect.ValueOf(v).Elem().FieldByName(idx)
							val.Grow(len(pa))
							val.SetLen(len(pa))
							for secIdx, arrayElement := range pa {
								err := unmarshalPlutus(&arrayElement, val.Index(secIdx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
								if err != nil {
									return fmt.Errorf("error at index %s.%d: %v:", idx, secIdx, err)
								}
							}
							reflect.ValueOf(v).Elem().FieldByName(idx).Set(val)
						case "DefList":
							pa, ok := pAEl.Value.(PlutusData.PlutusDefArray)
							if !ok {
								return fmt.Errorf("error: value is not a PlutusArray")
							}
							val := reflect.ValueOf(v).Elem().FieldByName(idx)
							val.Grow(len(pa))
							val.SetLen(len(pa))
							for secIdx, arrayElement := range pa {
								err := unmarshalPlutus(&arrayElement, val.Index(secIdx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
								if err != nil {
									return fmt.Errorf("error at index %s.%d: %v:", idx, secIdx, err)
								}
							}
							reflect.ValueOf(v).Elem().FieldByName(idx).Set(val)
						}
					case PlutusData.PlutusMap:
						err := unmarshalPlutus(&pAEl, reflect.ValueOf(v).Elem().FieldByName(idx).Addr().Interface(), pAEl.TagNr, pAEl.PlutusDataType, network)
						if err != nil {
							return fmt.Errorf("error at index %s: %v", idx, err)
						}
					}
				}
			}
		default:
			return fmt.Errorf("error: unknown type")
		}
	} else {
		if types.Kind() == reflect.Ptr {
			types = types.Elem()
		}
		switch types.Kind() {
		case reflect.String:
			if data.PlutusDataType != PlutusData.PlutusBytes {
				return fmt.Errorf("error: Bytes field is not a slice")
			}
			if reflect.TypeOf(v).Kind() != reflect.Ptr {
				return fmt.Errorf("error: v is not a pointer")
			}
			if reflect.TypeOf(v).Elem().Kind() != reflect.String {
				return fmt.Errorf("error: v is not a string")
			}
			reflect.ValueOf(v).Elem().SetString(string(data.Value.([]byte)))
		case reflect.Int:
			if data.PlutusDataType != PlutusData.PlutusInt {
				return fmt.Errorf("error: Int field is not int64")
			}
			if reflect.TypeOf(v).Kind() != reflect.Ptr {
				return fmt.Errorf("error: v is not a pointer")
			}
			if reflect.TypeOf(v).Elem().Kind() != reflect.Int {
				return fmt.Errorf("error: v is not an int")
			}
			reflect.ValueOf(v).Elem().SetInt(int64(data.Value.(uint64)))
		case reflect.Slice:
			if data.PlutusDataType != PlutusData.PlutusBytes {
				return fmt.Errorf("error: Bytes field is not a slice")
			}
			if reflect.TypeOf(v).Kind() != reflect.Ptr {
				return fmt.Errorf("error: v is not a pointer")
			}
			if reflect.TypeOf(v).Elem().Kind() != reflect.Slice {
				return fmt.Errorf("error: v is not a slice")
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(data.Value))
		default:
			return fmt.Errorf("error: unknown type")
		}
	}

	return nil
}
