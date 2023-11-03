package plutusencoder

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/PlutusData"
)

func MarshalPlutus(v interface{}) (*PlutusData.PlutusData, error) {
	var overallContainer interface{}
	var containerConstr = uint64(0)
	var isMap = false
	var isIndef = true
	types := reflect.TypeOf(v)
	values := reflect.ValueOf(v)
	//get Container type
	fields, ok := types.FieldByName("_")
	if ok {
		typeOfStruct := fields.Tag.Get("plutusType")
		Constr := fields.Tag.Get("plutusConstr")
		if Constr != "" {
			parsedConstr, err := strconv.Atoi(Constr)
			if err != nil {
				return nil, fmt.Errorf("error parsing constructor: %v", err)
			}
			containerConstr = 121 + uint64(parsedConstr)
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
			overallContainer = PlutusData.PlutusIndefArray{}
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
				constr = 121 + uint64(parsedConstr)
			}
			switch typeOfField {
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
					nameBytes := serialization.CustomBytes{Value: name}
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
					nameBytes := serialization.CustomBytes{Value: name}

					overallContainer.(map[serialization.CustomBytes]PlutusData.PlutusData)[nameBytes] = pdi
				} else {
					if isIndef {
						overallContainer = append(overallContainer.(PlutusData.PlutusIndefArray), pdi)
					} else {
						overallContainer = append(overallContainer.(PlutusData.PlutusDefArray), pdi)
					}
				}
			default:
				pd, err := MarshalPlutus(values.Field(i).Interface())
				if err != nil {
					return nil, fmt.Errorf("error marshalling: %v", err)
				}
				if isMap {
					nameBytes := serialization.CustomBytes{Value: name}

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
	pd := PlutusData.PlutusData{
		PlutusDataType: PlutusData.PlutusArray,
		Value:          &overallContainer,
		TagNr:          containerConstr,
	}
	return &pd, nil
}

func UnmarshalPlutus(data *PlutusData.PlutusData, v interface{}) error {
	//TODO
	return nil
}
