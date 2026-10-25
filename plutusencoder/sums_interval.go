package plutusencoder

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

// OutputDatum is the Plutus OutputDatum sum type. Neither Inline nor Hash set
// encodes constructor 0 with no fields. Inline set encodes constructor 1 with
// that PlutusData. Hash set encodes constructor 2 with the 32-byte datum hash.
type OutputDatum struct {
	Inline data.PlutusData
	Hash   []byte
}

// BoundKind selects an interval bound variant.
type BoundKind int

const (
	// BoundNegInf is constructor 0, negative infinity.
	BoundNegInf BoundKind = iota
	// BoundFinite is constructor 1, a finite integer bound.
	BoundFinite
	// BoundPosInf is constructor 2, positive infinity.
	BoundPosInf
)

// IntervalBound is one endpoint of a Plutus Interval.
type IntervalBound struct {
	Kind   BoundKind
	Value  int64
	Closed bool
}

// Interval is a Plutus Interval of integers, such as a POSIXTimeRange.
// Each endpoint is encoded with the closure flag the caller sets.
type Interval struct {
	Lower IntervalBound
	Upper IntervalBound
}

func marshalEither(fieldVal reflect.Value) (data.PlutusData, error) {
	if fieldVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("either requires a two-field struct, got %s", fieldVal.Kind())
	}
	leftIdx, rightIdx, err := eitherFieldIndexes(fieldVal.Type())
	if err != nil {
		return nil, err
	}
	left := fieldVal.Field(leftIdx)
	right := fieldVal.Field(rightIdx)
	leftSet := !left.IsNil()
	rightSet := !right.IsNil()
	if leftSet == rightSet {
		return nil, errors.New("either requires one of Left or Right to be non-nil")
	}
	if leftSet {
		inner, err := marshalSumElement(left.Elem(), fieldVal.Type().Field(leftIdx))
		if err != nil {
			return nil, err
		}
		return data.NewConstr(0, inner), nil
	}
	inner, err := marshalSumElement(right.Elem(), fieldVal.Type().Field(rightIdx))
	if err != nil {
		return nil, err
	}
	return data.NewConstr(1, inner), nil
}

func unmarshalEither(pd data.PlutusData, fieldVal reflect.Value) error {
	if fieldVal.Kind() != reflect.Struct {
		return fmt.Errorf("either requires a two-field struct, got %s", fieldVal.Kind())
	}
	leftIdx, rightIdx, err := eitherFieldIndexes(fieldVal.Type())
	if err != nil {
		return err
	}
	constr, ok := pd.(*data.Constr)
	if !ok || len(constr.Fields) != 1 {
		return fmt.Errorf("expected Either constructor with 1 field, got %T", pd)
	}
	leftField := fieldVal.Field(leftIdx)
	rightField := fieldVal.Field(rightIdx)
	leftField.SetZero()
	rightField.SetZero()

	switch {
	case constrTagEqual(constr.Tag, 0):
		ptr := reflect.New(leftField.Type().Elem())
		if err := unmarshalSumElement(constr.Fields[0], ptr.Elem(), fieldVal.Type().Field(leftIdx)); err != nil {
			return err
		}
		leftField.Set(ptr)
		return nil
	case constrTagEqual(constr.Tag, 1):
		ptr := reflect.New(rightField.Type().Elem())
		if err := unmarshalSumElement(constr.Fields[0], ptr.Elem(), fieldVal.Type().Field(rightIdx)); err != nil {
			return err
		}
		rightField.Set(ptr)
		return nil
	default:
		return fmt.Errorf("expected Either constructor 0 or 1, got %s", constr.Tag)
	}
}

func eitherFieldIndexes(typ reflect.Type) (int, int, error) {
	exported := make([]int, 0, 2)
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Name == "_" || !f.IsExported() {
			continue
		}
		if f.Type.Kind() != reflect.Pointer {
			return 0, 0, fmt.Errorf("either field %s must be a pointer", f.Name)
		}
		exported = append(exported, i)
	}
	if len(exported) != 2 {
		return 0, 0, fmt.Errorf("either struct %s must have 2 exported pointer fields, got %d", typ.Name(), len(exported))
	}
	return exported[0], exported[1], nil
}

func marshalSumElement(elem reflect.Value, field reflect.StructField) (data.PlutusData, error) {
	if field.Tag.Get("plutusType") != "" {
		return marshalField(elem, field)
	}
	if elem.Kind() == reflect.Struct {
		return marshalValue(elem)
	}
	switch elem.Kind() {
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
	default:
		return nil, fmt.Errorf("unsupported Either element type %s", elem.Type())
	}
	return nil, fmt.Errorf("unsupported Either element type %s", elem.Type())
}

func unmarshalSumElement(pd data.PlutusData, elem reflect.Value, field reflect.StructField) error {
	if field.Tag.Get("plutusType") != "" {
		return unmarshalField(pd, elem, field)
	}
	if elem.Kind() == reflect.Struct {
		return unmarshalValue(pd, elem)
	}
	switch elem.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return unmarshalInt(pd, elem)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return unmarshalInt(pd, elem)
	case reflect.String:
		return unmarshalStringBytes(pd, elem)
	case reflect.Slice:
		if elem.Type().Elem().Kind() == reflect.Uint8 {
			return unmarshalBytes(pd, elem)
		}
	default:
		return fmt.Errorf("unsupported Either element type %s", elem.Type())
	}
	return fmt.Errorf("unsupported Either element type %s", elem.Type())
}

func marshalOutputDatum(od OutputDatum) (data.PlutusData, error) {
	hasInline := od.Inline != nil
	hasHash := len(od.Hash) > 0
	if hasInline && hasHash {
		return nil, errors.New("output datum cannot set both Inline and Hash")
	}
	if hasInline {
		return data.NewConstr(1, od.Inline), nil
	}
	if hasHash {
		if len(od.Hash) != common.Blake2b256Size {
			return nil, fmt.Errorf("expected %d-byte datum hash, got %d", common.Blake2b256Size, len(od.Hash))
		}
		return data.NewConstr(2, data.NewByteString(append([]byte(nil), od.Hash...))), nil
	}
	return data.NewConstr(0), nil
}

func unmarshalOutputDatum(pd data.PlutusData) (OutputDatum, error) {
	constr, ok := pd.(*data.Constr)
	if !ok {
		return OutputDatum{}, fmt.Errorf("expected OutputDatum constructor, got %T", pd)
	}
	switch {
	case constrTagEqual(constr.Tag, 0):
		if len(constr.Fields) != 0 {
			return OutputDatum{}, errors.New("no-datum OutputDatum must have no fields")
		}
		return OutputDatum{}, nil
	case constrTagEqual(constr.Tag, 1):
		if len(constr.Fields) != 1 || constr.Fields[0] == nil {
			return OutputDatum{}, errors.New("inline OutputDatum requires 1 field")
		}
		return OutputDatum{Inline: constr.Fields[0]}, nil
	case constrTagEqual(constr.Tag, 2):
		if len(constr.Fields) != 1 {
			return OutputDatum{}, errors.New("datum-hash OutputDatum requires 1 field")
		}
		raw, ok := constr.Fields[0].(*data.ByteString)
		if !ok || len(raw.Inner) != common.Blake2b256Size {
			return OutputDatum{}, fmt.Errorf("expected %d-byte datum hash", common.Blake2b256Size)
		}
		return OutputDatum{Hash: append([]byte(nil), raw.Inner...)}, nil
	default:
		return OutputDatum{}, fmt.Errorf("expected OutputDatum constructor 0, 1, or 2, got %s", constr.Tag)
	}
}

func marshalInterval(iv Interval) (data.PlutusData, error) {
	lower, err := marshalIntervalBound(iv.Lower)
	if err != nil {
		return nil, fmt.Errorf("lower bound: %w", err)
	}
	upper, err := marshalIntervalBound(iv.Upper)
	if err != nil {
		return nil, fmt.Errorf("upper bound: %w", err)
	}
	return data.NewConstr(0, lower, upper), nil
}

func marshalIntervalBound(b IntervalBound) (data.PlutusData, error) {
	var ext data.PlutusData
	switch b.Kind {
	case BoundNegInf:
		ext = data.NewConstr(0)
	case BoundFinite:
		ext = data.NewConstr(1, data.NewInteger(big.NewInt(b.Value)))
	case BoundPosInf:
		ext = data.NewConstr(2)
	default:
		return nil, fmt.Errorf("unknown interval bound kind %d", b.Kind)
	}
	closedTag := uint64(0)
	if b.Closed {
		closedTag = 1
	}
	return data.NewConstr(0, ext, data.NewConstr(closedTag)), nil
}

func unmarshalInterval(pd data.PlutusData) (Interval, error) {
	constr, ok := pd.(*data.Constr)
	if !ok || !constrTagEqual(constr.Tag, 0) || len(constr.Fields) != 2 {
		return Interval{}, fmt.Errorf("expected Interval constructor 0 with 2 fields, got %T", pd)
	}
	lower, err := unmarshalIntervalBound(constr.Fields[0])
	if err != nil {
		return Interval{}, fmt.Errorf("lower bound: %w", err)
	}
	upper, err := unmarshalIntervalBound(constr.Fields[1])
	if err != nil {
		return Interval{}, fmt.Errorf("upper bound: %w", err)
	}
	return Interval{Lower: lower, Upper: upper}, nil
}

func unmarshalIntervalBound(pd data.PlutusData) (IntervalBound, error) {
	bound, ok := pd.(*data.Constr)
	if !ok || !constrTagEqual(bound.Tag, 0) || len(bound.Fields) != 2 {
		return IntervalBound{}, fmt.Errorf("expected interval bound constructor 0 with 2 fields, got %T", pd)
	}
	ext, ok := bound.Fields[0].(*data.Constr)
	if !ok {
		return IntervalBound{}, fmt.Errorf("expected Extended constructor, got %T", bound.Fields[0])
	}

	var result IntervalBound
	switch {
	case constrTagEqual(ext.Tag, 0) && len(ext.Fields) == 0:
		result.Kind = BoundNegInf
	case constrTagEqual(ext.Tag, 1) && len(ext.Fields) == 1:
		integer, ok := ext.Fields[0].(*data.Integer)
		if !ok || integer.Inner == nil || !integer.Inner.IsInt64() {
			return IntervalBound{}, errors.New("expected int64 finite bound")
		}
		result.Kind = BoundFinite
		result.Value = integer.Inner.Int64()
	case constrTagEqual(ext.Tag, 2) && len(ext.Fields) == 0:
		result.Kind = BoundPosInf
	default:
		return IntervalBound{}, errors.New("expected Extended NegInf (0), Finite (1), or PosInf (2)")
	}

	closure, ok := bound.Fields[1].(*data.Constr)
	if !ok || len(closure.Fields) != 0 {
		return IntervalBound{}, errors.New("expected closure Bool constructor with no fields")
	}
	switch {
	case constrTagEqual(closure.Tag, 0):
		result.Closed = false
	case constrTagEqual(closure.Tag, 1):
		result.Closed = true
	default:
		return IntervalBound{}, fmt.Errorf("expected closure Bool constructor 0 or 1, got %s", closure.Tag)
	}
	return result, nil
}
