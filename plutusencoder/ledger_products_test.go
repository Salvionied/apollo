package plutusencoder

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/plutigo/data"
)

type productCodecsDatum struct {
	_           struct{}          `plutusType:"DefList" plutusConstr:"0"`
	Owner       common.Credential `plutusType:"Credential"`
	Utxo        TxOutRef          `plutusType:"TxOutRef"`
	RewardAsset AssetClass        `plutusType:"AssetClass"`
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
	wantRef, err := data.Encode(data.NewConstr(0, data.NewByteString(txId), data.NewInteger(big.NewInt(7))))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotRef, wantRef) {
		t.Fatal("TxOutRef encoding does not match constructor 0 of [txId, index]")
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
