package blueprint

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"

	"github.com/Salvionied/apollo/v2"
	"github.com/Salvionied/apollo/v2/constants"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

func dataIsNil(pd data.PlutusData) bool {
	if pd == nil {
		return true
	}
	v := reflect.ValueOf(pd)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

func checkDataDepth(depth int, loc string) error {
	if depth > constants.MaxPlutusDataDepth {
		return fmt.Errorf("%s: plutus data exceeds maximum depth %d", loc, constants.MaxPlutusDataDepth)
	}
	return nil
}

func containerCap(n int) int {
	if n < 0 {
		return 0
	}
	if n > constants.MaxContainerPrealloc {
		return constants.MaxContainerPrealloc
	}
	return n
}

func clonePlutusData(pd data.PlutusData, loc string, depth int) (data.PlutusData, error) {
	if err := checkOpaquePlutus(pd, loc, depth); err != nil {
		return nil, err
	}
	return pd.Clone(), nil
}

// checkOpaquePlutus rejects a typed-nil term anywhere in an opaque Plutus
// Data tree. Clone panics on those pointers, including a nil list item, a nil
// constructor field, and a nil map key or value.
func checkOpaquePlutus(pd data.PlutusData, loc string, depth int) error {
	if err := checkDataDepth(depth, loc); err != nil {
		return err
	}
	if dataIsNil(pd) {
		return fmt.Errorf("%s: nil PlutusData", loc)
	}
	switch v := pd.(type) {
	case *data.List:
		for i, item := range v.Items {
			if err := checkOpaquePlutus(item, fmt.Sprintf("%s[%d]", loc, i), depth+1); err != nil {
				return err
			}
		}
	case *data.Constr:
		for i, field := range v.Fields {
			if err := checkOpaquePlutus(field, fmt.Sprintf("%s[%d]", loc, i), depth+1); err != nil {
				return err
			}
		}
	case *data.Map:
		for i, pair := range v.Pairs {
			if err := checkOpaquePlutus(pair[0], fmt.Sprintf("%s[%d]/key", loc, i), depth+1); err != nil {
				return err
			}
			if err := checkOpaquePlutus(pair[1], fmt.Sprintf("%s[%d]/value", loc, i), depth+1); err != nil {
				return err
			}
		}
	case *data.Value:
		if v.Inner != nil {
			if err := checkOpaquePlutus(v.Inner, loc+"/value", depth+1); err != nil {
				return err
			}
		}
	case *data.Integer:
		if v.Inner == nil {
			return fmt.Errorf("%s: nil PlutusData", loc)
		}
	default:
		return nil
	}
	return nil
}

// plutusIdentity is a stable equality key. Integers and byte strings skip
// CBOR encoding; every other term uses its canonical encoding.
func plutusIdentity(pd data.PlutusData) (string, error) {
	switch v := pd.(type) {
	case *data.ByteString:
		if v == nil {
			return "", errors.New("nil byte string")
		}
		return "b:" + string(v.Inner), nil
	case *data.Integer:
		if v == nil || v.Inner == nil {
			return "", errors.New("nil integer")
		}
		return "i:" + v.Inner.String(), nil
	default:
		raw, err := data.Encode(pd)
		if err != nil {
			return "", err
		}
		return "c:" + string(raw), nil
	}
}

// rememberPlutus records pd in seen. fresh is false when pd was already present.
func rememberPlutus(seen map[string]struct{}, pd data.PlutusData) (fresh bool, err error) {
	key, err := plutusIdentity(pd)
	if err != nil {
		return false, err
	}
	if _, exists := seen[key]; exists {
		return false, nil
	}
	seen[key] = struct{}{}
	return true, nil
}

// Encode encodes a Go value into PlutusData according to a schema from bp.
func (bp *Blueprint) Encode(schema *Schema, v any) (data.PlutusData, error) {
	if schema == nil {
		return nil, errors.New("schema is nil")
	}
	return bp.encodeValue(schema, reflect.ValueOf(v), rootName(schema), 0)
}

// Decode decodes PlutusData into dst (a non-nil pointer) according to a schema
// from bp.
func (bp *Blueprint) Decode(schema *Schema, pd data.PlutusData, dst any) error {
	if schema == nil {
		return errors.New("schema is nil")
	}
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return errors.New("decode requires a non-nil pointer")
	}
	return bp.decodeValue(schema, pd, val.Elem(), rootName(schema), 0)
}

// EncodeDatum encodes v against the named validator's datum schema.
func (bp *Blueprint) EncodeDatum(validatorTitle string, v any) (data.PlutusData, error) {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return nil, err
	}
	if val.Datum == nil {
		return nil, fmt.Errorf("validator %q has no datum schema", validatorTitle)
	}
	return bp.Encode(&val.Datum.Schema, v)
}

// DecodeDatum decodes pd into dst against the named validator's datum schema.
func (bp *Blueprint) DecodeDatum(validatorTitle string, pd data.PlutusData, dst any) error {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return err
	}
	if val.Datum == nil {
		return fmt.Errorf("validator %q has no datum schema", validatorTitle)
	}
	return bp.Decode(&val.Datum.Schema, pd, dst)
}

// EncodeRedeemer encodes v against the named validator's redeemer schema for
// purpose. When the validator has a single redeemer, purpose may be empty.
func (bp *Blueprint) EncodeRedeemer(validatorTitle, purpose string, v any) (data.PlutusData, error) {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return nil, err
	}
	s, err := val.RedeemerSchema(purpose)
	if err != nil {
		return nil, err
	}
	return bp.Encode(s, v)
}

// DecodeRedeemer decodes pd into dst against the named validator's redeemer
// schema for purpose.
func (bp *Blueprint) DecodeRedeemer(validatorTitle, purpose string, pd data.PlutusData, dst any) error {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return err
	}
	s, err := val.RedeemerSchema(purpose)
	if err != nil {
		return err
	}
	return bp.Decode(s, pd, dst)
}

// EncodeDatumValue encodes v against the named validator's datum schema and
// wraps the result in a *common.Datum ready for Apollo transaction builders.
func (bp *Blueprint) EncodeDatumValue(validatorTitle string, v any) (*common.Datum, error) {
	pd, err := bp.EncodeDatum(validatorTitle, v)
	if err != nil {
		return nil, err
	}
	datum := &common.Datum{Data: pd}
	if _, err := apollo.DatumWireCbor(datum); err != nil {
		return nil, err
	}
	return datum, nil
}

// DecodeDatumValue decodes d into dst against the named validator's datum
// schema.
func (bp *Blueprint) DecodeDatumValue(validatorTitle string, d *common.Datum, dst any) error {
	if d == nil || d.Data == nil {
		return errors.New("datum is nil")
	}
	return bp.DecodeDatum(validatorTitle, d.Data, dst)
}

// EncodeRedeemerValue encodes v against the named validator's redeemer schema
// for purpose and wraps the result in a *common.Datum.
func (bp *Blueprint) EncodeRedeemerValue(validatorTitle, purpose string, v any) (*common.Datum, error) {
	pd, err := bp.EncodeRedeemer(validatorTitle, purpose, v)
	if err != nil {
		return nil, err
	}
	datum := &common.Datum{Data: pd}
	if _, err := apollo.DatumWireCbor(datum); err != nil {
		return nil, err
	}
	return datum, nil
}

// DecodeRedeemerValue decodes d into dst against the named validator's
// redeemer schema for purpose.
func (bp *Blueprint) DecodeRedeemerValue(validatorTitle, purpose string, d *common.Datum, dst any) error {
	if d == nil || d.Data == nil {
		return errors.New("redeemer datum is nil")
	}
	return bp.DecodeRedeemer(validatorTitle, purpose, d.Data, dst)
}

// DecodeOutputDatum reads the inline datum from output and decodes it into dst
// against the named validator's datum schema.
func (bp *Blueprint) DecodeOutputDatum(validatorTitle string, output common.TransactionOutput, dst any) error {
	if output == nil {
		return errors.New("transaction output is nil")
	}
	d := output.Datum()
	if d == nil || d.Data == nil {
		return fmt.Errorf("output has no inline datum for validator %q", validatorTitle)
	}
	return bp.DecodeDatum(validatorTitle, d.Data, dst)
}

// DatumHash encodes v against the named validator's datum schema and returns
// its 32-byte Blake2b-256 datum hash.
func (bp *Blueprint) DatumHash(validatorTitle string, v any) (common.Blake2b256, error) {
	d, err := bp.EncodeDatumValue(validatorTitle, v)
	if err != nil {
		return common.Blake2b256{}, err
	}
	return apollo.DatumHash(d)
}

// DatumHashHex encodes v against the named validator's datum schema and
// returns its hex-encoded 32-byte Blake2b-256 datum hash.
func (bp *Blueprint) DatumHashHex(validatorTitle string, v any) (string, error) {
	h, err := bp.DatumHash(validatorTitle, v)
	if err != nil {
		return "", err
	}
	return h.String(), nil
}

// EncodeDefinition encodes v against the named schema under definitions.
func (bp *Blueprint) EncodeDefinition(name string, v any) (data.PlutusData, error) {
	s, err := bp.Definition(name)
	if err != nil {
		return nil, err
	}
	return bp.Encode(s, v)
}

// DecodeDefinition decodes pd into dst against the named schema under
// definitions.
func (bp *Blueprint) DecodeDefinition(name string, pd data.PlutusData, dst any) error {
	s, err := bp.Definition(name)
	if err != nil {
		return err
	}
	return bp.Decode(s, pd, dst)
}

// EncodeParameter encodes v against validators[validatorTitle].parameters[index].Schema
// without applying it to compiledCode.
func (bp *Blueprint) EncodeParameter(validatorTitle string, index int, v any) (data.PlutusData, error) {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(val.Parameters) {
		return nil, fmt.Errorf("validator %q parameter index %d out of range [0,%d)", validatorTitle, index, len(val.Parameters))
	}
	return bp.Encode(&val.Parameters[index].Schema, v)
}

// DecodeParameter decodes pd into dst against
// validators[validatorTitle].parameters[index].Schema.
func (bp *Blueprint) DecodeParameter(validatorTitle string, index int, pd data.PlutusData, dst any) error {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(val.Parameters) {
		return fmt.Errorf("validator %q parameter index %d out of range [0,%d)", validatorTitle, index, len(val.Parameters))
	}
	return bp.Decode(&val.Parameters[index].Schema, pd, dst)
}

// ValidateData checks that pd conforms to schema without decoding into a Go
// struct.
func (bp *Blueprint) ValidateData(schema *Schema, pd data.PlutusData) error {
	if schema == nil {
		return errors.New("schema is nil")
	}
	return bp.validatePlutusData(schema, pd, rootName(schema), 0)
}

// ValidateDatum checks that pd conforms to the named validator's datum schema.
func (bp *Blueprint) ValidateDatum(validatorTitle string, pd data.PlutusData) error {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return err
	}
	if val.Datum == nil {
		return fmt.Errorf("validator %q has no datum schema", validatorTitle)
	}
	return bp.ValidateData(&val.Datum.Schema, pd)
}

// ValidateRedeemer checks that pd conforms to the named validator's redeemer
// schema for purpose.
func (bp *Blueprint) ValidateRedeemer(validatorTitle, purpose string, pd data.PlutusData) error {
	val, err := bp.Validator(validatorTitle)
	if err != nil {
		return err
	}
	s, err := val.RedeemerSchema(purpose)
	if err != nil {
		return err
	}
	return bp.ValidateData(s, pd)
}

func (bp *Blueprint) validatePlutusData(s *Schema, pd data.PlutusData, loc string, depth int) error {
	if err := checkDataDepth(depth, loc); err != nil {
		return err
	}
	if dataIsNil(pd) {
		return fmt.Errorf("%s: nil PlutusData", loc)
	}
	resolved, err := bp.deref(s)
	if err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	s = resolved

	if s.DataType == "" && len(s.AnyOf) == 0 {
		return checkOpaquePlutus(pd, loc, depth)
	}
	if len(s.AnyOf) > 0 {
		constr, ok := pd.(*data.Constr)
		if !ok || constr.Tag == nil || !constr.Tag.IsUint64() {
			return fmt.Errorf("%s: expected Constr for anyOf, got %T", loc, pd)
		}
		tag := constr.Tag.Uint64()
		for _, v := range s.AnyOf {
			if *v.Index == tag {
				return bp.validatePlutusConstructor(v, constr, loc+"/"+v.Title, depth)
			}
		}
		return fmt.Errorf("%s: constructor index %d is not in anyOf schema", loc, tag)
	}

	switch s.DataType {
	case "integer":
		integer, ok := pd.(*data.Integer)
		if !ok || integer.Inner == nil {
			return fmt.Errorf("%s: expected Integer, got %T", loc, pd)
		}
		return checkIntegerBounds(s, integer.Inner, loc)
	case "bytes":
		bs, ok := pd.(*data.ByteString)
		if !ok {
			return fmt.Errorf("%s: expected ByteString, got %T", loc, pd)
		}
		return checkBytesBounds(s, bs.Inner, loc)
	case "list":
		list, ok := pd.(*data.List)
		if !ok {
			return fmt.Errorf("%s: expected List, got %T", loc, pd)
		}
		if err := checkCountBounds(s.MinItems, s.MaxItems, len(list.Items), loc); err != nil {
			return err
		}
		if s.Items.Tuple != nil && len(list.Items) != len(s.Items.Tuple) {
			return fmt.Errorf("%s: tuple expects %d elements, got %d", loc, len(s.Items.Tuple), len(list.Items))
		}
		var seen map[string]struct{}
		if s.UniqueItems {
			seen = make(map[string]struct{}, containerCap(len(list.Items)))
		}
		for i, item := range list.Items {
			elemSchema := s.Items.Single
			if s.Items.Tuple != nil {
				elemSchema = s.Items.Tuple[i]
			}
			if err := bp.validatePlutusData(elemSchema, item, fmt.Sprintf("%s[%d]", loc, i), depth+1); err != nil {
				return err
			}
			if s.UniqueItems {
				fresh, err := rememberPlutus(seen, item)
				if err != nil {
					return err
				}
				if !fresh {
					return fmt.Errorf("%s[%d]: duplicate list item violates uniqueItems", loc, i)
				}
			}
		}
		return nil
	case "map":
		m, ok := pd.(*data.Map)
		if !ok {
			return fmt.Errorf("%s: expected Map, got %T", loc, pd)
		}
		if err := checkCountBounds(s.MinItems, s.MaxItems, len(m.Pairs), loc); err != nil {
			return err
		}
		for i, pair := range m.Pairs {
			if err := bp.validatePlutusData(s.Keys, pair[0], fmt.Sprintf("%s[%d]/key", loc, i), depth+1); err != nil {
				return err
			}
			if err := bp.validatePlutusData(s.Values, pair[1], fmt.Sprintf("%s[%d]/value", loc, i), depth+1); err != nil {
				return err
			}
		}
		return nil
	case "constructor":
		constr, ok := pd.(*data.Constr)
		if !ok {
			return fmt.Errorf("%s: expected Constr, got %T", loc, pd)
		}
		return bp.validatePlutusConstructor(s, constr, loc, depth)
	case "value":
		v, ok := pd.(*data.Value)
		if !ok || v == nil {
			return fmt.Errorf("%s: expected *data.Value, got %T", loc, pd)
		}
		return v.Validate()
	default:
		return fmt.Errorf("%s: unsupported dataType %q", loc, s.DataType)
	}
}

func (bp *Blueprint) validatePlutusConstructor(s *Schema, constr *data.Constr, loc string, depth int) error {
	if constr.Tag == nil || !constr.Tag.IsUint64() || constr.Tag.Uint64() != *s.Index {
		return fmt.Errorf("%s: constructor index %v does not match schema index %d", loc, constr.Tag, *s.Index)
	}
	if len(constr.Fields) != len(s.Fields) {
		return fmt.Errorf("%s: constructor has %d fields, schema expects %d", loc, len(constr.Fields), len(s.Fields))
	}
	for i, fieldSchema := range s.Fields {
		if err := bp.validatePlutusData(fieldSchema, constr.Fields[i], fmt.Sprintf("%s.%s", loc, fieldLabel(fieldSchema, i)), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func rootName(s *Schema) string {
	if s.Title != "" {
		return s.Title
	}
	if s.DataType != "" {
		return s.DataType
	}
	return "schema"
}

func (bp *Blueprint) deref(s *Schema) (*Schema, error) {
	if s == nil {
		return nil, errors.New("schema is nil")
	}
	cur := s
	for hops := 0; cur.Ref != ""; hops++ {
		if hops > constants.MaxPlutusDataDepth {
			return nil, fmt.Errorf("$ref cycle involving %q", cur.Ref)
		}
		if cur.resolved == nil {
			_, target, err := bp.resolveRef(cur.Ref)
			if err != nil {
				return nil, err
			}
			cur.resolved = target
		}
		next := cur.resolved
		if next == nil {
			return nil, fmt.Errorf("unresolved $ref %q", cur.Ref)
		}
		cur = next
	}
	return cur, nil
}

func (bp *Blueprint) encodeValue(s *Schema, val reflect.Value, loc string, depth int) (data.PlutusData, error) {
	if err := checkDataDepth(depth, loc); err != nil {
		return nil, err
	}
	resolved, err := bp.deref(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", loc, err)
	}
	s = resolved

	// Opaque Plutus Data schema accepts any data.PlutusData directly.
	if s.DataType == "" && len(s.AnyOf) == 0 {
		return encodeOpaqueData(val, loc, depth)
	}

	if val.IsValid() && val.Type() == reflect.TypeOf((*big.Int)(nil)) {
		if s.DataType != "integer" {
			return nil, fmt.Errorf("%s: *big.Int is only valid for integer schema", loc)
		}
		if val.IsNil() {
			return nil, fmt.Errorf("%s: nil *big.Int", loc)
		}
		return encodeInteger(s, new(big.Int).Set(val.Interface().(*big.Int)), loc)
	}

	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil, fmt.Errorf("%s: nil value", loc)
		}
		val = val.Elem()
	}

	if len(s.AnyOf) > 0 {
		return bp.encodeAnyOf(s, val, loc, depth)
	}

	switch s.DataType {
	case "integer":
		n, err := toBigInt(val, loc)
		if err != nil {
			return nil, err
		}
		return encodeInteger(s, n, loc)
	case "bytes":
		return encodeBytes(s, val, loc)
	case "list":
		return bp.encodeList(s, val, loc, depth)
	case "map":
		return bp.encodeMap(s, val, loc, depth)
	case "constructor":
		return bp.encodeConstructor(s, val, loc, depth)
	case "value":
		return encodeV4Value(val, loc)
	default:
		return nil, fmt.Errorf("%s: unsupported dataType %q", loc, s.DataType)
	}
}

func encodeV4Value(val reflect.Value, loc string) (data.PlutusData, error) {
	if val.Type() == reflect.TypeOf(data.Value{}) {
		v := val.Interface().(data.Value)
		if err := v.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", loc, err)
		}
		return v.Clone(), nil
	}
	if val.Type() == reflect.TypeOf((*data.Value)(nil)) {
		v := val.Interface().(*data.Value)
		if v == nil {
			return nil, fmt.Errorf("%s: nil *data.Value", loc)
		}
		if err := v.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", loc, err)
		}
		return v.Clone(), nil
	}
	return nil, fmt.Errorf("%s: value schema requires *data.Value or data.Value, got %s", loc, val.Type())
}

func decodeV4Value(pd data.PlutusData, dst reflect.Value, loc string) error {
	v, ok := pd.(*data.Value)
	if !ok || v == nil {
		return fmt.Errorf("%s: expected *data.Value, got %T", loc, pd)
	}
	if err := v.Validate(); err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	cloned, ok := v.Clone().(*data.Value)
	if !ok || cloned == nil {
		return fmt.Errorf("%s: expected *data.Value, got %T", loc, pd)
	}
	if dst.Type() == reflect.TypeOf(data.Value{}) {
		dst.Set(reflect.ValueOf(*cloned))
		return nil
	}
	if dst.Type() == reflect.TypeOf((*data.Value)(nil)).Elem() || dst.Type().Implements(reflect.TypeOf((*data.PlutusData)(nil)).Elem()) {
		dst.Set(reflect.ValueOf(cloned))
		return nil
	}
	return fmt.Errorf("%s: cannot decode V4 Value into %s", loc, dst.Type())
}

func encodeOpaqueData(val reflect.Value, loc string, depth int) (data.PlutusData, error) {
	if !val.IsValid() {
		return nil, fmt.Errorf("%s: nil PlutusData", loc)
	}
	for val.Kind() == reflect.Interface || val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, fmt.Errorf("%s: nil PlutusData", loc)
		}
		if pd, ok := val.Interface().(data.PlutusData); ok {
			return clonePlutusData(pd, loc, depth)
		}
		val = val.Elem()
	}
	if pd, ok := val.Interface().(data.PlutusData); ok {
		return clonePlutusData(pd, loc, depth)
	}
	return nil, fmt.Errorf("%s: opaque Data schema requires data.PlutusData, got %s", loc, val.Type())
}

func toBigInt(val reflect.Value, loc string) (*big.Int, error) {
	if val.Type() == reflect.TypeOf(big.Int{}) {
		b := val.Interface().(big.Int)
		return new(big.Int).Set(&b), nil
	}
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return big.NewInt(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return new(big.Int).SetUint64(val.Uint()), nil
	default:
		return nil, fmt.Errorf("%s: expected integer, got %s", loc, val.Type())
	}
}

func encodeInteger(s *Schema, n *big.Int, loc string) (data.PlutusData, error) {
	if err := checkIntegerBounds(s, n, loc); err != nil {
		return nil, err
	}
	return data.NewInteger(n), nil
}

func checkIntegerBounds(s *Schema, n *big.Int, loc string) error {
	if s.Minimum != nil && n.Cmp(s.Minimum) < 0 {
		return fmt.Errorf("%s: integer %s is below minimum %s", loc, n, s.Minimum)
	}
	if s.Maximum != nil && n.Cmp(s.Maximum) > 0 {
		return fmt.Errorf("%s: integer %s exceeds maximum %s", loc, n, s.Maximum)
	}
	return nil
}

func encodeBytes(s *Schema, val reflect.Value, loc string) (data.PlutusData, error) {
	var raw []byte
	switch val.Kind() {
	case reflect.String:
		decoded, err := hex.DecodeString(val.String())
		if err != nil {
			return nil, fmt.Errorf("%s: invalid hex string: %w", loc, err)
		}
		raw = decoded
	case reflect.Slice:
		if val.Type().Elem().Kind() != reflect.Uint8 {
			return nil, fmt.Errorf("%s: expected []byte or hex string, got %s", loc, val.Type())
		}
		raw = append([]byte(nil), val.Bytes()...)
	case reflect.Array:
		if val.Type().Elem().Kind() != reflect.Uint8 {
			return nil, fmt.Errorf("%s: expected byte array, got %s", loc, val.Type())
		}
		raw = make([]byte, val.Len())
		for i := 0; i < val.Len(); i++ {
			raw[i] = byte(val.Index(i).Uint() & 0xFF)
		}
	default:
		return nil, fmt.Errorf("%s: expected []byte or hex string, got %s", loc, val.Type())
	}
	if err := checkBytesBounds(s, raw, loc); err != nil {
		return nil, err
	}
	return data.NewByteString(raw), nil
}

func checkBytesBounds(s *Schema, raw []byte, loc string) error {
	if s.MinLength != nil && len(raw) < *s.MinLength {
		return fmt.Errorf("%s: byte length %d is below minLength %d", loc, len(raw), *s.MinLength)
	}
	if s.MaxLength != nil && len(raw) > *s.MaxLength {
		return fmt.Errorf("%s: byte length %d exceeds maxLength %d", loc, len(raw), *s.MaxLength)
	}
	if len(s.Enum) > 0 {
		gotHex := hex.EncodeToString(raw)
		matched := false
		for _, allowed := range s.Enum {
			if strings.EqualFold(gotHex, allowed) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s: bytes %q not in enum", loc, gotHex)
		}
	}
	return nil
}

func (bp *Blueprint) encodeList(s *Schema, val reflect.Value, loc string, depth int) (data.PlutusData, error) {
	if s.Items.Tuple != nil && val.Kind() == reflect.Struct {
		fields := exportedFields(val)
		if len(fields) != len(s.Items.Tuple) {
			return nil, fmt.Errorf("%s: tuple expects %d elements, struct has %d", loc, len(s.Items.Tuple), len(fields))
		}
		items := make([]data.PlutusData, len(fields))
		for i, f := range fields {
			pd, err := bp.encodeValue(s.Items.Tuple[i], f.val, fmt.Sprintf("%s[%d]", loc, i), depth+1)
			if err != nil {
				return nil, err
			}
			items[i] = pd
		}
		return data.NewList(items...), nil
	}

	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, fmt.Errorf("%s: expected slice or array for list, got %s", loc, val.Type())
	}
	n := val.Len()
	if err := checkCountBounds(s.MinItems, s.MaxItems, n, loc); err != nil {
		return nil, err
	}
	if s.Items.Tuple != nil && n != len(s.Items.Tuple) {
		return nil, fmt.Errorf("%s: tuple expects %d elements, got %d", loc, len(s.Items.Tuple), n)
	}

	items := make([]data.PlutusData, n)
	var seen map[string]struct{}
	if s.UniqueItems {
		seen = make(map[string]struct{}, containerCap(n))
	}
	for i := 0; i < n; i++ {
		elemSchema := s.Items.Single
		if s.Items.Tuple != nil {
			elemSchema = s.Items.Tuple[i]
		}
		pd, err := bp.encodeValue(elemSchema, val.Index(i), fmt.Sprintf("%s[%d]", loc, i), depth+1)
		if err != nil {
			return nil, err
		}
		if s.UniqueItems {
			fresh, err := rememberPlutus(seen, pd)
			if err != nil {
				return nil, err
			}
			if !fresh {
				return nil, fmt.Errorf("%s[%d]: duplicate list item violates uniqueItems", loc, i)
			}
		}
		items[i] = pd
	}
	return data.NewList(items...), nil
}

func checkCountBounds(minItems, maxItems *int, n int, loc string) error {
	if minItems != nil && n < *minItems {
		return fmt.Errorf("%s: length %d is below minItems %d", loc, n, *minItems)
	}
	if maxItems != nil && n > *maxItems {
		return fmt.Errorf("%s: length %d exceeds maxItems %d", loc, n, *maxItems)
	}
	return nil
}

type encodedMapEntry struct {
	keyBytes []byte
	pair     [2]data.PlutusData
}

func (bp *Blueprint) encodeMap(s *Schema, val reflect.Value, loc string, depth int) (data.PlutusData, error) {
	if val.Kind() != reflect.Map {
		return nil, fmt.Errorf("%s: expected Go map, got %s", loc, val.Type())
	}
	n := val.Len()
	if err := checkCountBounds(s.MinItems, s.MaxItems, n, loc); err != nil {
		return nil, err
	}

	entries := make([]encodedMapEntry, 0, n)
	iter := val.MapRange()
	for iter.Next() {
		kPd, err := bp.encodeValue(s.Keys, iter.Key(), loc+"/key", depth+1)
		if err != nil {
			return nil, err
		}
		vPd, err := bp.encodeValue(s.Values, iter.Value(), loc+"/value", depth+1)
		if err != nil {
			return nil, err
		}
		kBytes, err := data.Encode(kPd)
		if err != nil {
			return nil, err
		}
		entries = append(entries, encodedMapEntry{
			keyBytes: kBytes,
			pair:     [2]data.PlutusData{kPd, vPd},
		})
	}

	slices.SortFunc(entries, func(a, b encodedMapEntry) int {
		return bytes.Compare(a.keyBytes, b.keyBytes)
	})
	pairs := make([][2]data.PlutusData, len(entries))
	for i := range entries {
		if i > 0 && bytes.Equal(entries[i-1].keyBytes, entries[i].keyBytes) {
			return nil, fmt.Errorf("%s: duplicate encoded map key", loc)
		}
		pairs[i] = entries[i].pair
	}
	return data.NewMap(pairs), nil
}

type structFieldVal struct {
	meta reflect.StructField
	val  reflect.Value
}

func exportedFields(val reflect.Value) []structFieldVal {
	typ := val.Type()
	out := make([]structFieldVal, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Name == "_" || !f.IsExported() {
			continue
		}
		out = append(out, structFieldVal{meta: f, val: val.Field(i)})
	}
	return out
}

func normalizeName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", ""))
}

func (bp *Blueprint) encodeConstructor(s *Schema, val reflect.Value, loc string, depth int) (data.PlutusData, error) {
	if len(s.Fields) == 0 {
		if val.Kind() == reflect.Struct && len(exportedFields(val)) == 0 {
			return data.NewConstr(*s.Index), nil
		}
		return nil, fmt.Errorf("%s: 0-field constructor requires an empty struct, got %s", loc, val.Type())
	}

	if len(s.Fields) == 1 && shouldUnwrapSingleField(s.Fields[0], val.Type()) {
		pd, err := bp.encodeValue(s.Fields[0], val, fmt.Sprintf("%s.%s", loc, fieldLabel(s.Fields[0], 0)), depth+1)
		if err != nil {
			return nil, err
		}
		return data.NewConstr(*s.Index, pd), nil
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s: expected struct for constructor, got %s", loc, val.Type())
	}
	fields := exportedFields(val)
	if len(fields) != len(s.Fields) {
		return nil, fmt.Errorf("%s: constructor expects %d fields, struct %s has %d", loc, len(s.Fields), val.Type().Name(), len(fields))
	}

	encoded := make([]data.PlutusData, len(s.Fields))
	for i, schemaField := range s.Fields {
		goField := fields[i]
		if schemaField.Title != "" && !fieldMatchesTitle(goField.meta, schemaField.Title) {
			return nil, fmt.Errorf("%s: field %d is %q in Go struct but %q in blueprint schema", loc, i, goField.meta.Name, schemaField.Title)
		}
		pd, err := bp.encodeValue(schemaField, goField.val, fmt.Sprintf("%s.%s", loc, fieldLabel(schemaField, i)), depth+1)
		if err != nil {
			return nil, err
		}
		encoded[i] = pd
	}
	return data.NewConstr(*s.Index, encoded...), nil
}

func fieldLabel(s *Schema, idx int) string {
	if s.Title != "" {
		return s.Title
	}
	return fmt.Sprintf("field[%d]", idx)
}

func fieldMatchesTitle(f reflect.StructField, title string) bool {
	if tag := f.Tag.Get("blueprint"); tag != "" {
		return normalizeName(tag) == normalizeName(title)
	}
	return normalizeName(f.Name) == normalizeName(title)
}

func shouldUnwrapSingleField(fieldSchema *Schema, goType reflect.Type) bool {
	if goType.Kind() != reflect.Struct {
		return true
	}
	if fieldSchema.Title == "" {
		return true
	}
	fields := make([]reflect.StructField, 0, goType.NumField())
	for i := 0; i < goType.NumField(); i++ {
		f := goType.Field(i)
		if f.Name == "_" || !f.IsExported() {
			continue
		}
		fields = append(fields, f)
	}
	if len(fields) == 1 && fieldMatchesTitle(fields[0], fieldSchema.Title) {
		return false
	}
	return true
}

func (bp *Blueprint) encodeAnyOf(s *Schema, val reflect.Value, loc string, depth int) (data.PlutusData, error) {
	if len(s.AnyOf) == 1 {
		return bp.encodeConstructor(s.AnyOf[0], val, loc, depth)
	}
	if isBooleanAnyOf(s.AnyOf) && val.Kind() == reflect.Bool {
		if val.Bool() {
			return data.NewConstr(1), nil
		}
		return data.NewConstr(0), nil
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%s: anyOf requires a struct of variant pointers, got %s", loc, val.Type())
	}

	fields := exportedFields(val)
	var chosen *Schema
	var chosenVal reflect.Value
	for _, f := range fields {
		if f.val.Kind() != reflect.Pointer {
			return nil, fmt.Errorf("%s: anyOf field %s must be a pointer", loc, f.meta.Name)
		}
		if f.val.IsNil() {
			continue
		}
		if chosen != nil {
			return nil, fmt.Errorf("%s: multiple anyOf variants are set", loc)
		}
		variant := findVariant(s.AnyOf, f.meta)
		if variant == nil {
			return nil, fmt.Errorf("%s: struct field %s does not match any variant in anyOf", loc, f.meta.Name)
		}
		chosen = variant
		chosenVal = f.val.Elem()
	}
	if chosen == nil {
		return nil, fmt.Errorf("%s: no anyOf variant is set", loc)
	}
	return bp.encodeConstructor(chosen, chosenVal, loc+"/"+chosen.Title, depth)
}

func isBooleanAnyOf(variants []*Schema) bool {
	if len(variants) != 2 {
		return false
	}
	var hasFalse, hasTrue bool
	for _, v := range variants {
		if len(v.Fields) != 0 || v.Index == nil {
			return false
		}
		if *v.Index == 0 && strings.EqualFold(v.Title, "False") {
			hasFalse = true
		}
		if *v.Index == 1 && strings.EqualFold(v.Title, "True") {
			hasTrue = true
		}
	}
	return hasFalse && hasTrue
}

func findVariant(variants []*Schema, f reflect.StructField) *Schema {
	for _, v := range variants {
		if v.Title != "" && fieldMatchesTitle(f, v.Title) {
			return v
		}
	}
	return nil
}

func (bp *Blueprint) decodeValue(s *Schema, pd data.PlutusData, dst reflect.Value, loc string, depth int) error {
	if err := checkDataDepth(depth, loc); err != nil {
		return err
	}
	if dataIsNil(pd) {
		return fmt.Errorf("%s: nil PlutusData", loc)
	}
	resolved, err := bp.deref(s)
	if err != nil {
		return fmt.Errorf("%s: %w", loc, err)
	}
	s = resolved

	if s.DataType == "" && len(s.AnyOf) == 0 {
		if pd == nil {
			return fmt.Errorf("%s: nil PlutusData", loc)
		}
		if dst.Type() == reflect.TypeOf((*data.PlutusData)(nil)).Elem() || dst.Type().Implements(reflect.TypeOf((*data.PlutusData)(nil)).Elem()) {
			cloned, err := clonePlutusData(pd, loc, depth)
			if err != nil {
				return err
			}
			dst.Set(reflect.ValueOf(cloned))
			return nil
		}
		return fmt.Errorf("%s: opaque Data target must be data.PlutusData, got %s", loc, dst.Type())
	}

	if dst.Type() == reflect.TypeOf((*big.Int)(nil)) {
		n, err := decodeInteger(s, pd, loc)
		if err != nil {
			return err
		}
		dst.Set(reflect.ValueOf(n))
		return nil
	}

	for dst.Kind() == reflect.Pointer {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}

	if len(s.AnyOf) > 0 {
		return bp.decodeAnyOf(s, pd, dst, loc, depth)
	}

	switch s.DataType {
	case "integer":
		n, err := decodeInteger(s, pd, loc)
		if err != nil {
			return err
		}
		return assignBigInt(n, dst, loc)
	case "bytes":
		return decodeBytes(s, pd, dst, loc)
	case "list":
		return bp.decodeList(s, pd, dst, loc, depth)
	case "map":
		return bp.decodeMap(s, pd, dst, loc, depth)
	case "constructor":
		return bp.decodeConstructor(s, pd, dst, loc, depth)
	case "value":
		return decodeV4Value(pd, dst, loc)
	default:
		return fmt.Errorf("%s: unsupported dataType %q", loc, s.DataType)
	}
}

func decodeInteger(s *Schema, pd data.PlutusData, loc string) (*big.Int, error) {
	integer, ok := pd.(*data.Integer)
	if !ok || integer.Inner == nil {
		return nil, fmt.Errorf("%s: expected Integer, got %T", loc, pd)
	}
	if err := checkIntegerBounds(s, integer.Inner, loc); err != nil {
		return nil, err
	}
	return new(big.Int).Set(integer.Inner), nil
}

func assignBigInt(n *big.Int, dst reflect.Value, loc string) error {
	if dst.Type() == reflect.TypeOf(big.Int{}) {
		dst.Set(reflect.ValueOf(*n))
		return nil
	}
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !n.IsInt64() || dst.OverflowInt(n.Int64()) {
			return fmt.Errorf("%s: integer %s overflows %s", loc, n, dst.Type())
		}
		dst.SetInt(n.Int64())
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if n.Sign() < 0 || !n.IsUint64() || dst.OverflowUint(n.Uint64()) {
			return fmt.Errorf("%s: integer %s overflows %s", loc, n, dst.Type())
		}
		dst.SetUint(n.Uint64())
		return nil
	default:
		return fmt.Errorf("%s: cannot decode integer into %s", loc, dst.Type())
	}
}

func decodeBytes(s *Schema, pd data.PlutusData, dst reflect.Value, loc string) error {
	bs, ok := pd.(*data.ByteString)
	if !ok {
		return fmt.Errorf("%s: expected ByteString, got %T", loc, pd)
	}
	if err := checkBytesBounds(s, bs.Inner, loc); err != nil {
		return err
	}
	switch dst.Kind() {
	case reflect.String:
		dst.SetString(hex.EncodeToString(bs.Inner))
		return nil
	case reflect.Slice:
		if dst.Type().Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("%s: cannot decode bytes into %s", loc, dst.Type())
		}
		dst.SetBytes(append([]byte(nil), bs.Inner...))
		return nil
	default:
		return fmt.Errorf("%s: cannot decode bytes into %s", loc, dst.Type())
	}
}

func (bp *Blueprint) decodeList(s *Schema, pd data.PlutusData, dst reflect.Value, loc string, depth int) error {
	list, ok := pd.(*data.List)
	if !ok {
		return fmt.Errorf("%s: expected List, got %T", loc, pd)
	}
	n := len(list.Items)
	if err := checkCountBounds(s.MinItems, s.MaxItems, n, loc); err != nil {
		return err
	}
	if s.UniqueItems {
		seen := make(map[string]struct{}, containerCap(n))
		for i, item := range list.Items {
			fresh, err := rememberPlutus(seen, item)
			if err != nil {
				return err
			}
			if !fresh {
				return fmt.Errorf("%s[%d]: duplicate list item violates uniqueItems", loc, i)
			}
		}
	}

	if s.Items.Tuple != nil {
		if n != len(s.Items.Tuple) {
			return fmt.Errorf("%s: tuple expects %d elements, got %d", loc, len(s.Items.Tuple), n)
		}
		if dst.Kind() == reflect.Struct {
			fields := exportedFields(dst)
			if len(fields) != len(s.Items.Tuple) {
				return fmt.Errorf("%s: tuple expects %d fields, struct has %d", loc, len(s.Items.Tuple), len(fields))
			}
			for i, f := range fields {
				if err := bp.decodeValue(s.Items.Tuple[i], list.Items[i], f.val, fmt.Sprintf("%s[%d]", loc, i), depth+1); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if dst.Kind() != reflect.Slice {
		return fmt.Errorf("%s: expected slice target for list, got %s", loc, dst.Type())
	}
	slice := reflect.MakeSlice(dst.Type(), 0, containerCap(n))
	elemType := dst.Type().Elem()
	for i := 0; i < n; i++ {
		elemSchema := s.Items.Single
		if s.Items.Tuple != nil {
			elemSchema = s.Items.Tuple[i]
		}
		elem := reflect.New(elemType).Elem()
		if err := bp.decodeValue(elemSchema, list.Items[i], elem, fmt.Sprintf("%s[%d]", loc, i), depth+1); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}
	dst.Set(slice)
	return nil
}

func (bp *Blueprint) decodeMap(s *Schema, pd data.PlutusData, dst reflect.Value, loc string, depth int) error {
	m, ok := pd.(*data.Map)
	if !ok {
		return fmt.Errorf("%s: expected Map, got %T", loc, pd)
	}
	if dst.Kind() != reflect.Map {
		return fmt.Errorf("%s: expected Go map target, got %s", loc, dst.Type())
	}
	n := len(m.Pairs)
	if err := checkCountBounds(s.MinItems, s.MaxItems, n, loc); err != nil {
		return err
	}

	out := reflect.MakeMapWithSize(dst.Type(), containerCap(n))
	for i, pair := range m.Pairs {
		kVal := reflect.New(dst.Type().Key()).Elem()
		if err := bp.decodeValue(s.Keys, pair[0], kVal, fmt.Sprintf("%s[%d]/key", loc, i), depth+1); err != nil {
			return err
		}
		vVal := reflect.New(dst.Type().Elem()).Elem()
		if err := bp.decodeValue(s.Values, pair[1], vVal, fmt.Sprintf("%s[%d]/value", loc, i), depth+1); err != nil {
			return err
		}
		out.SetMapIndex(kVal, vVal)
	}
	dst.Set(out)
	return nil
}

func (bp *Blueprint) decodeConstructor(s *Schema, pd data.PlutusData, dst reflect.Value, loc string, depth int) error {
	constr, ok := pd.(*data.Constr)
	if !ok {
		return fmt.Errorf("%s: expected Constr, got %T", loc, pd)
	}
	if constr.Tag == nil || !constr.Tag.IsUint64() || constr.Tag.Uint64() != *s.Index {
		return fmt.Errorf("%s: constructor index %v does not match schema index %d", loc, constr.Tag, *s.Index)
	}
	if len(constr.Fields) != len(s.Fields) {
		return fmt.Errorf("%s: constructor has %d fields, schema expects %d", loc, len(constr.Fields), len(s.Fields))
	}
	if len(s.Fields) == 0 {
		return nil
	}
	if len(s.Fields) == 1 && shouldUnwrapSingleField(s.Fields[0], dst.Type()) {
		return bp.decodeValue(s.Fields[0], constr.Fields[0], dst, fmt.Sprintf("%s.%s", loc, fieldLabel(s.Fields[0], 0)), depth+1)
	}
	if dst.Kind() != reflect.Struct {
		return fmt.Errorf("%s: expected struct target for constructor, got %s", loc, dst.Type())
	}
	fields := exportedFields(dst)
	if len(fields) != len(s.Fields) {
		return fmt.Errorf("%s: struct %s has %d fields, schema expects %d", loc, dst.Type().Name(), len(fields), len(s.Fields))
	}
	for i, schemaField := range s.Fields {
		goField := fields[i]
		if schemaField.Title != "" && !fieldMatchesTitle(goField.meta, schemaField.Title) {
			return fmt.Errorf("%s: field %d is %q in Go struct but %q in blueprint schema", loc, i, goField.meta.Name, schemaField.Title)
		}
		if err := bp.decodeValue(schemaField, constr.Fields[i], goField.val, fmt.Sprintf("%s.%s", loc, fieldLabel(schemaField, i)), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (bp *Blueprint) decodeAnyOf(s *Schema, pd data.PlutusData, dst reflect.Value, loc string, depth int) error {
	if len(s.AnyOf) == 1 {
		return bp.decodeConstructor(s.AnyOf[0], pd, dst, loc, depth)
	}
	constr, ok := pd.(*data.Constr)
	if !ok || constr.Tag == nil || !constr.Tag.IsUint64() {
		return fmt.Errorf("%s: expected Constr for anyOf, got %T", loc, pd)
	}
	tag := constr.Tag.Uint64()
	if isBooleanAnyOf(s.AnyOf) && dst.Kind() == reflect.Bool {
		if len(constr.Fields) != 0 || (tag != 0 && tag != 1) {
			return fmt.Errorf("%s: invalid Bool constructor %d", loc, tag)
		}
		dst.SetBool(tag == 1)
		return nil
	}
	if dst.Kind() != reflect.Struct {
		return fmt.Errorf("%s: anyOf requires a struct target, got %s", loc, dst.Type())
	}

	var matchedVariant *Schema
	for _, v := range s.AnyOf {
		if *v.Index == tag {
			matchedVariant = v
			break
		}
	}
	if matchedVariant == nil {
		return fmt.Errorf("%s: constructor index %d is not in anyOf schema", loc, tag)
	}

	fields := exportedFields(dst)
	var targetField *structFieldVal
	for i := range fields {
		f := &fields[i]
		if f.val.Kind() != reflect.Pointer {
			return fmt.Errorf("%s: anyOf field %s must be a pointer", loc, f.meta.Name)
		}
		f.val.SetZero()
		if matchedVariant.Title != "" && fieldMatchesTitle(f.meta, matchedVariant.Title) {
			targetField = f
		}
	}
	if targetField == nil {
		return fmt.Errorf("%s: no struct field matches variant %q (index %d)", loc, matchedVariant.Title, tag)
	}
	ptr := reflect.New(targetField.val.Type().Elem())
	if err := bp.decodeConstructor(matchedVariant, pd, ptr.Elem(), loc+"/"+matchedVariant.Title, depth); err != nil {
		return err
	}
	targetField.val.Set(ptr)
	return nil
}
