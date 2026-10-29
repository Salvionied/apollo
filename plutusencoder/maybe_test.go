package plutusencoder

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

type maybeDatum struct {
	_       struct{}        `plutusType:"DefList" plutusConstr:"0"`
	Amount  *int64          `plutusType:"Maybe"`
	Address *common.Address `plutusType:"Maybe"`
	Nested  *maybeInner     `plutusType:"Maybe"`
}

type maybeInner struct {
	_    struct{} `plutusType:"DefList" plutusConstr:"2"`
	Name string   `plutusType:"StringBytes"`
}

func TestMaybeNothingAndJust(t *testing.T) {
	payment := bytes.Repeat([]byte{0x11}, common.AddressHashSize)
	stake := bytes.Repeat([]byte{0x22}, common.AddressHashSize)
	addr, err := common.NewAddressFromParts(common.AddressTypeKeyKey, common.AddressNetworkTestnet, payment, stake)
	if err != nil {
		t.Fatal(err)
	}
	amount := int64(9)
	original := maybeDatum{
		Amount:  &amount,
		Address: &addr,
		Nested:  &maybeInner{Name: "order"},
	}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	constr := pd.(*data.Constr)
	if len(constr.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(constr.Fields))
	}
	for i, field := range constr.Fields {
		got, ok := field.(*data.Constr)
		if !ok || !constrTagEqual(got.Tag, 0) || len(got.Fields) != 1 {
			t.Fatalf("field %d: expected Just, got %#v", i, field)
		}
	}

	var decoded maybeDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Amount == nil || *decoded.Amount != 9 || decoded.Address == nil || decoded.Nested == nil || decoded.Nested.Name != "order" {
		t.Fatalf("unexpected decoded value: %+v", decoded)
	}
	if decoded.Address.Type() != common.AddressTypeKeyKey {
		t.Fatalf("address type %d", decoded.Address.Type())
	}

	empty := maybeDatum{}
	pd, err = MarshalPlutus(&empty)
	if err != nil {
		t.Fatal(err)
	}
	constr = pd.(*data.Constr)
	for i, field := range constr.Fields {
		got, ok := field.(*data.Constr)
		if !ok || !constrTagEqual(got.Tag, 1) || len(got.Fields) != 0 {
			t.Fatalf("field %d: expected Nothing, got %#v", i, field)
		}
	}
	decoded = maybeDatum{}
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Amount != nil || decoded.Address != nil || decoded.Nested != nil {
		t.Fatalf("expected nils, got %+v", decoded)
	}
}

func TestMaybeAddressCoversShelleyForms(t *testing.T) {
	payment := bytes.Repeat([]byte{0x11}, common.AddressHashSize)
	stake := bytes.Repeat([]byte{0x22}, common.AddressHashSize)
	script := bytes.Repeat([]byte{0x33}, common.AddressHashSize)
	pointer := append([]byte{common.AddressTypeKeyPointer << 4}, payment...)
	pointer = append(pointer, 1, 2, 3)
	pointerAddr, err := common.NewAddressFromBytes(pointer)
	if err != nil {
		t.Fatal(err)
	}

	cases := make([]common.Address, 1, 7)
	cases[0] = pointerAddr
	for _, built := range []struct {
		kind  uint8
		pay   []byte
		stake []byte
	}{
		{common.AddressTypeKeyKey, payment, stake},
		{common.AddressTypeScriptKey, script, stake},
		{common.AddressTypeKeyScript, payment, script},
		{common.AddressTypeScriptScript, script, script},
		{common.AddressTypeKeyNone, payment, nil},
		{common.AddressTypeScriptNone, script, nil},
	} {
		addr, err := common.NewAddressFromParts(built.kind, common.AddressNetworkTestnet, built.pay, built.stake)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, addr)
	}

	for _, addr := range cases {
		original := maybeDatum{Address: &addr}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatalf("type %d: marshal: %v", addr.Type(), err)
		}
		outer := pd.(*data.Constr)
		just, ok := outer.Fields[1].(*data.Constr)
		if !ok || !constrTagEqual(just.Tag, 0) || len(just.Fields) != 1 {
			t.Fatalf("type %d: expected Just Address, got %#v", addr.Type(), outer.Fields[1])
		}
		got, err := data.Encode(just.Fields[0])
		if err != nil {
			t.Fatal(err)
		}
		want, err := data.Encode(addr.ToPlutusData())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("type %d: Maybe did not wrap the address encoding", addr.Type())
		}
		var decoded maybeDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatalf("type %d: unmarshal: %v", addr.Type(), err)
		}
		if decoded.Address == nil || decoded.Address.Type() != addr.Type() {
			t.Fatalf("type %d decoded as %#v", addr.Type(), decoded.Address)
		}
	}
}

func TestMaybeRejectsUnknownConstructor(t *testing.T) {
	bad := data.NewConstr(0,
		data.NewConstr(2),
		data.NewConstr(1),
		data.NewConstr(1),
	)
	var decoded maybeDatum
	if err := UnmarshalPlutus(bad, &decoded); err == nil {
		t.Fatal("expected unknown Maybe constructor to fail")
	}
}

type maybeValueQty int64

func (q maybeValueQty) ToPlutusData() (data.PlutusData, error) {
	return data.NewInteger(big.NewInt(int64(q) + 100)), nil
}

func (q maybeValueQty) FromPlutusData(pd data.PlutusData, res any) error {
	integer, ok := pd.(*data.Integer)
	if !ok {
		return fmt.Errorf("expected Integer, got %T", pd)
	}
	target, ok := res.(*maybeValueQty)
	if !ok {
		return fmt.Errorf("expected *maybeValueQty, got %T", res)
	}
	*target = maybeValueQty(integer.Inner.Int64() - 100)
	return nil
}

type maybePtrQty int64

func (q *maybePtrQty) ToPlutusData() (data.PlutusData, error) {
	return data.NewInteger(big.NewInt(int64(*q) + 100)), nil
}

func (q *maybePtrQty) FromPlutusData(pd data.PlutusData, res any) error {
	integer, ok := pd.(*data.Integer)
	if !ok {
		return fmt.Errorf("expected Integer, got %T", pd)
	}
	target, ok := res.(*maybePtrQty)
	if !ok {
		return fmt.Errorf("expected *maybePtrQty, got %T", res)
	}
	*target = maybePtrQty(integer.Inner.Int64() - 100)
	return nil
}

type maybeCustomDatum struct {
	_   struct{}       `plutusType:"DefList" plutusConstr:"0"`
	Val *maybeValueQty `plutusType:"Maybe"`
	Ptr *maybePtrQty   `plutusType:"Maybe"`
}

func TestMaybeCustomCodecs(t *testing.T) {
	val := maybeValueQty(1)
	ptr := maybePtrQty(1)
	pd, err := MarshalPlutus(&maybeCustomDatum{Val: &val, Ptr: &ptr})
	if err != nil {
		t.Fatal(err)
	}
	for i, field := range pd.(*data.Constr).Fields {
		just := field.(*data.Constr)
		integer := just.Fields[0].(*data.Integer)
		if integer.Inner.Cmp(big.NewInt(101)) != 0 {
			t.Fatalf("field %d encoded %s, want 101", i, integer.Inner)
		}
	}
	var decoded maybeCustomDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Val == nil || *decoded.Val != 1 || decoded.Ptr == nil || *decoded.Ptr != 1 {
		t.Fatalf("decoded %+v", decoded)
	}
}

type maybeBigDatum struct {
	_ struct{} `plutusType:"DefList" plutusConstr:"0"`
	N *big.Int `plutusType:"Maybe"`
}

func TestMaybeBigInt(t *testing.T) {
	for _, want := range []*big.Int{big.NewInt(5), big.NewInt(0), big.NewInt(-7)} {
		original := maybeBigDatum{N: new(big.Int).Set(want)}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatal(err)
		}
		just := pd.(*data.Constr).Fields[0].(*data.Constr)
		if !constrTagEqual(just.Tag, 0) || just.Fields[0].(*data.Integer).Inner.Cmp(want) != 0 {
			t.Fatalf("Just %s encoded as %#v", want, just)
		}
		var decoded maybeBigDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.N == nil || decoded.N.Cmp(want) != 0 {
			t.Fatalf("decoded %v, want %s", decoded.N, want)
		}
	}
	pd, err := MarshalPlutus(&maybeBigDatum{})
	if err != nil {
		t.Fatal(err)
	}
	nothing := pd.(*data.Constr).Fields[0].(*data.Constr)
	if !constrTagEqual(nothing.Tag, 1) || len(nothing.Fields) != 0 {
		t.Fatalf("nil *big.Int: %#v", nothing)
	}
}

type maybeScalarDatum struct {
	_ struct{} `plutusType:"DefList" plutusConstr:"0"`
	I *int64   `plutusType:"Maybe"`
	U *uint64  `plutusType:"Maybe"`
	S *string  `plutusType:"Maybe"`
	B *[]byte  `plutusType:"Maybe"`
}

func TestMaybeScalarsIncludeZeroAndEmpty(t *testing.T) {
	i := int64(0)
	u := uint64(0)
	s := ""
	b := []byte{}
	original := maybeScalarDatum{I: &i, U: &u, S: &s, B: &b}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range pd.(*data.Constr).Fields {
		just := field.(*data.Constr)
		if !constrTagEqual(just.Tag, 0) || len(just.Fields) != 1 {
			t.Fatalf("zero scalar was not Just: %#v", field)
		}
	}
	var decoded maybeScalarDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.I == nil || *decoded.I != 0 || decoded.U == nil || *decoded.U != 0 || decoded.S == nil || *decoded.S != "" || decoded.B == nil || len(*decoded.B) != 0 {
		t.Fatalf("decoded %+v", decoded)
	}
}
