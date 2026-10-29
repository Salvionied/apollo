package plutusencoder

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

const (
	payKeyByte       = 0x11
	payScriptByte    = 0x22
	stakeKeyByte     = 0x33
	stakeScriptByte  = 0x44
	rewardKeyByte    = 0x55
	rewardScriptByte = 0x66
)

func hashPattern(b byte) []byte {
	return bytes.Repeat([]byte{b}, common.AddressHashSize)
}

type addrBare struct {
	_ struct{}       `plutusType:"DefList" plutusConstr:"0"`
	A common.Address `plutusType:"Address"`
}

type addrMaybe struct {
	_ struct{}        `plutusType:"DefList" plutusConstr:"0"`
	A *common.Address `plutusType:"Maybe"`
}

type addrItem struct {
	_ struct{}       `plutusType:"DefList" plutusConstr:"0"`
	A common.Address `plutusType:"Address"`
}

type addrList struct {
	_     struct{}   `plutusType:"DefList" plutusConstr:"0"`
	Items []addrItem `plutusType:"DefList"`
}

type addrMap struct {
	_ struct{}       `plutusType:"Map"`
	A common.Address `plutusType:"Address" plutusKey:"addr"`
}

func mustParts(t *testing.T, kind uint8, network byte, pay, stake []byte) common.Address {
	t.Helper()
	addr, err := common.NewAddressFromParts(kind, network, pay, stake)
	if err != nil {
		t.Fatalf("NewAddressFromParts type %d network %d: %v", kind, network, err)
	}
	return addr
}

func mustPointer(t *testing.T, script bool, network byte, pay []byte, slot, tx, cert uint64) common.Address {
	t.Helper()
	var kind uint8 = common.AddressTypeKeyPointer
	if script {
		kind = common.AddressTypeScriptPointer
	}
	raw := append([]byte{(kind << 4) | network}, pay...)
	raw = append(raw, encodePointer(slot, tx, cert)...)
	addr, err := common.NewAddressFromBytes(raw)
	if err != nil {
		t.Fatalf("pointer type %d network %d: %v", kind, network, err)
	}
	return addr
}

func assertBech32RoundTrip(t *testing.T, addr common.Address) {
	t.Helper()
	parsed, err := common.NewAddress(addr.String())
	if err != nil {
		t.Fatalf("NewAddress(%q): %v", addr.String(), err)
	}
	got, err := parsed.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	want, err := addr.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bech32 round trip changed %q", addr.String())
	}
	hrp := "addr"
	switch addr.Type() {
	case common.AddressTypeNoneKey, common.AddressTypeNoneScript:
		hrp = "stake"
	}
	if addr.NetworkId() != uint(common.AddressNetworkMainnet) {
		hrp += "_test"
	}
	if !strings.HasPrefix(addr.String(), hrp) {
		t.Fatalf("address %q: hrp want prefix %s", addr.String(), hrp)
	}
}

func addressPlutusBytes(t *testing.T, pd data.PlutusData) []byte {
	t.Helper()
	encoded, err := data.Encode(pd)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func wantPlutusBytes(t *testing.T, addr common.Address) []byte {
	t.Helper()
	encoded := addr.ToPlutusData()
	if encoded == nil {
		t.Fatal("ToPlutusData returned nil")
	}
	return addressPlutusBytes(t, encoded)
}

func decodeAddress(t *testing.T, pd data.PlutusData, network byte, carrier int) common.Address {
	t.Helper()
	var opts []UnmarshalOption
	if network == common.AddressNetworkMainnet {
		opts = []UnmarshalOption{WithAddressNetwork(common.AddressNetworkMainnet)}
	}
	switch carrier {
	case 0:
		var decoded addrBare
		if err := UnmarshalPlutus(pd, &decoded, opts...); err != nil {
			t.Fatal(err)
		}
		return decoded.A
	case 1:
		var decoded addrMaybe
		if err := UnmarshalPlutus(pd, &decoded, opts...); err != nil {
			t.Fatal(err)
		}
		if decoded.A == nil {
			t.Fatal("Maybe address decoded nil")
		}
		return *decoded.A
	case 2:
		var decoded addrList
		if err := UnmarshalPlutus(pd, &decoded, opts...); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Items) != 1 {
			t.Fatalf("list length %d", len(decoded.Items))
		}
		return decoded.Items[0].A
	case 3:
		var decoded addrMap
		if err := UnmarshalPlutus(pd, &decoded, opts...); err != nil {
			t.Fatal(err)
		}
		return decoded.A
	default:
		t.Fatalf("carrier %d", carrier)
		return common.Address{}
	}
}

func marshalCarrier(t *testing.T, addr common.Address, carrier int) (data.PlutusData, data.PlutusData) {
	t.Helper()
	switch carrier {
	case 0:
		pd, err := MarshalPlutus(&addrBare{A: addr})
		if err != nil {
			t.Fatal(err)
		}
		return pd, pd.(*data.Constr).Fields[0]
	case 1:
		pd, err := MarshalPlutus(&addrMaybe{A: &addr})
		if err != nil {
			t.Fatal(err)
		}
		just := pd.(*data.Constr).Fields[0].(*data.Constr)
		if !constrTagEqual(just.Tag, 0) || len(just.Fields) != 1 {
			t.Fatalf("Maybe address: expected Just, got %#v", just)
		}
		return pd, just.Fields[0]
	case 2:
		pd, err := MarshalPlutus(&addrList{Items: []addrItem{{A: addr}}})
		if err != nil {
			t.Fatal(err)
		}
		list := pd.(*data.Constr).Fields[0].(*data.List)
		item := list.Items[0].(*data.Constr)
		return pd, item.Fields[0]
	case 3:
		pd, err := MarshalPlutus(&addrMap{A: addr})
		if err != nil {
			t.Fatal(err)
		}
		return pd, pd.(*data.Map).Pairs[0][1]
	default:
		t.Fatalf("carrier %d", carrier)
		return nil, nil
	}
}

func assertSameAddress(t *testing.T, got, want common.Address) {
	t.Helper()
	if got.Type() != want.Type() || got.NetworkId() != want.NetworkId() {
		t.Fatalf("got type %d network %d, want type %d network %d", got.Type(), got.NetworkId(), want.Type(), want.NetworkId())
	}
	gotBytes, err := got.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := want.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotBytes, wantBytes) {
		t.Fatalf("address bytes differ for type %d", want.Type())
	}
}

func roundTripCarriers(t *testing.T, addr common.Address, network byte) int {
	t.Helper()
	runs := 0
	for carrier := 0; carrier < 4; carrier++ {
		runs++
		pd, field := marshalCarrier(t, addr, carrier)
		if !bytes.Equal(addressPlutusBytes(t, field), wantPlutusBytes(t, addr)) {
			t.Fatalf("type %d network %d carrier %d: bytes differ from ToPlutusData", addr.Type(), network, carrier)
		}
		got := decodeAddress(t, pd, network, carrier)
		assertSameAddress(t, got, addr)
		if network == common.AddressNetworkTestnet && carrier == 0 {
			var explicit addrBare
			if err := UnmarshalPlutus(pd, &explicit, WithAddressNetwork(common.AddressNetworkTestnet)); err != nil {
				t.Fatal(err)
			}
			assertSameAddress(t, explicit.A, addr)
		}
		if network == common.AddressNetworkMainnet && carrier == 0 {
			var asTestnet addrBare
			if err := UnmarshalPlutus(pd, &asTestnet); err != nil {
				t.Fatal(err)
			}
			if asTestnet.A.NetworkId() != uint(common.AddressNetworkTestnet) || asTestnet.A.Type() != addr.Type() {
				t.Fatalf("default decode of mainnet type %d: network %d type %d", addr.Type(), asTestnet.A.NetworkId(), asTestnet.A.Type())
			}
		}
	}
	return runs
}

func TestPaymentAddressMatrix(t *testing.T) {
	payKey := hashPattern(payKeyByte)
	payScript := hashPattern(payScriptByte)
	stakeKey := hashPattern(stakeKeyByte)
	stakeScript := hashPattern(stakeScriptByte)
	runs := 0
	for _, network := range []byte{common.AddressNetworkTestnet, common.AddressNetworkMainnet} {
		forms := []common.Address{
			mustParts(t, common.AddressTypeKeyKey, network, payKey, stakeKey),
			mustParts(t, common.AddressTypeScriptKey, network, payScript, stakeKey),
			mustParts(t, common.AddressTypeKeyScript, network, payKey, stakeScript),
			mustParts(t, common.AddressTypeScriptScript, network, payScript, stakeScript),
			mustPointer(t, false, network, payKey, 1, 2, 3),
			mustPointer(t, true, network, payScript, 4, 5, 6),
			mustParts(t, common.AddressTypeKeyNone, network, payKey, nil),
			mustParts(t, common.AddressTypeScriptNone, network, payScript, nil),
		}
		if len(forms) != 8 {
			t.Fatalf("payment forms %d", len(forms))
		}
		for _, addr := range forms {
			assertBech32RoundTrip(t, addr)
			runs += roundTripCarriers(t, addr, network)
		}
	}
	if runs != 64 {
		t.Fatalf("payment matrix ran %d cases, want 64", runs)
	}
}

func TestRewardAddressMatrix(t *testing.T) {
	runs := 0
	for _, network := range []byte{common.AddressNetworkTestnet, common.AddressNetworkMainnet} {
		forms := []common.Address{
			mustParts(t, common.AddressTypeNoneKey, network, nil, hashPattern(rewardKeyByte)),
			mustParts(t, common.AddressTypeNoneScript, network, nil, hashPattern(rewardScriptByte)),
		}
		for _, addr := range forms {
			assertBech32RoundTrip(t, addr)
			runs += roundTripCarriers(t, addr, network)
		}
	}
	if runs != 16 {
		t.Fatalf("reward matrix ran %d cases, want 16", runs)
	}
}

func TestPointerAddressCorners(t *testing.T) {
	payKey := hashPattern(payKeyByte)
	payScript := hashPattern(payScriptByte)
	bounds := []uint64{0, math.MaxUint32}
	indexBounds := []uint64{0, math.MaxUint16}
	success := 0
	for _, script := range []bool{false, true} {
		pay := payKey
		if script {
			pay = payScript
		}
		for _, network := range []byte{common.AddressNetworkTestnet, common.AddressNetworkMainnet} {
			for _, slot := range bounds {
				for _, tx := range indexBounds {
					for _, cert := range indexBounds {
						addr := mustPointer(t, script, network, pay, slot, tx, cert)
						if err := common.CheckAddressPointerInRange(addr); err != nil {
							t.Fatal(err)
						}
						success++
						pd, field := marshalCarrier(t, addr, 0)
						if !bytes.Equal(addressPlutusBytes(t, field), wantPlutusBytes(t, addr)) {
							t.Fatal("pointer corner bytes differ from ToPlutusData")
						}
						got := decodeAddress(t, pd, network, 0)
						assertSameAddress(t, got, addr)
						ptr, ok := got.StakingPayload().(common.AddressPayloadPointer)
						if !ok || ptr.Slot != slot || ptr.TxIndex != tx || ptr.CertIndex != cert {
							t.Fatalf("pointer got %#v, want slot %d tx %d cert %d", got.StakingPayload(), slot, tx, cert)
						}
					}
				}
			}
		}
	}
	if success != 32 {
		t.Fatalf("pointer successes %d, want 32", success)
	}

	failures := 0
	past := [][3]int64{
		{int64(1) << 32, 0, 0},
		{0, int64(math.MaxUint16) + 1, 0},
		{0, 0, int64(math.MaxUint16) + 1},
		{int64(1) << 32, int64(math.MaxUint16) + 1, int64(math.MaxUint16) + 1},
	}
	for _, script := range []bool{false, true} {
		payTag := uint64(0)
		pay := payKey
		if script {
			payTag = 1
			pay = payScript
		}
		for _, network := range []byte{common.AddressNetworkTestnet, common.AddressNetworkMainnet} {
			for _, coords := range past {
				pd := data.NewConstr(0,
					data.NewConstr(payTag, data.NewByteString(pay)),
					data.NewConstr(0, data.NewConstr(1,
						data.NewInteger(big.NewInt(coords[0])),
						data.NewInteger(big.NewInt(coords[1])),
						data.NewInteger(big.NewInt(coords[2])),
					)),
				)
				root := data.NewConstr(0, pd)
				var opts []UnmarshalOption
				if network == common.AddressNetworkMainnet {
					opts = []UnmarshalOption{WithAddressNetwork(network)}
				}
				var decoded addrBare
				err := UnmarshalPlutus(root, &decoded, opts...)
				if !errors.Is(err, common.ErrAddressPointerOutOfRange) {
					t.Fatalf("pointer %v network %d script %v: %v", coords, network, script, err)
				}
				failures++
			}
		}
	}
	if failures != 16 {
		t.Fatalf("pointer failures %d, want 16", failures)
	}
}

func TestAddressRejections(t *testing.T) {
	pay := hashPattern(payKeyByte)
	stake := hashPattern(stakeKeyByte)

	if _, err := MarshalPlutus(&addrBare{A: common.Address{}}); err == nil {
		t.Fatal("zero address encoded")
	}

	for _, kind := range []uint8{common.AddressTypeNoneKey, common.AddressTypeNoneScript} {
		if _, err := common.NewAddressFromParts(kind, common.AddressNetworkTestnet, nil, nil); err == nil {
			t.Fatalf("type %d accepted an empty stake hash", kind)
		}
		if _, err := common.NewAddressFromParts(kind, common.AddressNetworkTestnet, nil, []byte{}); err == nil {
			t.Fatalf("type %d accepted a zero-length stake hash", kind)
		}
		for _, tag := range []uint64{0, 1} {
			if kind == common.AddressTypeNoneScript && tag == 0 {
				continue
			}
			if kind == common.AddressTypeNoneKey && tag == 1 {
				continue
			}
			bad := data.NewConstr(tag, data.NewByteString(nil))
			if _, err := unmarshalAddress(bad, common.AddressNetworkTestnet); err == nil {
				t.Fatalf("constructor %d of an empty stake hash decoded", tag)
			}
		}
	}

	byronKey, err := common.NewByronAddressFromParts(common.ByronAddressTypePubkey, pay, common.ByronAddressAttributes{})
	if err != nil {
		t.Fatal(err)
	}
	byronScript, err := common.NewByronAddressFromParts(common.ByronAddressTypeScript, pay, common.ByronAddressAttributes{})
	if err != nil {
		t.Fatal(err)
	}
	byronRedeem, err := common.NewByronAddressRedeem(bytes.Repeat([]byte{0xab}, 32), common.ByronAddressAttributes{})
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range []common.Address{byronKey, byronScript, byronRedeem} {
		if _, err := MarshalPlutus(&addrBare{A: addr}); err == nil {
			t.Fatalf("byron type %d encoded", addr.ByronType())
		}
	}

	future := 0
	for typ := byte(9); typ <= 13; typ++ {
		for _, network := range []byte{common.AddressNetworkTestnet, common.AddressNetworkMainnet} {
			raw := append([]byte{(typ << 4) | network}, pay...)
			if _, err := common.NewAddressFromBytes(raw); err == nil {
				t.Fatalf("header type %d network %d was accepted", typ, network)
			}
			future++
		}
	}
	if future != 10 {
		t.Fatalf("future headers %d, want 10", future)
	}

	hashErrors := 0
	for n := 0; n <= 32; n++ {
		if n == common.AddressHashSize {
			continue
		}
		badPay := data.NewConstr(0,
			data.NewConstr(0, data.NewByteString(bytes.Repeat([]byte{payKeyByte}, n))),
			data.NewConstr(1),
		)
		if _, err := unmarshalAddress(badPay, common.AddressNetworkTestnet); err == nil {
			t.Fatalf("payment hash length %d decoded", n)
		}
		badStake := data.NewConstr(0,
			data.NewConstr(0, data.NewByteString(pay)),
			data.NewConstr(0, data.NewConstr(0, data.NewConstr(0, data.NewByteString(bytes.Repeat([]byte{stakeKeyByte}, n))))),
		)
		if _, err := unmarshalAddress(badStake, common.AddressNetworkTestnet); err == nil {
			t.Fatalf("stake hash length %d decoded", n)
		}
		hashErrors += 2
	}
	if hashErrors != 64 {
		t.Fatalf("hash length errors %d, want 64", hashErrors)
	}

	credShapes := []*data.Constr{
		data.NewConstr(2, data.NewByteString(pay)).(*data.Constr),
		data.NewConstr(0).(*data.Constr),
		data.NewConstr(1).(*data.Constr),
		data.NewConstr(0, data.NewByteString(pay), data.NewByteString(stake)).(*data.Constr),
		data.NewConstr(1, data.NewByteString(pay), data.NewByteString(stake)).(*data.Constr),
	}
	for _, cred := range credShapes {
		pd := data.NewConstr(0, cred, data.NewConstr(1))
		_, err := unmarshalAddress(pd, common.AddressNetworkTestnet)
		if err == nil || !strings.Contains(err.Error(), "constructor") {
			t.Fatalf("credential %#v: %v", cred, err)
		}
	}

	for _, width := range []int{2, 4} {
		fields := make([]data.PlutusData, width)
		for i := range fields {
			fields[i] = data.NewInteger(big.NewInt(1))
		}
		pd := data.NewConstr(0,
			data.NewConstr(0, data.NewByteString(pay)),
			data.NewConstr(0, data.NewConstr(1, fields...)),
		)
		_, err := unmarshalAddress(pd, common.AddressNetworkTestnet)
		if err == nil || !strings.Contains(err.Error(), "constructor") {
			t.Fatalf("StakingPtr width %d: %v", width, err)
		}
	}

	for i := 0; i < 3; i++ {
		nums := []data.PlutusData{
			data.NewInteger(big.NewInt(1)),
			data.NewInteger(big.NewInt(1)),
			data.NewInteger(big.NewInt(1)),
		}
		nums[i] = data.NewInteger(big.NewInt(-1))
		pd := data.NewConstr(0,
			data.NewConstr(0, data.NewByteString(pay)),
			data.NewConstr(0, data.NewConstr(1, nums...)),
		)
		if _, err := unmarshalAddress(pd, common.AddressNetworkTestnet); err == nil {
			t.Fatalf("negative pointer component %d decoded", i)
		}
	}

	maybeShapes := []data.PlutusData{
		data.NewConstr(2, data.NewInteger(big.NewInt(1))),
		data.NewConstr(1, data.NewInteger(big.NewInt(1))),
		data.NewConstr(0),
		data.NewConstr(0, data.NewInteger(big.NewInt(1)), data.NewInteger(big.NewInt(2))),
	}
	for _, shape := range maybeShapes {
		pd := data.NewList(shape)
		var decoded struct {
			_ struct{} `plutusType:"DefList"`
			N *int64   `plutusType:"Maybe"`
		}
		if err := UnmarshalPlutus(pd, &decoded); err == nil || !strings.Contains(err.Error(), "constructor") {
			t.Fatalf("maybe shape %#v: %v", shape, err)
		}
	}

	for _, pd := range []data.PlutusData{
		data.NewConstr(0),
		data.NewConstr(0, data.NewByteString(pay), data.NewConstr(1), data.NewInteger(big.NewInt(1))),
		data.NewConstr(0, data.NewInteger(big.NewInt(1))),
		data.NewConstr(0, nil, data.NewConstr(1)),
		data.NewConstr(0, data.NewInteger(big.NewInt(1)), data.NewConstr(1)),
		data.NewConstr(0, data.NewList(), data.NewConstr(1)),
	} {
		if _, err := unmarshalAddress(pd, common.AddressNetworkTestnet); err == nil {
			t.Fatalf("address shape %#v decoded", pd)
		}
	}

	valid, err := MarshalPlutus(&addrBare{A: mustParts(t, common.AddressTypeKeyNone, common.AddressNetworkTestnet, pay, nil)})
	if err != nil {
		t.Fatal(err)
	}
	networkErrors := 0
	for n := 2; n <= 255; n++ {
		var decoded addrBare
		if err := UnmarshalPlutus(valid, &decoded, WithAddressNetwork(byte(n))); err == nil {
			t.Fatalf("network %d accepted", n)
		}
		networkErrors++
	}
	if networkErrors != 254 {
		t.Fatalf("illegal networks %d, want 254", networkErrors)
	}
}
