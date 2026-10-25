package plutusencoder

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

type eitherChoice struct {
	Left  *int64
	Right *string
}

type sumIntervalDatum struct {
	_      struct{}     `plutusType:"DefList" plutusConstr:"0"`
	Choice eitherChoice `plutusType:"Either"`
	Datum  OutputDatum  `plutusType:"OutputDatum"`
	Window Interval     `plutusType:"Interval"`
}

func TestEitherOutputDatumAndIntervalRoundTrip(t *testing.T) {
	leftVal := int64(42)
	hash := bytes.Repeat([]byte{0x55}, common.Blake2b256Size)
	first := sumIntervalDatum{
		Choice: eitherChoice{Left: &leftVal},
		Datum:  OutputDatum{Hash: hash},
		Window: Interval{
			Lower: IntervalBound{Kind: BoundFinite, Value: 1000, Closed: true},
			Upper: IntervalBound{Kind: BoundPosInf, Closed: true},
		},
	}

	pd, err := MarshalPlutus(&first)
	if err != nil {
		t.Fatal(err)
	}
	outer := pd.(*data.Constr)
	if !constrTagEqual(outer.Fields[0].(*data.Constr).Tag, 0) {
		t.Fatal("expected Left as constructor 0")
	}
	if !constrTagEqual(outer.Fields[1].(*data.Constr).Tag, 2) {
		t.Fatal("expected datum hash as constructor 2")
	}

	var decoded sumIntervalDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Choice.Left == nil || *decoded.Choice.Left != 42 || decoded.Choice.Right != nil {
		t.Fatalf("unexpected Either decode: %+v", decoded.Choice)
	}
	if !bytes.Equal(decoded.Datum.Hash, hash) || decoded.Datum.Inline != nil {
		t.Fatalf("unexpected OutputDatum decode: %+v", decoded.Datum)
	}
	if decoded.Window != first.Window {
		t.Fatalf("interval = %+v, want %+v", decoded.Window, first.Window)
	}

	rightVal := "ok"
	inline := data.NewInteger(big.NewInt(9))
	second := sumIntervalDatum{
		Choice: eitherChoice{Right: &rightVal},
		Datum:  OutputDatum{Inline: inline},
		Window: Interval{
			Lower: IntervalBound{Kind: BoundNegInf, Closed: false},
			Upper: IntervalBound{Kind: BoundFinite, Value: 2000, Closed: false},
		},
	}
	pd, err = MarshalPlutus(&second)
	if err != nil {
		t.Fatal(err)
	}
	decoded = sumIntervalDatum{}
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Choice.Right == nil || *decoded.Choice.Right != "ok" || decoded.Choice.Left != nil {
		t.Fatalf("unexpected Right decode: %+v", decoded.Choice)
	}
	if decoded.Datum.Inline == nil || decoded.Window != second.Window {
		t.Fatalf("unexpected second decode: %+v", decoded)
	}

	noneDatum := sumIntervalDatum{
		Choice: eitherChoice{Left: &leftVal},
		Datum:  OutputDatum{},
		Window: first.Window,
	}
	pd, err = MarshalPlutus(&noneDatum)
	if err != nil {
		t.Fatal(err)
	}
	if !constrTagEqual(pd.(*data.Constr).Fields[1].(*data.Constr).Tag, 0) {
		t.Fatal("expected empty OutputDatum as constructor 0")
	}
}

func TestEitherAndOutputDatumRejectInvalidStates(t *testing.T) {
	leftVal := int64(1)
	rightVal := "two"

	if _, err := MarshalPlutus(&sumIntervalDatum{}); err == nil {
		t.Fatal("expected empty Either to fail")
	}
	both := sumIntervalDatum{Choice: eitherChoice{Left: &leftVal, Right: &rightVal}}
	if _, err := MarshalPlutus(&both); err == nil {
		t.Fatal("expected Either with both sides set to fail")
	}

	bothDatum := sumIntervalDatum{
		Choice: eitherChoice{Left: &leftVal},
		Datum:  OutputDatum{Inline: data.NewInteger(big.NewInt(1)), Hash: bytes.Repeat([]byte{0x01}, common.Blake2b256Size)},
	}
	if _, err := MarshalPlutus(&bothDatum); err == nil {
		t.Fatal("expected OutputDatum with both Inline and Hash to fail")
	}

	badTag := data.NewConstr(0,
		data.NewConstr(0, data.NewInteger(big.NewInt(1))),
		data.NewConstr(3),
		data.NewConstr(0,
			data.NewConstr(0, data.NewConstr(0), data.NewConstr(0)),
			data.NewConstr(0, data.NewConstr(2), data.NewConstr(0)),
		),
	)
	var decoded sumIntervalDatum
	if err := UnmarshalPlutus(badTag, &decoded); err == nil {
		t.Fatal("expected OutputDatum constructor 3 to fail")
	}
}
