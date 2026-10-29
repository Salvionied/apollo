package plutusencoder

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

type valueDatum struct {
	_      struct{} `plutusType:"DefList" plutusConstr:"0"`
	Locked Value    `plutusType:"Value"`
}

func TestValueAdaOnlyNativeOnlyAndCombined(t *testing.T) {
	policyA, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x22}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	policyB, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x11}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	multi := common.NewMultiAsset(map[common.Blake2b224]map[cbor.ByteString]uint64{
		policyA: {cbor.NewByteString([]byte("BETA")): 5},
		policyB: {cbor.NewByteString([]byte("ALPHA")): 3},
	})

	adaOnly := valueDatum{Locked: Value{Lovelace: 2_000_000}}
	pd, err := MarshalPlutus(&adaOnly)
	if err != nil {
		t.Fatal(err)
	}
	var decoded valueDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Locked.Lovelace != 2_000_000 || len(decoded.Locked.Assets.Policies()) != 0 {
		t.Fatalf("unexpected ada-only decode: %+v", decoded.Locked)
	}

	nativeOnly := valueDatum{Locked: Value{Assets: multi}}
	pd, err = MarshalPlutus(&nativeOnly)
	if err != nil {
		t.Fatal(err)
	}
	outer := pd.(*data.Constr)
	gotNative, err := data.Encode(outer.Fields[0])
	if err != nil {
		t.Fatal(err)
	}
	wantNative, err := data.Encode(multi.ToPlutusData())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotNative, wantNative) {
		t.Fatal("native-only Value does not match MultiAsset.ToPlutusData")
	}

	combined := valueDatum{Locked: Value{Lovelace: 5_000_000, Assets: multi}}
	pd, err = MarshalPlutus(&combined)
	if err != nil {
		t.Fatal(err)
	}
	decoded = valueDatum{}
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Locked.Lovelace != 5_000_000 {
		t.Fatalf("lovelace = %d, want 5000000", decoded.Locked.Lovelace)
	}
	if decoded.Locked.Assets.Asset(policyA, []byte("BETA")) != 5 || decoded.Locked.Assets.Asset(policyB, []byte("ALPHA")) != 3 {
		t.Fatalf("unexpected decoded assets: %+v", decoded.Locked.Assets)
	}
}

func TestV4ValueTaggedRoundTrip(t *testing.T) {
	type v4ValueDatum struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Locked Value    `plutusType:"V4Value"`
	}
	policy, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x44}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	multi := common.NewMultiAsset(map[common.Blake2b224]map[cbor.ByteString]uint64{
		policy: {cbor.NewByteString([]byte("TOK")): 9},
	})
	original := v4ValueDatum{Locked: Value{Lovelace: 3_000_000, Assets: multi}}

	pd, err := MarshalPlutus(&original)
	if err != nil {
		t.Fatal(err)
	}
	outer := pd.(*data.Constr)
	if _, ok := outer.Fields[0].(*data.Value); !ok {
		t.Fatalf("expected *data.Value (CBOR tag 1401), got %T", outer.Fields[0])
	}
	raw, err := data.Encode(pd)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := data.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}

	var decodedV4 v4ValueDatum
	if err := UnmarshalPlutus(wire, &decodedV4); err != nil {
		t.Fatal(err)
	}
	if decodedV4.Locked.Lovelace != 3_000_000 || decodedV4.Locked.Assets.Asset(policy, []byte("TOK")) != 9 {
		t.Fatalf("unexpected V4Value decode: %+v", decodedV4.Locked)
	}

	var decodedMap valueDatum
	if err := UnmarshalPlutus(wire, &decodedMap); err != nil {
		t.Fatal(err)
	}
	if decodedMap.Locked.Lovelace != 3_000_000 {
		t.Fatalf("Value tag must also decode *data.Value, got %+v", decodedMap.Locked)
	}
}

func TestValueRejectsDuplicatesAndNegativeQuantity(t *testing.T) {
	policyBytes := bytes.Repeat([]byte{0x33}, common.Blake2b224Size)
	dupPolicy := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{
			data.NewByteString(policyBytes),
			data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("A")), data.NewInteger(big.NewInt(1))}}),
		},
		{
			data.NewByteString(policyBytes),
			data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("B")), data.NewInteger(big.NewInt(1))}}),
		},
	}))
	var out valueDatum
	if err := UnmarshalPlutus(dupPolicy, &out); err == nil {
		t.Fatal("expected duplicate policy to fail")
	}

	negative := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{
			data.NewByteString(policyBytes),
			data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("A")), data.NewInteger(big.NewInt(-1))}}),
		},
	}))
	if err := UnmarshalPlutus(negative, &out); err == nil {
		t.Fatal("expected negative quantity to fail")
	}
}

func TestValueZeroQuantityIsKeptOnDecodeAndDroppedOnEncode(t *testing.T) {
	policyBytes := bytes.Repeat([]byte{0xaa}, common.Blake2b224Size)
	zeroAda := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{
			data.NewByteString(nil),
			data.NewMap([][2]data.PlutusData{{data.NewByteString(nil), data.NewInteger(big.NewInt(0))}}),
		},
	}))
	var decoded valueDatum
	if err := UnmarshalPlutus(zeroAda, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Locked.Lovelace != 0 {
		t.Fatalf("lovelace = %d, want 0", decoded.Locked.Lovelace)
	}
	encoded, err := MarshalPlutus(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.(*data.Constr).Fields[0].(*data.Map).Pairs) != 0 {
		t.Fatal("encoding a zero lovelace quantity must omit it")
	}

	zeroAsset := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{
			data.NewByteString(policyBytes),
			data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("Z")), data.NewInteger(big.NewInt(0))}}),
		},
	}))
	decoded = valueDatum{}
	if err := UnmarshalPlutus(zeroAsset, &decoded); err != nil {
		t.Fatal(err)
	}
	policy, err := common.NewBlake2b224Checked(policyBytes)
	if err != nil {
		t.Fatal(err)
	}
	names := decoded.Locked.Assets.Assets(policy)
	if len(names) != 1 || decoded.Locked.Assets.Asset(policy, []byte("Z")) != 0 {
		t.Fatalf("zero asset was not kept: %q", names)
	}
	encoded, err = MarshalPlutus(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.(*data.Constr).Fields[0].(*data.Map).Pairs) != 0 {
		t.Fatal("encoding a zero asset quantity must omit it")
	}
}

func TestValueLenientMapBecomesCanonical(t *testing.T) {
	high := bytes.Repeat([]byte{0x22}, common.Blake2b224Size)
	low := bytes.Repeat([]byte{0x11}, common.Blake2b224Size)
	unsorted := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{data.NewByteString(high), data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("B")), data.NewInteger(big.NewInt(2))}})},
		{data.NewByteString(low), data.NewMap([][2]data.PlutusData{{data.NewByteString([]byte("A")), data.NewInteger(big.NewInt(1))}})},
	}))
	var decoded valueDatum
	if err := UnmarshalPlutus(unsorted, &decoded); err != nil {
		t.Fatal(err)
	}
	encoded, err := MarshalPlutus(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	pairs := encoded.(*data.Constr).Fields[0].(*data.Map).Pairs
	if len(pairs) != 2 || !bytes.Equal(pairs[0][0].(*data.ByteString).Inner, low) {
		t.Fatal("re-encode did not sort policy ids")
	}

	emptyInner := data.NewConstr(0, data.NewMap([][2]data.PlutusData{
		{data.NewByteString(low), data.NewMap(nil)},
	}))
	decoded = valueDatum{}
	if err := UnmarshalPlutus(emptyInner, &decoded); err != nil {
		t.Fatal(err)
	}
	encoded, err = MarshalPlutus(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded.(*data.Constr).Fields[0].(*data.Map).Pairs) != 0 {
		t.Fatal("re-encode kept an empty token map")
	}
}

func TestValueRejectsLongTokenNameAndGoldenHex(t *testing.T) {
	longName := bytes.Repeat([]byte{0x61}, 33)
	policy, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x44}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	tooLong := Value{Assets: common.NewMultiAsset(map[common.Blake2b224]map[cbor.ByteString]uint64{
		policy: {cbor.NewByteString(longName): 1},
	})}
	if _, err := MarshalPlutus(&valueDatum{Locked: tooLong}); err == nil {
		t.Fatal("expected a 33-byte token name to fail")
	}

	adaAndAsset := valueDatum{Locked: Value{
		Lovelace: 1,
		Assets: common.NewMultiAsset(map[common.Blake2b224]map[cbor.ByteString]uint64{
			policy: {cbor.NewByteString([]byte("N")): 2},
		}),
	}}
	pd, err := MarshalPlutus(&adaAndAsset)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := data.Encode(pd.(*data.Constr).Fields[0])
	if err != nil {
		t.Fatal(err)
	}
	const want = "a240a14001581c44444444444444444444444444444444444444444444444444444444a1414e02"
	if got := hex.EncodeToString(raw); got != want {
		t.Fatalf("golden hex = %s", got)
	}

	zeroMap := data.NewMap([][2]data.PlutusData{{
		data.NewByteString(nil),
		data.NewMap([][2]data.PlutusData{{data.NewByteString(nil), data.NewInteger(big.NewInt(0))}}),
	}}).(*data.Map)
	tagged := data.NewConstr(0, &data.Value{Inner: zeroMap})
	var v4 struct {
		_      struct{} `plutusType:"DefList" plutusConstr:"0"`
		Locked Value    `plutusType:"V4Value"`
	}
	if err := UnmarshalPlutus(tagged, &v4); err == nil {
		t.Fatal("V4 Value must reject a zero quantity")
	}
}

func TestValueRejectsTypedNilComponents(t *testing.T) {
	policy := bytes.Repeat([]byte{0x11}, common.Blake2b224Size)
	token := data.NewByteString([]byte("N"))
	qty := data.NewInteger(big.NewInt(1))
	asset := data.NewMap([][2]data.PlutusData{{token, qty}})
	cases := []data.PlutusData{
		(*data.Map)(nil),
		(*data.Value)(nil),
		&data.Value{},
		data.NewMap([][2]data.PlutusData{{(*data.ByteString)(nil), asset}}),
		data.NewMap([][2]data.PlutusData{{data.NewByteString(policy), (*data.Map)(nil)}}),
		data.NewMap([][2]data.PlutusData{{data.NewByteString(policy), data.NewMap([][2]data.PlutusData{{(*data.ByteString)(nil), qty}})}}),
		data.NewMap([][2]data.PlutusData{{data.NewByteString(nil), data.NewMap([][2]data.PlutusData{{data.NewByteString(nil), (*data.Integer)(nil)}})}}),
	}
	for i, value := range cases {
		var decoded valueDatum
		if err := UnmarshalPlutus(data.NewConstr(0, value), &decoded); err == nil {
			t.Fatalf("case %d decoded a typed-nil Value component", i)
		}
	}
}
