package plutusencoder

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
	"github.com/blinklabs-io/plutigo/data"
)

type productCodecsDatum struct {
	_           struct{}          `plutusType:"DefList" plutusConstr:"0"`
	Owner       common.Credential `plutusType:"Credential"`
	Utxo        TxOutRef          `plutusType:"TxOutRef"`
	RewardAsset AssetClass        `plutusType:"AssetClass"`
}

func TestProductCodecsRejectTypedNilChildren(t *testing.T) {
	hash := bytes.Repeat([]byte{0xab}, common.Blake2b224Size)
	txID := bytes.Repeat([]byte{0xcd}, common.Blake2b256Size)
	cred := data.NewConstr(0, data.NewByteString(hash))
	ref := data.NewConstr(0, data.NewByteString(txID), data.NewInteger(big.NewInt(1)))
	asset := data.NewConstr(0, data.NewByteString(nil), data.NewByteString(nil))
	cases := []data.PlutusData{
		data.NewConstr(0, data.NewConstr(0, (*data.ByteString)(nil)), ref, asset),
		data.NewConstr(0, cred, data.NewConstr(0, (*data.ByteString)(nil), data.NewInteger(big.NewInt(1))), asset),
		data.NewConstr(0, cred, data.NewConstr(0, data.NewByteString(txID), (*data.Integer)(nil)), asset),
		data.NewConstr(0, cred, ref, data.NewConstr(0, (*data.ByteString)(nil), data.NewByteString(nil))),
		data.NewConstr(0, cred, ref, data.NewConstr(0, data.NewByteString(nil), (*data.ByteString)(nil))),
	}
	for i, pd := range cases {
		var decoded productCodecsDatum
		if err := UnmarshalPlutus(pd, &decoded); err == nil {
			t.Fatalf("case %d decoded a typed-nil child", i)
		}
	}
}

func TestCredentialMatchesGouroboros(t *testing.T) {
	hash, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0xab}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	txId := bytes.Repeat([]byte{0xcd}, common.Blake2b256Size)

	for _, kind := range []uint{common.CredentialTypeAddrKeyHash, common.CredentialTypeScriptHash} {
		cred := common.Credential{CredType: kind, Credential: hash}
		datum := productCodecsDatum{
			Owner: cred,
			Utxo:  TxOutRef{TxId: txId, Index: 3},
		}
		pd, err := MarshalPlutus(&datum)
		if err != nil {
			t.Fatalf("kind %d: %v", kind, err)
		}
		outer := pd.(*data.Constr)
		got, err := data.Encode(outer.Fields[0])
		if err != nil {
			t.Fatal(err)
		}
		want, err := data.Encode(cred.ToPlutusData())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("kind %d: credential bytes do not match Credential.ToPlutusData", kind)
		}

		var decoded productCodecsDatum
		if err := UnmarshalPlutus(pd, &decoded); err != nil {
			t.Fatalf("kind %d unmarshal: %v", kind, err)
		}
		if decoded.Owner.CredType != cred.CredType || decoded.Owner.Credential != cred.Credential {
			t.Fatalf("kind %d decoded as %+v", kind, decoded.Owner)
		}
	}
}

func TestTxOutRefMatchesShelleyInputShape(t *testing.T) {
	hash, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x11}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	txId := bytes.Repeat([]byte{0x22}, common.Blake2b256Size)
	datum := productCodecsDatum{
		Owner: common.Credential{CredType: common.CredentialTypeAddrKeyHash, Credential: hash},
		Utxo:  TxOutRef{TxId: txId, Index: 7},
		RewardAsset: AssetClass{
			PolicyId:  bytes.Repeat([]byte{0x33}, common.Blake2b224Size),
			TokenName: []byte("TOKEN"),
		},
	}

	pd, err := MarshalPlutus(&datum)
	if err != nil {
		t.Fatal(err)
	}
	outer := pd.(*data.Constr)
	gotRef, err := data.Encode(outer.Fields[1])
	if err != nil {
		t.Fatal(err)
	}
	txHash, err := common.NewBlake2b256Checked(txId)
	if err != nil {
		t.Fatal(err)
	}
	wantRef, err := data.Encode(shelley.ShelleyTransactionInput{
		TxId:        txHash,
		OutputIndex: 7,
	}.ToPlutusData())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotRef, wantRef) {
		t.Fatal("TxOutRef encoding does not match ShelleyTransactionInput.ToPlutusData")
	}

	var decoded productCodecsDatum
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.Utxo.TxId, txId) || decoded.Utxo.Index != 7 {
		t.Fatalf("unexpected TxOutRef: %+v", decoded.Utxo)
	}
	if !bytes.Equal(decoded.RewardAsset.PolicyId, datum.RewardAsset.PolicyId) || !bytes.Equal(decoded.RewardAsset.TokenName, []byte("TOKEN")) {
		t.Fatalf("unexpected AssetClass: %+v", decoded.RewardAsset)
	}
}

func TestProductCodecsRejectWrongHashLengths(t *testing.T) {
	hash, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x11}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	validCred := common.Credential{CredType: common.CredentialTypeAddrKeyHash, Credential: hash}
	validTxId := bytes.Repeat([]byte{0x22}, common.Blake2b256Size)

	shortTxId := productCodecsDatum{
		Owner: validCred,
		Utxo:  TxOutRef{TxId: []byte{0x01, 0x02}, Index: 0},
	}
	if _, err := MarshalPlutus(&shortTxId); err == nil {
		t.Fatal("expected short TxOutRef txId to fail")
	}

	shortPolicy := productCodecsDatum{
		Owner:       validCred,
		Utxo:        TxOutRef{TxId: validTxId, Index: 0},
		RewardAsset: AssetClass{PolicyId: []byte{0x01}, TokenName: []byte("T")},
	}
	if _, err := MarshalPlutus(&shortPolicy); err == nil {
		t.Fatal("expected 1-byte policy id to fail")
	}

	badCredWire := data.NewConstr(0,
		data.NewConstr(0, data.NewByteString([]byte{0x01})),
		data.NewConstr(0, data.NewByteString(validTxId), data.NewInteger(big.NewInt(0))),
		data.NewConstr(0, data.NewByteString(nil), data.NewByteString(nil)),
	)
	var decoded productCodecsDatum
	if err := UnmarshalPlutus(badCredWire, &decoded); err == nil {
		t.Fatal("expected 1-byte credential hash to fail on unmarshal")
	}
}

func TestZeroCredentialRoundTrip(t *testing.T) {
	txId := bytes.Repeat([]byte{0xcd}, common.Blake2b256Size)
	for _, kind := range []uint{common.CredentialTypeAddrKeyHash, common.CredentialTypeScriptHash} {
		cred := common.Credential{CredType: kind}
		pd, err := marshalCredential(cred)
		if err != nil {
			t.Fatalf("kind %d: %v", kind, err)
		}
		got, err := data.Encode(pd)
		if err != nil {
			t.Fatal(err)
		}
		want, err := data.Encode(cred.ToPlutusData())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("kind %d encoding does not match Credential.ToPlutusData", kind)
		}
		decoded, err := unmarshalCredential(pd)
		if err != nil {
			t.Fatalf("kind %d decode: %v", kind, err)
		}
		if decoded.CredType != cred.CredType || decoded.Credential != cred.Credential {
			t.Fatalf("kind %d decoded as %+v", kind, decoded)
		}

		datum := productCodecsDatum{
			Owner: cred,
			Utxo:  TxOutRef{TxId: txId},
		}
		wire, err := MarshalPlutus(&datum)
		if err != nil {
			t.Fatalf("kind %d datum: %v", kind, err)
		}
		var back productCodecsDatum
		if err := UnmarshalPlutus(wire, &back); err != nil {
			t.Fatalf("kind %d datum decode: %v", kind, err)
		}
		if back.Owner.CredType != cred.CredType || back.Owner.Credential != cred.Credential {
			t.Fatalf("kind %d datum owner = %+v", kind, back.Owner)
		}
	}
}

func TestCredentialHashLengths(t *testing.T) {
	const (
		minLen   = 0
		maxLen   = 32
		wantLens = maxLen - minLen + 1
	)
	runs := 0
	roundTrips := 0
	encodeRejects := 0
	decodeRejects := 0
	for n := minLen; n <= maxLen; n++ {
		runs++
		raw := bytes.Repeat([]byte{0xab}, n)
		if n == common.Blake2b224Size {
			for _, kind := range []struct {
				name     string
				credType uint
			}{
				{name: "key", credType: common.CredentialTypeAddrKeyHash},
				{name: "script", credType: common.CredentialTypeScriptHash},
			} {
				roundTrips++
				hash, err := common.NewBlake2b224Checked(raw)
				if err != nil {
					t.Fatalf("%s hash: %v", kind.name, err)
				}
				cred := common.Credential{CredType: kind.credType, Credential: hash}
				pd, err := marshalCredential(cred)
				if err != nil {
					t.Fatalf("%s encode: %v", kind.name, err)
				}
				got, err := data.Encode(pd)
				if err != nil {
					t.Fatal(err)
				}
				want, err := data.Encode(cred.ToPlutusData())
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s encoding does not match Credential.ToPlutusData", kind.name)
				}
				decoded, err := unmarshalCredential(pd)
				if err != nil {
					t.Fatalf("%s decode: %v", kind.name, err)
				}
				if decoded.CredType != cred.CredType || decoded.Credential != cred.Credential {
					t.Fatalf("%s decoded as %+v", kind.name, decoded)
				}
			}
			continue
		}

		// common.Credential stores a fixed 28-byte hash, so a different
		// length cannot be built. That rejection is the encode error.
		if _, err := common.NewBlake2b224Checked(raw); err == nil {
			t.Fatalf("length %d built a credential hash", n)
		}
		encodeRejects++

		for _, tag := range []uint64{0, 1} {
			pd := data.NewConstr(tag, data.NewByteString(raw))
			if _, err := unmarshalCredential(pd); err == nil {
				t.Fatalf("length %d constructor %d decoded", n, tag)
			}
			decodeRejects++
		}
	}
	if runs != wantLens {
		t.Fatalf("hash lengths ran %d, want %d", runs, wantLens)
	}
	if roundTrips != 2 {
		t.Fatalf("length %d round trips = %d, want 2", common.Blake2b224Size, roundTrips)
	}
	if encodeRejects != wantLens-1 {
		t.Fatalf("encode rejections = %d, want %d", encodeRejects, wantLens-1)
	}
	if decodeRejects != (wantLens-1)*2 {
		t.Fatalf("decode rejections = %d, want %d", decodeRejects, (wantLens-1)*2)
	}
}

func TestProductConstructorShapeErrors(t *testing.T) {
	hash := bytes.Repeat([]byte{0xab}, common.Blake2b224Size)
	shapes := []struct {
		tag    uint64
		fields int
	}{
		{tag: 0, fields: 0},
		{tag: 0, fields: 2},
		{tag: 2, fields: 1},
	}
	runs := 0
	for _, shape := range shapes {
		fields := make([]data.PlutusData, shape.fields)
		for i := range fields {
			fields[i] = data.NewByteString(hash)
		}
		pd := data.NewConstr(shape.tag, fields...)
		runs++

		_, credErr := unmarshalCredential(pd)
		assertConstrWidthError(t, credErr, shape.tag, shape.fields)

		_, txErr := unmarshalTxOutRef(pd)
		_, assetErr := unmarshalAssetClass(pd)
		// Constructor 0 with two fields is the TxOutRef and AssetClass shape.
		// Two 28-byte strings are a valid asset and a short transaction id.
		if shape.tag == 0 && shape.fields == 2 {
			if txErr == nil {
				t.Fatal("expected TxOutRef with a 28-byte transaction id to fail")
			}
			if assetErr != nil {
				t.Fatalf("28-byte policy and token name: %v", assetErr)
			}
			continue
		}
		assertConstrWidthError(t, txErr, shape.tag, shape.fields)
		assertConstrWidthError(t, assetErr, shape.tag, shape.fields)
	}
	if runs != 3 {
		t.Fatalf("constructor shapes ran %d, want 3", runs)
	}

	notConstr := data.NewInteger(big.NewInt(1))
	_, credTypeErr := unmarshalCredential(notConstr)
	_, txTypeErr := unmarshalTxOutRef(notConstr)
	_, assetTypeErr := unmarshalAssetClass(notConstr)
	typeChecks := 0
	for _, err := range []error{credTypeErr, txTypeErr, assetTypeErr} {
		typeChecks++
		if err == nil {
			t.Fatal("expected a non-constructor to fail")
		}
		if !strings.Contains(err.Error(), "*data.Integer") {
			t.Fatalf("non-constructor error %q does not keep the value type", err)
		}
		if strings.Contains(err.Error(), "got constructor") {
			t.Fatalf("non-constructor error names a constructor: %v", err)
		}
	}
	if typeChecks != 3 {
		t.Fatalf("non-constructor cases = %d, want 3", typeChecks)
	}
}

func TestTxOutRefBounds(t *testing.T) {
	lengths := []int{common.Blake2b256Size - 1, common.Blake2b256Size + 1}
	lengthRuns := 0
	for _, n := range lengths {
		lengthRuns++
		txID := bytes.Repeat([]byte{0xcd}, n)
		if _, err := marshalTxOutRef(TxOutRef{TxId: txID, Index: 0}); err == nil {
			t.Fatalf("%d-byte transaction id encoded", n)
		}
		pd := data.NewConstr(0, data.NewByteString(txID), data.NewInteger(big.NewInt(0)))
		if _, err := unmarshalTxOutRef(pd); err == nil {
			t.Fatalf("%d-byte transaction id decoded", n)
		}
	}
	if lengthRuns != 2 {
		t.Fatalf("transaction id lengths ran %d, want 2", lengthRuns)
	}

	txID := bytes.Repeat([]byte{0xcd}, common.Blake2b256Size)
	indexes := []uint32{0, math.MaxUint32}
	indexRuns := 0
	for _, idx := range indexes {
		indexRuns++
		pd, err := marshalTxOutRef(TxOutRef{TxId: txID, Index: idx})
		if err != nil {
			t.Fatalf("index %d encode: %v", idx, err)
		}
		got, err := unmarshalTxOutRef(pd)
		if err != nil {
			t.Fatalf("index %d decode: %v", idx, err)
		}
		if !bytes.Equal(got.TxId, txID) || got.Index != idx {
			t.Fatalf("index %d decoded as %+v", idx, got)
		}
	}
	if indexRuns != 2 {
		t.Fatalf("output indexes ran %d, want 2", indexRuns)
	}

	overflow := data.NewConstr(0,
		data.NewByteString(txID),
		data.NewInteger(new(big.Int).SetUint64(uint64(math.MaxUint32)+1)),
	)
	if _, err := unmarshalTxOutRef(overflow); err == nil {
		t.Fatal("expected an output index above uint32 to fail")
	}
}

func TestAssetClassTokenNameLengths(t *testing.T) {
	if maxTokenNameSize != 32 {
		t.Fatalf("maxTokenNameSize = %d, want 32", maxTokenNameSize)
	}
	cases := []struct {
		policy int
		name   int
		ok     bool
	}{
		{policy: common.Blake2b224Size, name: 0, ok: true},
		{policy: common.Blake2b224Size, name: 32, ok: true},
		{policy: 0, name: 0, ok: true},
		{policy: 0, name: 1, ok: false},
		{policy: common.Blake2b224Size, name: 33, ok: false},
	}
	runs := 0
	for _, tc := range cases {
		runs++
		asset := AssetClass{
			PolicyId:  bytes.Repeat([]byte{0x33}, tc.policy),
			TokenName: bytes.Repeat([]byte{0x7a}, tc.name),
		}
		pd, err := marshalAssetClass(asset)
		if tc.ok {
			if err != nil {
				t.Fatalf("policy %d name %d encode: %v", tc.policy, tc.name, err)
			}
			got, err := unmarshalAssetClass(pd)
			if err != nil {
				t.Fatalf("policy %d name %d decode: %v", tc.policy, tc.name, err)
			}
			if !bytes.Equal(got.PolicyId, asset.PolicyId) || !bytes.Equal(got.TokenName, asset.TokenName) {
				t.Fatalf("policy %d name %d decoded as %+v", tc.policy, tc.name, got)
			}
			continue
		}
		if err == nil {
			t.Fatalf("policy %d name %d encoded", tc.policy, tc.name)
		}
		wire := data.NewConstr(0, data.NewByteString(asset.PolicyId), data.NewByteString(asset.TokenName))
		if _, err := unmarshalAssetClass(wire); err == nil {
			t.Fatalf("policy %d name %d decoded", tc.policy, tc.name)
		}
	}
	if runs != 5 {
		t.Fatalf("asset class cases = %d, want 5", runs)
	}
}

func assertConstrWidthError(t *testing.T, err error, tag uint64, fields int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected constructor %d with %d fields to fail", tag, fields)
	}
	want := fmt.Sprintf("constructor %d with %d fields", tag, fields)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not include %q", err, want)
	}
}

func TestTxOutRefRejectsNegativeIndex(t *testing.T) {
	txID := bytes.Repeat([]byte{0xcd}, common.Blake2b256Size)
	pd := data.NewConstr(0, data.NewByteString(txID), data.NewInteger(big.NewInt(-1)))
	if _, err := unmarshalTxOutRef(pd); err == nil {
		t.Fatal("expected a negative output index to fail")
	}
}

func TestCredentialRejectsUnsupportedType(t *testing.T) {
	hash, err := common.NewBlake2b224Checked(bytes.Repeat([]byte{0x11}, common.Blake2b224Size))
	if err != nil {
		t.Fatal(err)
	}
	cred := common.Credential{CredType: 99, Credential: hash}
	if _, err := marshalCredential(cred); err == nil {
		t.Fatal("expected an unsupported credential type to fail")
	}
}

func TestProductCodecsRejectWrongGoType(t *testing.T) {
	type wrong struct {
		_ struct{} `plutusType:"DefList" plutusConstr:"0"`
		C int64    `plutusType:"Credential"`
		R int64    `plutusType:"TxOutRef"`
		A int64    `plutusType:"AssetClass"`
	}
	_, err := MarshalPlutus(&wrong{})
	if err == nil {
		t.Fatal("expected a wrong Go field type to fail")
	}
	if strings.Count(err.Error(), "field C:") != 1 {
		t.Fatalf("field name repeated: %v", err)
	}
	if err := UnmarshalPlutus(data.NewConstr(0, data.NewInteger(big.NewInt(1)), data.NewInteger(big.NewInt(1)), data.NewInteger(big.NewInt(1))), &wrong{}); err == nil {
		t.Fatal("expected decode of a wrong Go field type to fail")
	}
}
