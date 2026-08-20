package plutusencoder

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

type customQuantity int64

func (q customQuantity) ToPlutusData() (data.PlutusData, error) {
	return data.NewInteger(big.NewInt(int64(q) + 100)), nil
}

func (q customQuantity) FromPlutusData(pd data.PlutusData, res any) error {
	integer, ok := pd.(*data.Integer)
	if !ok {
		return fmt.Errorf("expected Integer, got %T", pd)
	}
	target, ok := res.(*customQuantity)
	if !ok {
		return fmt.Errorf("expected *customQuantity result, got %T", res)
	}
	*target = customQuantity(integer.Inner.Int64() - 100)
	return nil
}

func (q customQuantity) IsZero() bool {
	return q == 0
}

type customSliceDatum struct {
	_          struct{}         `plutusType:"DefList" plutusConstr:"0"`
	Quantities []customQuantity `plutusType:"DefList"`
}

func TestRoundTripCustomMarshalerSliceElements(t *testing.T) {
	original := customSliceDatum{Quantities: []customQuantity{1, 2}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	constr, ok := pd.(*data.Constr)
	if !ok {
		t.Fatalf("expected Constr, got %T", pd)
	}
	list, ok := constr.Fields[0].(*data.List)
	if !ok {
		t.Fatalf("expected quantity list, got %T", constr.Fields[0])
	}
	first, ok := list.Items[0].(*data.Integer)
	if !ok {
		t.Fatalf("expected custom integer, got %T", list.Items[0])
	}
	if first.Inner.Int64() != 101 {
		t.Fatalf("expected custom marshaler to add offset 100, got %d", first.Inner.Int64())
	}

	var decoded customSliceDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Quantities) != 2 || decoded.Quantities[0] != 1 || decoded.Quantities[1] != 2 {
		t.Fatalf("unexpected decoded quantities: %+v", decoded.Quantities)
	}
}

type customPointerDatum struct {
	_     struct{}      `plutusType:"DefList" plutusConstr:"0"`
	Token customTokenID `plutusType:"Custom"`
	Count *big.Int      `plutusType:"BigInt"`
}

type customTokenID string

func (id customTokenID) ToPlutusData() (data.PlutusData, error) {
	return data.NewByteString([]byte(id)), nil
}

func (id customTokenID) FromPlutusData(pd data.PlutusData, res any) error {
	bs, ok := pd.(*data.ByteString)
	if !ok {
		return fmt.Errorf("expected ByteString, got %T", pd)
	}
	target, ok := res.(*customTokenID)
	if !ok {
		return fmt.Errorf("expected *customTokenID result, got %T", res)
	}
	*target = customTokenID(bs.Inner)
	return nil
}

func TestCustomMarshalerFieldStillWorks(t *testing.T) {
	original := customPointerDatum{Token: "alpha", Count: big.NewInt(42)}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded customPointerDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Token != "alpha" || decoded.Count.Cmp(big.NewInt(42)) != 0 {
		t.Fatalf("unexpected decoded datum: %+v", decoded)
	}
}
