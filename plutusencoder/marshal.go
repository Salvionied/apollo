package plutusencoder

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/blinklabs-io/plutigo/data"
)

// MarshalPlutus encodes a Go struct to PlutusData using struct tags.
func MarshalPlutus(v any) (data.PlutusData, error) {
	return marshalValue(reflect.ValueOf(v))
}

func marshalValue(val reflect.Value) (data.PlutusData, error) {
	// Dereference pointers
	for val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, errors.New("nil pointer")
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("MarshalPlutus requires a struct, got %s", val.Kind())
	}

	// Check if the type implements PlutusMarshaler (pointer or value receiver)
	if val.CanAddr() {
		if m, ok := val.Addr().Interface().(PlutusMarshaler); ok {
			return m.ToPlutusData()
		}
	}
	if m, ok := val.Interface().(PlutusMarshaler); ok {
		return m.ToPlutusData()
	}

	typ := val.Type()

	// Read container tags from the anonymous `_` field
	containerType := ""
	constrTag := uint(0)
	hasConstr := false

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" {
			containerType = field.Tag.Get("plutusType")
			if constrStr := field.Tag.Get("plutusConstr"); constrStr != "" {
				c, err := strconv.ParseUint(constrStr, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid plutusConstr tag %q: %w", constrStr, err)
				}
				constrTag = uint(c)
				hasConstr = true
			}
			break
		}
	}

	switch containerType {
	case "Map":
		return marshalMap(val, typ, constrTag, hasConstr)
	default:
		// IndefList, DefList, or no tag. The reference Plutus encoder uses
		// indefinite-length lists for the unannotated container form.
		useIndef := containerType != "DefList"
		return marshalList(val, typ, constrTag, hasConstr, useIndef)
	}
}

func marshalField(fieldVal reflect.Value, field reflect.StructField) (data.PlutusData, error) {
	plutusType := field.Tag.Get("plutusType")

	// BigInt handles nil *big.Int directly, so dispatch before pointer dereference
	if plutusType == "BigInt" {
		return marshalBigInt(fieldVal)
	}

	// Dereference pointers
	for fieldVal.Kind() == reflect.Pointer {
		if fieldVal.IsNil() {
			return nil, fmt.Errorf("nil pointer for field %s", field.Name)
		}
		fieldVal = fieldVal.Elem()
	}

	// Check for PlutusMarshaler interface (pointer or value receiver)
	if fieldVal.CanAddr() {
		if m, ok := fieldVal.Addr().Interface().(PlutusMarshaler); ok {
			return m.ToPlutusData()
		}
	}
	if m, ok := fieldVal.Interface().(PlutusMarshaler); ok {
		return m.ToPlutusData()
	}

	switch plutusType {
	case "Int":
		return marshalInt(fieldVal)
	case "Bytes":
		return marshalBytes(fieldVal)
	case "StringBytes":
		return marshalStringBytes(fieldVal)
	case "HexString":
		return marshalHexString(fieldVal)
	case "Bool":
		return marshalBool(fieldVal, false)
	case "IndefBool":
		return marshalBool(fieldVal, true)
	case "IndefList":
		return marshalSliceOrNested(fieldVal, field, true)
	case "DefList":
		return marshalSliceOrNested(fieldVal, field, false)
	case "Map":
		return marshalSliceAsMap(fieldVal, field)
	case "Custom":
		return nil, fmt.Errorf("field %s tagged Custom but doesn't implement PlutusMarshaler", field.Name)
	default:
		// No tag - recursively marshal as nested struct
		if fieldVal.Kind() == reflect.Struct {
			return marshalValue(fieldVal)
		}
		return nil, fmt.Errorf("unsupported field type %s for field %s", fieldVal.Kind(), field.Name)
	}
}
