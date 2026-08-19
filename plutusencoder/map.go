package plutusencoder

import (
	"fmt"
	"reflect"

	"github.com/blinklabs-io/plutigo/data"
)

func marshalMap(val reflect.Value, typ reflect.Type, constrTag uint, hasConstr bool) (data.PlutusData, error) {
	var pairs [][2]data.PlutusData
	seenKeys := make(map[string]struct{}, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" || !field.IsExported() {
			continue
		}

		ignored, err := isIgnoredField(field)
		if err != nil {
			return nil, err
		}
		if ignored {
			continue
		}

		fieldVal := val.Field(i)
		omitEmpty, err := fieldOmitEmpty(field)
		if err != nil {
			return nil, err
		}
		if omitEmpty && isEmptyField(fieldVal) {
			continue
		}

		keyName := field.Tag.Get("plutusKey")
		if keyName == "" {
			keyName = field.Name
		}
		if _, duplicate := seenKeys[keyName]; duplicate {
			return nil, fmt.Errorf("field %s: duplicate map key %q", field.Name, keyName)
		}
		seenKeys[keyName] = struct{}{}

		key := data.NewByteString([]byte(keyName))
		value, err := marshalField(fieldVal, field)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		pairs = append(pairs, [2]data.PlutusData{key, value})
	}

	if hasConstr {
		mapData := data.NewMap(pairs)
		return data.NewConstr(constrTag, mapData), nil
	}
	return data.NewMap(pairs), nil
}

func marshalSliceAsMap(val reflect.Value, field reflect.StructField) (data.PlutusData, error) {
	if val.Kind() == reflect.Slice {
		var pairs [][2]data.PlutusData
		seenKeys := make(map[string]struct{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			for elem.Kind() == reflect.Pointer {
				if elem.IsNil() {
					return nil, fmt.Errorf("nil pointer at element %d", i)
				}
				elem = elem.Elem()
			}
			// Extract key from first exported field of each element
			key, keyIdx, err := extractMapKey(elem)
			if err != nil {
				return nil, fmt.Errorf("element %d key: %w", i, err)
			}
			keyString := key.String()
			if _, duplicate := seenKeys[keyString]; duplicate {
				return nil, fmt.Errorf("element %d: duplicate map key %s", i, keyString)
			}
			seenKeys[keyString] = struct{}{}
			// Marshal only non-key fields as the value to avoid duplicating
			// the key in both the map key and the value.
			pd, err := marshalMapValueFields(elem, keyIdx)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			pairs = append(pairs, [2]data.PlutusData{key, pd})
		}
		return data.NewMap(pairs), nil
	}
	return marshalValue(val)
}

// marshalMapValueFields marshals all exported fields of elem except the key field at keyIdx.
// If exactly one non-key field remains, it is returned directly; otherwise a list is returned.
func marshalMapValueFields(elem reflect.Value, keyIdx int) (data.PlutusData, error) {
	typ := elem.Type()
	var fields []data.PlutusData
	for i := 0; i < typ.NumField(); i++ {
		if i == keyIdx {
			continue
		}
		f := typ.Field(i)
		if f.Name == "_" || !f.IsExported() {
			continue
		}
		pd, err := marshalField(elem.Field(i), f)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}
		fields = append(fields, pd)
	}
	if len(fields) == 1 {
		return fields[0], nil
	}
	return data.NewList(fields...), nil
}

// extractMapKey gets the map key from a slice element.
// For structs, uses the first exported field as the key (string → ByteString, else marshalled).
// Returns the key, the field index used, and any error.
func extractMapKey(elem reflect.Value) (data.PlutusData, int, error) {
	if elem.Kind() == reflect.Struct {
		typ := elem.Type()
		for j := 0; j < typ.NumField(); j++ {
			f := typ.Field(j)
			if f.Name == "_" || !f.IsExported() {
				continue
			}
			fv := elem.Field(j)
			if fv.Kind() == reflect.String {
				return data.NewByteString([]byte(fv.String())), j, nil
			}
			// For non-string first fields, marshal it
			pd, err := marshalField(fv, f)
			if err != nil {
				return nil, -1, err
			}
			return pd, j, nil
		}
	}
	return nil, -1, fmt.Errorf("cannot extract map key from non-struct element of kind %s", elem.Kind())
}

func unmarshalFromMap(pd data.PlutusData, val reflect.Value, typ reflect.Type) error {
	container, expectedConstr, hasExpectedConstr, err := readContainerMetadata(typ)
	if err != nil {
		return err
	}
	if container != containerMap {
		return fmt.Errorf("unmarshalFromMap received non-map container: %d", container)
	}

	mapData, ok := pd.(*data.Map)
	if !ok {
		// Could be a Constr wrapping a Map
		if constr, ok := pd.(*data.Constr); ok && len(constr.Fields) == 1 {
			if hasExpectedConstr && constr.Tag != expectedConstr {
				return fmt.Errorf("expected Constr tag %d, got %d", expectedConstr, constr.Tag)
			}
			mapData, ok = constr.Fields[0].(*data.Map)
			if !ok {
				return fmt.Errorf("expected Map in Constr, got %T", constr.Fields[0])
			}
		} else if constr, ok := pd.(*data.Constr); ok {
			return fmt.Errorf("expected Constr with 1 field wrapping a Map, got Constr with %d fields", len(constr.Fields))
		} else {
			return fmt.Errorf("expected Map, got %T", pd)
		}
	} else if hasExpectedConstr {
		return fmt.Errorf("expected Constr with tag %d wrapping Map, got bare Map", expectedConstr)
	}

	// Build a lookup from key name to PlutusData. Keys must be ByteStrings
	// (the only key type produced by marshalMap) and must be unique. Anything
	// else is rejected so that untrusted datums cannot shadow or hide keys.
	keyMap := make(map[string]data.PlutusData, len(mapData.Pairs))
	for i, pair := range mapData.Pairs {
		bs, ok := pair[0].(*data.ByteString)
		if !ok {
			return fmt.Errorf("map pair %d: expected ByteString key, got %T", i, pair[0])
		}
		mapKey := string(bs.Inner)
		if _, dup := keyMap[mapKey]; dup {
			return fmt.Errorf("map pair %d: duplicate key %q", i, mapKey)
		}
		keyMap[mapKey] = pair[1]
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" || !field.IsExported() {
			continue
		}
		ignored, err := isIgnoredField(field)
		if err != nil {
			return err
		}
		if ignored {
			continue
		}

		keyName := field.Tag.Get("plutusKey")
		if keyName == "" {
			keyName = field.Name
		}

		value, exists := keyMap[keyName]
		if !exists {
			optional, err := isOptionalField(field)
			if err != nil {
				return err
			}
			if !optional {
				optional, err = fieldOmitEmpty(field)
				if err != nil {
					return err
				}
			}
			if optional {
				// Optional field absent from the map: leave the zero value.
				continue
			}
			return fmt.Errorf("missing required map key %q for field %s of struct %s", keyName, field.Name, typ.Name())
		}

		fieldVal := val.Field(i)
		if err := unmarshalField(value, fieldVal, field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

func unmarshalSliceAsMap(pd data.PlutusData, fieldVal reflect.Value, field reflect.StructField) error {
	if fieldVal.Kind() == reflect.Slice {
		mapData, ok := pd.(*data.Map)
		if !ok {
			return fmt.Errorf("expected Map for slice, got %T", pd)
		}

		elemType := fieldVal.Type().Elem()
		result := reflect.MakeSlice(fieldVal.Type(), len(mapData.Pairs), len(mapData.Pairs))
		for i, pair := range mapData.Pairs {
			var elem reflect.Value
			if elemType.Kind() == reflect.Pointer {
				elem = reflect.New(elemType.Elem()).Elem()
			} else {
				elem = reflect.New(elemType).Elem()
			}
			if err := unmarshalMapEntry(pair, elem); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
			if elemType.Kind() == reflect.Pointer {
				result.Index(i).Set(elem.Addr())
			} else {
				result.Index(i).Set(elem)
			}
		}
		fieldVal.Set(result)
		return nil
	}
	return unmarshalValue(pd, fieldVal)
}

// unmarshalMapEntry restores a map entry into a struct by setting the key field
// from pair[0] and the remaining value fields from pair[1].
func unmarshalMapEntry(pair [2]data.PlutusData, elem reflect.Value) error {
	if elem.Kind() != reflect.Struct {
		return unmarshalValue(pair[1], elem)
	}
	typ := elem.Type()

	// Find the first exported field (the key field)
	keyIdx := -1
	for j := 0; j < typ.NumField(); j++ {
		f := typ.Field(j)
		if f.Name == "_" || !f.IsExported() {
			continue
		}
		keyIdx = j
		break
	}
	if keyIdx < 0 {
		return unmarshalValue(pair[1], elem)
	}

	// Unmarshal the key into the key field
	keyField := typ.Field(keyIdx)
	if err := unmarshalField(pair[0], elem.Field(keyIdx), keyField); err != nil {
		return fmt.Errorf("key field %s: %w", keyField.Name, err)
	}

	// Collect non-key exported fields
	var valueFieldIdxs []int
	for j := 0; j < typ.NumField(); j++ {
		if j == keyIdx {
			continue
		}
		f := typ.Field(j)
		if f.Name == "_" || !f.IsExported() {
			continue
		}
		valueFieldIdxs = append(valueFieldIdxs, j)
	}

	if len(valueFieldIdxs) == 1 {
		// Single value field - unmarshal pair[1] directly into it
		f := typ.Field(valueFieldIdxs[0])
		return unmarshalField(pair[1], elem.Field(valueFieldIdxs[0]), f)
	}

	// Multiple value fields - expect pair[1] to be a List or Constr
	var items []data.PlutusData
	switch v := pair[1].(type) {
	case *data.List:
		items = v.Items
	case *data.Constr:
		items = v.Fields
	default:
		return fmt.Errorf("expected List for multi-field map value, got %T", pair[1])
	}
	if len(items) < len(valueFieldIdxs) {
		return fmt.Errorf("map value has %d items but struct expects %d non-key fields", len(items), len(valueFieldIdxs))
	}
	for i, fieldIdx := range valueFieldIdxs {
		f := typ.Field(fieldIdx)
		if err := unmarshalField(items[i], elem.Field(fieldIdx), f); err != nil {
			return fmt.Errorf("value field %s: %w", f.Name, err)
		}
	}
	return nil
}
