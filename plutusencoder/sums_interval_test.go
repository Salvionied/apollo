package plutusencoder

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"strings"
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
	if !constrTagEqual(outer.Fields[1].(*data.Constr).Tag, 1) {
		t.Fatal("expected datum hash as constructor 1")
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
	err := UnmarshalPlutus(badTag, &decoded)
	if err == nil || !strings.Contains(err.Error(), "got tag 3 with 0 fields") {
		t.Fatalf("expected OutputDatum constructor 3 and its field count, got: %v", err)
	}
}

// pointerQuantity encodes 1 as integer 101 through a pointer receiver.
type pointerQuantity int64

func (q *pointerQuantity) ToPlutusData() (data.PlutusData, error) {
	return data.NewInteger(big.NewInt(int64(*q) + 100)), nil
}

func (q *pointerQuantity) FromPlutusData(pd data.PlutusData, res any) error {
	integer, ok := pd.(*data.Integer)
	if !ok || integer.Inner == nil || !integer.Inner.IsInt64() {
		return fmt.Errorf("expected Integer, got %T", pd)
	}
	target, ok := res.(*pointerQuantity)
	if !ok {
		return fmt.Errorf("expected *pointerQuantity result, got %T", res)
	}
	*target = pointerQuantity(integer.Inner.Int64() - 100)
	return nil
}

type codecChoice struct {
	Left  *customQuantity
	Right *pointerQuantity
}

type codecDatum struct {
	_      struct{}    `plutusType:"DefList" plutusConstr:"0"`
	Choice codecChoice `plutusType:"Either"`
}

func TestEitherCustomCodecReceivers(t *testing.T) {
	t.Run("value receiver", func(t *testing.T) {
		qty := customQuantity(1)
		original := codecDatum{Choice: codecChoice{Left: &qty}}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatal(err)
		}
		either := nestedConstr(t, pd)
		if !constrTagEqual(either.Tag, 0) {
			t.Fatal("expected value-receiver codec on Left, constructor 0")
		}
		if got := constrInteger(t, either).Int64(); got != 101 {
			t.Fatalf("encoded quantity = %d, want 101", got)
		}

		var decoded codecDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Choice.Left == nil || *decoded.Choice.Left != 1 || decoded.Choice.Right != nil {
			t.Fatalf("unexpected value-receiver decode: %+v", decoded.Choice)
		}
	})

	t.Run("pointer receiver", func(t *testing.T) {
		qty := pointerQuantity(1)
		original := codecDatum{Choice: codecChoice{Right: &qty}}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatal(err)
		}
		either := nestedConstr(t, pd)
		if !constrTagEqual(either.Tag, 1) {
			t.Fatal("expected pointer-receiver codec on Right, constructor 1")
		}
		if got := constrInteger(t, either).Int64(); got != 101 {
			t.Fatalf("encoded quantity = %d, want 101", got)
		}

		var decoded codecDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Choice.Right == nil || *decoded.Choice.Right != 1 || decoded.Choice.Left != nil {
			t.Fatalf("unexpected pointer-receiver decode: %+v", decoded.Choice)
		}
	})
}

type bigChoice struct {
	Left  *big.Int
	Right *big.Int
}

type bigDatum struct {
	_      struct{}  `plutusType:"DefList" plutusConstr:"0"`
	Choice bigChoice `plutusType:"Either"`
}

func TestEitherBigIntValues(t *testing.T) {
	for _, raw := range []int64{5, 0, -9} {
		t.Run(fmt.Sprintf("left %d", raw), func(t *testing.T) {
			value := big.NewInt(raw)
			original := bigDatum{Choice: bigChoice{Left: value}}
			pd, err := MarshalPlutus(&original)
			if err != nil {
				t.Fatal(err)
			}
			either := nestedConstr(t, pd)
			if !constrTagEqual(either.Tag, 0) {
				t.Fatal("expected set Left *big.Int as constructor 0")
			}
			if _, isList := either.Fields[0].(*data.List); isList {
				t.Fatal("big.Int encoded as an empty list")
			}
			if constrInteger(t, either).Cmp(value) != 0 {
				t.Fatalf("encoded integer = %s, want %s", constrInteger(t, either), value)
			}

			var decoded bigDatum
			if err := UnmarshalPlutus(pd, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Choice.Left == nil || decoded.Choice.Left.Cmp(value) != 0 || decoded.Choice.Right != nil {
				t.Fatalf("unexpected *big.Int decode: left=%v right=%v", decoded.Choice.Left, decoded.Choice.Right)
			}
		})
	}

	t.Run("nil left selects right", func(t *testing.T) {
		value := big.NewInt(5)
		original := bigDatum{Choice: bigChoice{Right: value}}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatal(err)
		}
		either := nestedConstr(t, pd)
		if !constrTagEqual(either.Tag, 1) {
			t.Fatal("nil *big.Int Left must select Right, not encode as integer 0")
		}
		if constrInteger(t, either).Cmp(value) != 0 {
			t.Fatalf("encoded Right = %s, want 5", constrInteger(t, either))
		}

		var decoded bigDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Choice.Left != nil || decoded.Choice.Right == nil || decoded.Choice.Right.Cmp(value) != 0 {
			t.Fatalf("unexpected nil-Left decode: left=%v right=%v", decoded.Choice.Left, decoded.Choice.Right)
		}
	})
}

type eitherOnlyDatum struct {
	_      struct{}     `plutusType:"DefList" plutusConstr:"0"`
	Choice eitherChoice `plutusType:"Either"`
}

func TestEitherNilSelection(t *testing.T) {
	t.Run("nil left with right set", func(t *testing.T) {
		right := "ok"
		original := eitherOnlyDatum{Choice: eitherChoice{Right: &right}}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatal(err)
		}
		if !constrTagEqual(nestedConstr(t, pd).Tag, 1) {
			t.Fatal("expected Right as constructor 1")
		}
		var decoded eitherOnlyDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Choice.Left != nil || decoded.Choice.Right == nil || *decoded.Choice.Right != "ok" {
			t.Fatalf("unexpected selection: %+v", decoded.Choice)
		}
	})

	t.Run("nil right with left set", func(t *testing.T) {
		left := int64(5)
		original := eitherOnlyDatum{Choice: eitherChoice{Left: &left}}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatal(err)
		}
		either := nestedConstr(t, pd)
		if !constrTagEqual(either.Tag, 0) {
			t.Fatal("expected Left as constructor 0")
		}
		if constrInteger(t, either).Cmp(big.NewInt(5)) != 0 {
			t.Fatalf("encoded Left = %s, want 5", constrInteger(t, either))
		}
		var decoded eitherOnlyDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Choice.Right != nil || decoded.Choice.Left == nil || *decoded.Choice.Left != 5 {
			t.Fatalf("unexpected selection: %+v", decoded.Choice)
		}
	})

	t.Run("both set", func(t *testing.T) {
		left := int64(1)
		right := "two"
		_, err := MarshalPlutus(&eitherOnlyDatum{Choice: eitherChoice{Left: &left, Right: &right}})
		if err == nil || !strings.Contains(err.Error(), "one of Left or Right") {
			t.Fatalf("expected both sides set to fail, got: %v", err)
		}
	})

	t.Run("both nil", func(t *testing.T) {
		_, err := MarshalPlutus(&eitherOnlyDatum{})
		if err == nil || !strings.Contains(err.Error(), "one of Left or Right") {
			t.Fatalf("expected both sides nil to fail, got: %v", err)
		}
	})
}

func TestEitherConstructorWidth(t *testing.T) {
	widths := []struct {
		tag    uint64
		fields int
	}{
		{tag: 0, fields: 0},
		{tag: 0, fields: 2},
		{tag: 1, fields: 0},
		{tag: 1, fields: 2},
	}
	for _, tt := range widths {
		t.Run(fmt.Sprintf("tag %d fields %d", tt.tag, tt.fields), func(t *testing.T) {
			payload := make([]data.PlutusData, tt.fields)
			for i := range payload {
				payload[i] = data.NewInteger(big.NewInt(1))
			}
			pd := data.NewConstr(0, data.NewConstr(tt.tag, payload...))
			var decoded eitherOnlyDatum
			err := UnmarshalPlutus(pd, &decoded)
			want := fmt.Sprintf("got tag %d with %d fields", tt.tag, tt.fields)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error %v, want substring %q", err, want)
			}
		})
	}

	t.Run("non-constructor", func(t *testing.T) {
		pd := data.NewConstr(0, data.NewInteger(big.NewInt(1)))
		var decoded eitherOnlyDatum
		err := UnmarshalPlutus(pd, &decoded)
		if err == nil || !strings.Contains(err.Error(), "*data.Integer") || strings.Contains(err.Error(), "got tag") {
			t.Fatalf("expected dynamic type, got: %v", err)
		}
	})
}

type hexChoice struct {
	Left  *string `plutusType:"HexString"`
	Right *int64
}

type hexDatum struct {
	_      struct{}  `plutusType:"DefList" plutusConstr:"0"`
	Choice hexChoice `plutusType:"Either"`
}

func TestEitherTaggedFieldUsesFieldCodec(t *testing.T) {
	raw := "ff"
	original := hexDatum{Choice: hexChoice{Left: &raw}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	either := nestedConstr(t, pd)
	bs, ok := either.Fields[0].(*data.ByteString)
	if !ok || len(bs.Inner) != 1 || bs.Inner[0] != 0xff {
		t.Fatalf("tagged HexString encoded as %v", either.Fields)
	}

	var decoded hexDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Choice.Left == nil || *decoded.Choice.Left != "ff" || decoded.Choice.Right != nil {
		t.Fatalf("unexpected tagged decode: %+v", decoded.Choice)
	}
}

type eitherPoint struct {
	_ struct{} `plutusType:"DefList" plutusConstr:"0"`
	X int64    `plutusType:"Int"`
}

type pointChoice struct {
	Left  *eitherPoint
	Right *int64
}

type pointDatum struct {
	_      struct{}    `plutusType:"DefList" plutusConstr:"0"`
	Choice pointChoice `plutusType:"Either"`
}

func TestEitherStructElementRoundTrip(t *testing.T) {
	original := pointDatum{Choice: pointChoice{Left: &eitherPoint{X: 4}}}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded pointDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Choice.Left == nil || decoded.Choice.Left.X != 4 || decoded.Choice.Right != nil {
		t.Fatalf("unexpected struct decode: %+v", decoded.Choice)
	}
}

type outputDatumOnly struct {
	_     struct{}    `plutusType:"DefList" plutusConstr:"0"`
	Datum OutputDatum `plutusType:"OutputDatum"`
}

func TestOutputDatumConstructorsRoundTrip(t *testing.T) {
	hash := bytes.Repeat([]byte{0xab}, common.Blake2b256Size)
	inline := data.NewInteger(big.NewInt(9))
	cases := []struct {
		name  string
		datum OutputDatum
		tag   uint64
	}{
		{name: "none", datum: OutputDatum{}, tag: 0},
		{name: "inline", datum: OutputDatum{Inline: inline}, tag: 2},
		{name: "hash", datum: OutputDatum{Hash: hash}, tag: 1},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			pd, err := MarshalPlutus(&outputDatumOnly{Datum: tt.datum})
			if err != nil {
				t.Fatal(err)
			}
			got := nestedConstr(t, pd)
			if !constrTagEqual(got.Tag, tt.tag) {
				t.Fatalf("constructor tag = %s, want %d", got.Tag, tt.tag)
			}
			var decoded outputDatumOnly
			if err := UnmarshalPlutus(pd, &decoded); err != nil {
				t.Fatal(err)
			}
			switch tt.tag {
			case 0:
				if len(got.Fields) != 0 || decoded.Datum.Inline != nil || len(decoded.Datum.Hash) != 0 {
					t.Fatalf("unexpected none datum: wire=%v decoded=%+v", got.Fields, decoded.Datum)
				}
			case 1:
				if decoded.Datum.Inline != nil || !bytes.Equal(decoded.Datum.Hash, hash) {
					t.Fatalf("unexpected hash datum: %+v", decoded.Datum)
				}
				want := data.NewConstr(1, data.NewByteString(hash))
				gotBytes, err := data.Encode(got)
				if err != nil {
					t.Fatal(err)
				}
				wantBytes, err := data.Encode(want)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(gotBytes, wantBytes) {
					t.Fatal("hash datum does not match constructor 1 of the hash bytes")
				}
			case 2:
				decodedInline, ok := decoded.Datum.Inline.(*data.Integer)
				if !ok || decodedInline.Inner.Cmp(big.NewInt(9)) != 0 || len(decoded.Datum.Hash) != 0 {
					t.Fatalf("unexpected inline datum: %+v", decoded.Datum)
				}
				want := data.NewConstr(2, inline)
				gotBytes, err := data.Encode(got)
				if err != nil {
					t.Fatal(err)
				}
				wantBytes, err := data.Encode(want)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(gotBytes, wantBytes) {
					t.Fatal("inline datum does not match constructor 2 of the datum")
				}
			default:
				t.Fatalf("untested output datum tag %d", tt.tag)
			}
		})
	}

	t.Run("opposite shapes", func(t *testing.T) {
		hashAsInline := data.NewConstr(0, data.NewConstr(2, data.NewByteString(hash)))
		var decoded outputDatumOnly
		if err := UnmarshalPlutus(hashAsInline, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Datum.Hash != nil || decoded.Datum.Inline == nil {
			t.Fatalf("constructor 2 decoded as a hash: %+v", decoded.Datum)
		}
		integerAsHash := data.NewConstr(0, data.NewConstr(1, data.NewInteger(big.NewInt(5))))
		if err := UnmarshalPlutus(integerAsHash, &decoded); err == nil {
			t.Fatal("expected constructor 1 of an integer to fail")
		}
	})

	t.Run("empty hash", func(t *testing.T) {
		if _, err := MarshalPlutus(&outputDatumOnly{Datum: OutputDatum{Hash: []byte{}}}); err == nil {
			t.Fatal("expected a non-nil empty hash to fail")
		}
	})

	t.Run("constructor 3", func(t *testing.T) {
		pd := data.NewConstr(0, data.NewConstr(3))
		var decoded outputDatumOnly
		err := UnmarshalPlutus(pd, &decoded)
		if err == nil || !strings.Contains(err.Error(), "got tag 3 with 0 fields") {
			t.Fatalf("expected constructor 3 and its field count, got: %v", err)
		}
	})

	t.Run("non-constructor", func(t *testing.T) {
		pd := data.NewConstr(0, data.NewByteString([]byte{0x01}))
		var decoded outputDatumOnly
		err := UnmarshalPlutus(pd, &decoded)
		if err == nil || !strings.Contains(err.Error(), "*data.ByteString") || strings.Contains(err.Error(), "got tag") {
			t.Fatalf("expected dynamic type, got: %v", err)
		}
	})
}

type intervalDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"0"`
	Window Interval `plutusType:"Interval"`
}

func TestIntervalBoundRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		window Interval
	}{
		{
			name: "neg inf and finite",
			window: Interval{
				Lower: IntervalBound{Kind: BoundNegInf, Closed: false},
				Upper: IntervalBound{Kind: BoundFinite, Value: -42, Closed: true},
			},
		},
		{
			name: "finite and pos inf",
			window: Interval{
				Lower: IntervalBound{Kind: BoundFinite, Value: math.MaxInt64, Closed: true},
				Upper: IntervalBound{Kind: BoundPosInf, Closed: false},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			original := intervalDatum{Window: tt.window}
			pd, err := MarshalPlutus(&original)
			if err != nil {
				t.Fatal(err)
			}
			var decoded intervalDatum
			if err := UnmarshalPlutus(pd, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Window != original.Window {
				t.Fatalf("interval = %+v, want %+v", decoded.Window, original.Window)
			}
		})
	}

	t.Run("integer does not fit in int64", func(t *testing.T) {
		huge := new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
		lower := data.NewConstr(0,
			data.NewConstr(1, data.NewInteger(huge)),
			data.NewConstr(1),
		)
		upper := data.NewConstr(0,
			data.NewConstr(2),
			data.NewConstr(0),
		)
		pd := data.NewConstr(0, data.NewConstr(0, lower, upper))
		var decoded intervalDatum
		err := UnmarshalPlutus(pd, &decoded)
		if err == nil || !strings.Contains(err.Error(), "does not fit in int64") {
			t.Fatalf("expected int64 overflow, got: %v", err)
		}
	})

	t.Run("non-constructor", func(t *testing.T) {
		pd := data.NewConstr(0, data.NewList())
		var decoded intervalDatum
		err := UnmarshalPlutus(pd, &decoded)
		if err == nil || !strings.Contains(err.Error(), "*data.List") || strings.Contains(err.Error(), "got tag") {
			t.Fatalf("expected dynamic type, got: %v", err)
		}
	})
}

func nestedConstr(t *testing.T, pd data.PlutusData) *data.Constr {
	t.Helper()
	outer, ok := pd.(*data.Constr)
	if !ok || len(outer.Fields) != 1 {
		t.Fatalf("expected 1-field constructor, got %T", pd)
	}
	inner, ok := outer.Fields[0].(*data.Constr)
	if !ok {
		t.Fatalf("expected nested constructor, got %T", outer.Fields[0])
	}
	return inner
}

func constrInteger(t *testing.T, constr *data.Constr) *big.Int {
	t.Helper()
	if len(constr.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(constr.Fields))
	}
	integer, ok := constr.Fields[0].(*data.Integer)
	if !ok || integer.Inner == nil {
		t.Fatalf("expected Integer, got %T", constr.Fields[0])
	}
	return integer.Inner
}

func TestOutputDatumAndIntervalRejectTypedNil(t *testing.T) {
	hash := bytes.Repeat([]byte{0xab}, common.Blake2b256Size)
	closed := data.NewConstr(1)
	finite := data.NewConstr(1, data.NewInteger(big.NewInt(5)))
	lower := data.NewConstr(0, data.NewConstr(0), closed)
	cases := []struct {
		name string
		dst  any
		pd   data.PlutusData
	}{
		{
			name: "datum hash",
			dst:  &outputDatumOnly{},
			pd:   data.NewConstr(0, data.NewConstr(1, (*data.ByteString)(nil))),
		},
		{
			name: "inline datum",
			dst:  &outputDatumOnly{},
			pd:   data.NewConstr(0, data.NewConstr(2, (*data.Integer)(nil))),
		},
		{
			name: "finite bound",
			dst:  &intervalDatum{},
			pd: data.NewConstr(0, data.NewConstr(0,
				data.NewConstr(0, data.NewConstr(1, (*data.Integer)(nil)), closed),
				data.NewConstr(0, finite, closed),
			)),
		},
		{
			name: "closure",
			dst:  &intervalDatum{},
			pd: data.NewConstr(0, data.NewConstr(0,
				lower,
				data.NewConstr(0, finite, (*data.Constr)(nil)),
			)),
		},
	}
	for _, tc := range cases {
		if err := UnmarshalPlutus(tc.pd, tc.dst); err == nil {
			t.Errorf("%s: typed nil decoded", tc.name)
		}
	}
	var round outputDatumOnly
	if err := UnmarshalPlutus(data.NewConstr(0, data.NewConstr(1, data.NewByteString(hash))), &round); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(round.Datum.Hash, hash) {
		t.Fatalf("hash = %x", round.Datum.Hash)
	}
}
