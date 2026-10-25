package plutusencoder

import (
	"bytes"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

type addressDatum struct {
	_    struct{}       `plutusType:"DefList" plutusConstr:"0"`
	Addr common.Address `plutusType:"Address"`
}

func TestAddressRoundTrip(t *testing.T) {
	payment := bytes.Repeat([]byte{0x11}, common.AddressHashSize)
	stake := bytes.Repeat([]byte{0x22}, common.AddressHashSize)
	script := bytes.Repeat([]byte{0x33}, common.AddressHashSize)

	pointer := append([]byte{common.AddressTypeKeyPointer << 4}, payment...)
	pointer = append(pointer, 1, 2, 3)
	pointerAddr, err := common.NewAddressFromBytes(pointer)
	if err != nil {
		t.Fatal(err)
	}
	scriptPointer := append([]byte{common.AddressTypeScriptPointer << 4}, script...)
	scriptPointer = append(scriptPointer, 4, 5, 6)
	scriptPointerAddr, err := common.NewAddressFromBytes(scriptPointer)
	if err != nil {
		t.Fatal(err)
	}

	cases := []common.Address{}
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
	cases = append(cases, pointerAddr, scriptPointerAddr)

	for _, addr := range cases {
		original := addressDatum{Addr: addr}
		pd, err := MarshalPlutus(&original)
		if err != nil {
			t.Fatalf("type %d: marshal: %v", addr.Type(), err)
		}
		constr := pd.(*data.Constr)
		got, err := data.Encode(constr.Fields[0])
		if err != nil {
			t.Fatal(err)
		}
		want, err := data.Encode(addr.ToPlutusData())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("type %d: encoded address differs from common.Address.ToPlutusData", addr.Type())
		}

		var decoded addressDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatalf("type %d: unmarshal: %v", addr.Type(), err)
		}
		if decoded.Addr.Type() != addr.Type() {
			t.Fatalf("type %d decoded as %d", addr.Type(), decoded.Addr.Type())
		}
		again, err := data.Encode(decoded.Addr.ToPlutusData())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again, want) {
			t.Fatalf("type %d did not round-trip", addr.Type())
		}
	}
}

func TestByronAddressReturnsError(t *testing.T) {
	addr, err := common.NewByronAddressFromParts(common.ByronAddressTypePubkey, bytes.Repeat([]byte{0x44}, common.AddressHashSize), common.ByronAddressAttributes{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = MarshalPlutus(&addressDatum{Addr: addr})
	if err == nil {
		t.Fatal("expected Byron address to be rejected")
	}
}
