package plutusencoder

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"

	"github.com/blinklabs-io/plutigo/data"
)

// PlutusMarshaler is the interface for custom plutus data encoding/decoding.
type PlutusMarshaler interface {
	ToPlutusData() (data.PlutusData, error)
	FromPlutusData(pd data.PlutusData, res any) error
}

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
		// IndefList, DefList, or no tag (default to DefList)
		useIndef := containerType == "IndefList"
		return marshalList(val, typ, constrTag, hasConstr, useIndef)
	}
}

func marshalList(val reflect.Value, typ reflect.Type, constrTag uint, hasConstr bool, useIndef bool) (data.PlutusData, error) {
	var fields []data.PlutusData

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" || !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		pd, err := marshalField(fieldVal, field)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}
		fields = append(fields, pd)
	}

	if hasConstr {
		return data.NewConstrDefIndef(useIndef, constrTag, fields...), nil
	}
	return data.NewListDefIndef(useIndef, fields...), nil
}

func marshalMap(val reflect.Value, typ reflect.Type, constrTag uint, hasConstr bool) (data.PlutusData, error) {
	var pairs [][2]data.PlutusData

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" || !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)

		keyName := field.Tag.Get("plutusKey")
		if keyName == "" {
			keyName = field.Name
		}

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

func marshalInt(val reflect.Value) (data.PlutusData, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return data.NewInteger(big.NewInt(val.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return data.NewInteger(new(big.Int).SetUint64(val.Uint())), nil
	default:
		return nil, fmt.Errorf("int tag requires integer type, got %s", val.Kind())
	}
}

func marshalBigInt(val reflect.Value) (data.PlutusData, error) {
	switch v := val.Interface().(type) {
	case *big.Int:
		if v == nil {
			return data.NewInteger(big.NewInt(0)), nil
		}
		return data.NewInteger(v), nil
	case big.Int:
		return data.NewInteger(&v), nil
	default:
		return nil, fmt.Errorf("BigInt tag requires *big.Int or big.Int, got %T", val.Interface())
	}
}

func marshalBytes(val reflect.Value) (data.PlutusData, error) {
	if val.Kind() != reflect.Slice || val.Type().Elem().Kind() != reflect.Uint8 {
		return nil, fmt.Errorf("bytes tag requires []byte, got %s", val.Type())
	}
	return data.NewByteString(val.Bytes()), nil
}

func marshalStringBytes(val reflect.Value) (data.PlutusData, error) {
	if val.Kind() != reflect.String {
		return nil, fmt.Errorf("StringBytes tag requires string, got %s", val.Kind())
	}
	return data.NewByteString([]byte(val.String())), nil
}

func marshalHexString(val reflect.Value) (data.PlutusData, error) {
	if val.Kind() != reflect.String {
		return nil, fmt.Errorf("HexString tag requires string, got %s", val.Kind())
	}
	b, err := hex.DecodeString(val.String())
	if err != nil {
		return nil, fmt.Errorf("HexString: invalid hex: %w", err)
	}
	return data.NewByteString(b), nil
}

func marshalBool(val reflect.Value, useIndef bool) (data.PlutusData, error) {
	if val.Kind() != reflect.Bool {
		return nil, fmt.Errorf("bool tag requires bool, got %s", val.Kind())
	}
	tag := uint(0)
	if val.Bool() {
		tag = 1
	}
	return data.NewConstrDefIndef(useIndef, tag), nil
}

func marshalSliceOrNested(val reflect.Value, field reflect.StructField, useIndef bool) (data.PlutusData, error) {
	if val.Kind() == reflect.Slice {
		var items []data.PlutusData
		for i := 0; i < val.Len(); i++ {
			elem := val.Index(i)
			pd, err := marshalSliceElement(elem)
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i, err)
			}
			items = append(items, pd)
		}
		return data.NewListDefIndef(useIndef, items...), nil
	}
	// Nested struct
	return marshalValue(val)
}

// marshalSliceElement marshals a single slice element, handling both struct and primitive types.
func marshalSliceElement(elem reflect.Value) (data.PlutusData, error) {
	for elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return nil, errors.New("nil pointer in slice")
		}
		elem = elem.Elem()
	}
	switch elem.Kind() {
	case reflect.Struct:
		return marshalValue(elem)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return data.NewInteger(big.NewInt(elem.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return data.NewInteger(new(big.Int).SetUint64(elem.Uint())), nil
	case reflect.String:
		return data.NewByteString([]byte(elem.String())), nil
	case reflect.Slice:
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			return data.NewByteString(elem.Bytes()), nil
		}
		return nil, fmt.Errorf("unsupported slice element type: %s", elem.Type())
	default:
		return nil, fmt.Errorf("unsupported slice element kind: %s", elem.Kind())
	}
}

func marshalSliceAsMap(val reflect.Value, field reflect.StructField) (data.PlutusData, error) {
	if val.Kind() == reflect.Slice {
		var pairs [][2]data.PlutusData
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

// UnmarshalPlutus decodes PlutusData into a Go struct using struct tags.
func UnmarshalPlutus(pd data.PlutusData, v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return errors.New("UnmarshalPlutus requires a non-nil pointer")
	}
	return unmarshalValue(pd, val.Elem())
}

func unmarshalValue(pd data.PlutusData, val reflect.Value) error {
	// Check for PlutusMarshaler (pointer or value receiver)
	if val.CanAddr() {
		if m, ok := val.Addr().Interface().(PlutusMarshaler); ok {
			return m.FromPlutusData(pd, val.Addr().Interface())
		}
	}
	if m, ok := val.Interface().(PlutusMarshaler); ok {
		return m.FromPlutusData(pd, val.Interface())
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal target must be a struct, got %s", val.Kind())
	}

	typ := val.Type()

	// Read container type
	containerType := ""
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" {
			containerType = field.Tag.Get("plutusType")
			break
		}
	}

	switch containerType {
	case "Map":
		return unmarshalFromMap(pd, val, typ)
	default:
		return unmarshalFromList(pd, val, typ)
	}
}

func unmarshalFromList(pd data.PlutusData, val reflect.Value, typ reflect.Type) error {
	var fields []data.PlutusData

	// Read expected Constr tag from struct tag
	var expectedConstr uint
	hasExpectedConstr := false
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" {
			if constrStr := field.Tag.Get("plutusConstr"); constrStr != "" {
				c, err := strconv.ParseUint(constrStr, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid plutusConstr tag %q: %w", constrStr, err)
				}
				expectedConstr = uint(c)
				hasExpectedConstr = true
			}
			break
		}
	}

	switch v := pd.(type) {
	case *data.Constr:
		if hasExpectedConstr && v.Tag != expectedConstr {
			return fmt.Errorf("expected Constr tag %d, got %d", expectedConstr, v.Tag)
		}
		fields = v.Fields
	case *data.List:
		if hasExpectedConstr {
			return fmt.Errorf("expected Constr with tag %d, got List", expectedConstr)
		}
		fields = v.Items
	default:
		return fmt.Errorf("expected Constr or List, got %T", pd)
	}

	// Count exported fields (excluding the "_" tag field).
	exportedCount := 0
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Name != "_" && f.IsExported() {
			exportedCount++
		}
	}
	if len(fields) < exportedCount {
		return fmt.Errorf("plutus data has %d fields, struct %s expects %d", len(fields), typ.Name(), exportedCount)
	}
	// Extra fields in the PlutusData (len(fields) > exportedCount) are intentionally
	// ignored for forward-compatibility with newer datum schemas.

	fieldIdx := 0
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" || !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		if err := unmarshalField(fields[fieldIdx], fieldVal, field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
		fieldIdx++
	}
	return nil
}

func unmarshalFromMap(pd data.PlutusData, val reflect.Value, typ reflect.Type) error {
	// Read expected Constr tag from struct tag
	var expectedConstr uint
	hasExpectedConstr := false
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "_" {
			if constrStr := field.Tag.Get("plutusConstr"); constrStr != "" {
				c, err := strconv.ParseUint(constrStr, 10, 32)
				if err != nil {
					return fmt.Errorf("invalid plutusConstr tag %q: %w", constrStr, err)
				}
				expectedConstr = uint(c)
				hasExpectedConstr = true
			}
			break
		}
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

// isOptionalField reports whether a struct field is marked optional via the
// `plutusOptional:"true"` tag. Optional fields may be absent from an input
// map and are left at their zero value. Tag values are parsed with
// strconv.ParseBool; an invalid value is an error rather than being silently
// treated as required.
func isOptionalField(field reflect.StructField) (bool, error) {
	tagVal := field.Tag.Get("plutusOptional")
	if tagVal == "" {
		return false, nil
	}
	optional, err := strconv.ParseBool(tagVal)
	if err != nil {
		return false, fmt.Errorf("invalid plutusOptional tag %q on field %s: %w", tagVal, field.Name, err)
	}
	return optional, nil
}

func unmarshalField(pd data.PlutusData, fieldVal reflect.Value, field reflect.StructField) error {
	plutusType := field.Tag.Get("plutusType")

	// BigInt handles *big.Int directly, so dispatch before pointer dereference
	if plutusType == "BigInt" {
		return unmarshalBigInt(pd, fieldVal)
	}

	// Dereference / allocate pointers
	for fieldVal.Kind() == reflect.Pointer {
		if fieldVal.IsNil() {
			fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
		}
		fieldVal = fieldVal.Elem()
	}

	// Check for PlutusMarshaler (pointer or value receiver)
	if fieldVal.CanAddr() {
		if m, ok := fieldVal.Addr().Interface().(PlutusMarshaler); ok {
			return m.FromPlutusData(pd, fieldVal.Addr().Interface())
		}
	}
	if m, ok := fieldVal.Interface().(PlutusMarshaler); ok {
		return m.FromPlutusData(pd, fieldVal.Interface())
	}

	switch plutusType {
	case "Int":
		return unmarshalInt(pd, fieldVal)
	case "Bytes":
		return unmarshalBytes(pd, fieldVal)
	case "StringBytes":
		return unmarshalStringBytes(pd, fieldVal)
	case "HexString":
		return unmarshalHexString(pd, fieldVal)
	case "Bool", "IndefBool":
		return unmarshalBool(pd, fieldVal)
	case "IndefList", "DefList":
		return unmarshalSliceOrNested(pd, fieldVal, field)
	case "Map":
		return unmarshalSliceAsMap(pd, fieldVal, field)
	case "Custom":
		return fmt.Errorf("field %s tagged Custom but doesn't implement PlutusMarshaler", field.Name)
	default:
		// Nested struct
		if fieldVal.Kind() == reflect.Struct {
			return unmarshalValue(pd, fieldVal)
		}
		return fmt.Errorf("unsupported field type %s for field %s", fieldVal.Kind(), field.Name)
	}
}

func unmarshalInt(pd data.PlutusData, fieldVal reflect.Value) error {
	integer, ok := pd.(*data.Integer)
	if !ok {
		return fmt.Errorf("expected Integer, got %T", pd)
	}
	switch fieldVal.Kind() {
	case reflect.Int:
		if !integer.Inner.IsInt64() {
			return fmt.Errorf("integer value %s does not fit in int64", integer.Inner.String())
		}
		v := integer.Inner.Int64()
		if v < math.MinInt || v > math.MaxInt {
			return fmt.Errorf("integer value %d does not fit in int", v)
		}
		fieldVal.SetInt(v)
	case reflect.Int64:
		if !integer.Inner.IsInt64() {
			return fmt.Errorf("integer value %s does not fit in int64", integer.Inner.String())
		}
		fieldVal.SetInt(integer.Inner.Int64())
	case reflect.Int32:
		if !integer.Inner.IsInt64() {
			return fmt.Errorf("integer value %s overflows int64 (required for int32)", integer.Inner.String())
		}
		v := integer.Inner.Int64()
		if v < math.MinInt32 || v > math.MaxInt32 {
			return fmt.Errorf("integer value %d does not fit in int32", v)
		}
		fieldVal.SetInt(v)
	case reflect.Int16:
		if !integer.Inner.IsInt64() {
			return fmt.Errorf("integer value %s overflows int64 (required for int16)", integer.Inner.String())
		}
		v := integer.Inner.Int64()
		if v < math.MinInt16 || v > math.MaxInt16 {
			return fmt.Errorf("integer value %d does not fit in int16", v)
		}
		fieldVal.SetInt(v)
	case reflect.Int8:
		if !integer.Inner.IsInt64() {
			return fmt.Errorf("integer value %s overflows int64 (required for int8)", integer.Inner.String())
		}
		v := integer.Inner.Int64()
		if v < math.MinInt8 || v > math.MaxInt8 {
			return fmt.Errorf("integer value %d does not fit in int8", v)
		}
		fieldVal.SetInt(v)
	case reflect.Uint:
		if integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
			return fmt.Errorf("integer value %s does not fit in uint64", integer.Inner.String())
		}
		v := integer.Inner.Uint64()
		if v > math.MaxUint {
			return fmt.Errorf("integer value %d does not fit in uint", v)
		}
		fieldVal.SetUint(v)
	case reflect.Uint64:
		if integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
			return fmt.Errorf("integer value %s does not fit in uint64", integer.Inner.String())
		}
		fieldVal.SetUint(integer.Inner.Uint64())
	case reflect.Uint32:
		if integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
			return fmt.Errorf("integer value %s overflows uint64 (required for uint32)", integer.Inner.String())
		}
		v := integer.Inner.Uint64()
		if v > math.MaxUint32 {
			return fmt.Errorf("integer value %d does not fit in uint32", v)
		}
		fieldVal.SetUint(v)
	case reflect.Uint16:
		if integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
			return fmt.Errorf("integer value %s overflows uint64 (required for uint16)", integer.Inner.String())
		}
		v := integer.Inner.Uint64()
		if v > math.MaxUint16 {
			return fmt.Errorf("integer value %d does not fit in uint16", v)
		}
		fieldVal.SetUint(v)
	case reflect.Uint8:
		if integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
			return fmt.Errorf("integer value %s overflows uint64 (required for uint8)", integer.Inner.String())
		}
		v := integer.Inner.Uint64()
		if v > math.MaxUint8 {
			return fmt.Errorf("integer value %d does not fit in uint8", v)
		}
		fieldVal.SetUint(v)
	default:
		return fmt.Errorf("int tag requires integer type, got %s", fieldVal.Kind())
	}
	return nil
}

func unmarshalBigInt(pd data.PlutusData, fieldVal reflect.Value) error {
	integer, ok := pd.(*data.Integer)
	if !ok {
		return fmt.Errorf("expected Integer, got %T", pd)
	}
	switch fieldVal.Type() {
	case reflect.TypeFor[*big.Int]():
		fieldVal.Set(reflect.ValueOf(new(big.Int).Set(integer.Inner)))
	case reflect.TypeFor[big.Int]():
		fieldVal.Set(reflect.ValueOf(*new(big.Int).Set(integer.Inner)))
	default:
		return fmt.Errorf("BigInt tag requires *big.Int or big.Int, got %s", fieldVal.Type())
	}
	return nil
}

func unmarshalBytes(pd data.PlutusData, fieldVal reflect.Value) error {
	bs, ok := pd.(*data.ByteString)
	if !ok {
		return fmt.Errorf("expected ByteString, got %T", pd)
	}
	if fieldVal.Kind() != reflect.Slice || fieldVal.Type().Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("bytes tag requires []byte, got %s", fieldVal.Type())
	}
	fieldVal.SetBytes(append([]byte(nil), bs.Inner...))
	return nil
}

func unmarshalStringBytes(pd data.PlutusData, fieldVal reflect.Value) error {
	bs, ok := pd.(*data.ByteString)
	if !ok {
		return fmt.Errorf("expected ByteString, got %T", pd)
	}
	if fieldVal.Kind() != reflect.String {
		return fmt.Errorf("StringBytes tag requires string, got %s", fieldVal.Kind())
	}
	fieldVal.SetString(string(bs.Inner))
	return nil
}

func unmarshalHexString(pd data.PlutusData, fieldVal reflect.Value) error {
	bs, ok := pd.(*data.ByteString)
	if !ok {
		return fmt.Errorf("expected ByteString, got %T", pd)
	}
	if fieldVal.Kind() != reflect.String {
		return fmt.Errorf("HexString tag requires string, got %s", fieldVal.Kind())
	}
	fieldVal.SetString(hex.EncodeToString(bs.Inner))
	return nil
}

func unmarshalBool(pd data.PlutusData, fieldVal reflect.Value) error {
	constr, ok := pd.(*data.Constr)
	if !ok {
		return fmt.Errorf("expected Constr for Bool, got %T", pd)
	}
	if constr.Tag > 1 {
		return fmt.Errorf("expected Constr tag 0 or 1 for Bool, got %d", constr.Tag)
	}
	if fieldVal.Kind() != reflect.Bool {
		return fmt.Errorf("bool tag requires bool, got %s", fieldVal.Kind())
	}
	fieldVal.SetBool(constr.Tag == 1)
	return nil
}

func unmarshalSliceOrNested(pd data.PlutusData, fieldVal reflect.Value, field reflect.StructField) error {
	if fieldVal.Kind() == reflect.Slice {
		var items []data.PlutusData
		switch v := pd.(type) {
		case *data.List:
			items = v.Items
		case *data.Constr:
			items = v.Fields
		default:
			return fmt.Errorf("expected List or Constr for slice, got %T", pd)
		}

		elemType := fieldVal.Type().Elem()
		result := reflect.MakeSlice(fieldVal.Type(), len(items), len(items))
		for i, item := range items {
			// Handle pointer element types (e.g. []*MyStruct)
			if elemType.Kind() == reflect.Pointer {
				ptr := reflect.New(elemType.Elem())
				if err := unmarshalSliceElement(item, ptr.Elem()); err != nil {
					return fmt.Errorf("element %d: %w", i, err)
				}
				result.Index(i).Set(ptr)
			} else {
				elem := reflect.New(elemType).Elem()
				if err := unmarshalSliceElement(item, elem); err != nil {
					return fmt.Errorf("element %d: %w", i, err)
				}
				result.Index(i).Set(elem)
			}
		}
		fieldVal.Set(result)
		return nil
	}
	// Nested struct
	return unmarshalValue(pd, fieldVal)
}

// unmarshalSliceElement unmarshals a single slice element, handling both struct and primitive types.
func unmarshalSliceElement(pd data.PlutusData, elem reflect.Value) error {
	switch elem.Kind() {
	case reflect.Struct:
		return unmarshalValue(pd, elem)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer, ok := pd.(*data.Integer)
		if !ok {
			return fmt.Errorf("expected Integer, got %T", pd)
		}
		if !integer.Inner.IsInt64() {
			return fmt.Errorf("integer value %s does not fit in %s", integer.Inner.String(), elem.Kind())
		}
		v := integer.Inner.Int64()
		switch elem.Kind() {
		case reflect.Int:
			if v < math.MinInt || v > math.MaxInt {
				return fmt.Errorf("integer value %d does not fit in int", v)
			}
		case reflect.Int8:
			if v < math.MinInt8 || v > math.MaxInt8 {
				return fmt.Errorf("integer value %d does not fit in int8", v)
			}
		case reflect.Int16:
			if v < math.MinInt16 || v > math.MaxInt16 {
				return fmt.Errorf("integer value %d does not fit in int16", v)
			}
		case reflect.Int32:
			if v < math.MinInt32 || v > math.MaxInt32 {
				return fmt.Errorf("integer value %d does not fit in int32", v)
			}
		}
		elem.SetInt(v)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		integer, ok := pd.(*data.Integer)
		if !ok {
			return fmt.Errorf("expected Integer, got %T", pd)
		}
		if integer.Inner.Sign() < 0 || !integer.Inner.IsUint64() {
			return fmt.Errorf("integer value %s does not fit in %s", integer.Inner.String(), elem.Kind())
		}
		v := integer.Inner.Uint64()
		switch elem.Kind() {
		case reflect.Uint:
			if v > math.MaxUint {
				return fmt.Errorf("integer value %d does not fit in uint", v)
			}
		case reflect.Uint8:
			if v > math.MaxUint8 {
				return fmt.Errorf("integer value %d does not fit in uint8", v)
			}
		case reflect.Uint16:
			if v > math.MaxUint16 {
				return fmt.Errorf("integer value %d does not fit in uint16", v)
			}
		case reflect.Uint32:
			if v > math.MaxUint32 {
				return fmt.Errorf("integer value %d does not fit in uint32", v)
			}
		}
		elem.SetUint(v)
		return nil
	case reflect.String:
		bs, ok := pd.(*data.ByteString)
		if !ok {
			return fmt.Errorf("expected ByteString, got %T", pd)
		}
		elem.SetString(string(bs.Inner))
		return nil
	case reflect.Slice:
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			bs, ok := pd.(*data.ByteString)
			if !ok {
				return fmt.Errorf("expected ByteString, got %T", pd)
			}
			elem.SetBytes(append([]byte(nil), bs.Inner...))
			return nil
		}
		return fmt.Errorf("unsupported nested slice type: %s", elem.Type())
	default:
		return fmt.Errorf("unsupported slice element kind: %s", elem.Kind())
	}
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
