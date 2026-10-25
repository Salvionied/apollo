package plutusencoder

import (
	"bytes"
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
