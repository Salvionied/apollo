package apollo

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/blinklabs-io/gouroboros/cbor"
	"github.com/blinklabs-io/gouroboros/ledger/common"
	"github.com/blinklabs-io/gouroboros/ledger/shelley"
)

const (
	bodyInputsKey          = uint64(0)
	bodyCollateralKey      = uint64(13)
	bodyRequiredSignersKey = uint64(14)
	bodyReferenceInputsKey = uint64(18)
)

func transactionBodyFields(
	t *testing.T,
	body any,
) map[uint64]cbor.RawMessage {
	t.Helper()
	encoded, err := cbor.Encode(body)
	if err != nil {
		t.Fatal(err)
	}
	fields := make(map[uint64]cbor.RawMessage)
	if _, err := cbor.Decode(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func TestTransactionBodySetTagPolicyVectors(t *testing.T) {
	spendingHash := common.Blake2b256{0x01}
	collateralHash := common.Blake2b256{0x02}
	referenceHash := common.Blake2b256{0x03}
	signer := common.Blake2b224{0x04}

	inputVector := "81825820" + "01" + strings.Repeat("00", 31) + "00"
	collateralVector := "81825820" + "02" + strings.Repeat("00", 31) + "01"
	referenceVector := "81825820" + "03" + strings.Repeat("00", 31) + "02"
	signerVector := "81581c" + "04" + strings.Repeat("00", 27)
	tagged := func(vector string) string { return "d90102" + vector }

	tests := []struct {
		name      string
		configure func(*Apollo)
		want      map[uint64]string
	}{
		{
			name:      "legacy default",
			configure: func(*Apollo) {},
			want: map[uint64]string{
				bodyInputsKey:          inputVector,
				bodyCollateralKey:      tagged(collateralVector),
				bodyRequiredSignersKey: tagged(signerVector),
				bodyReferenceInputsKey: tagged(referenceVector),
			},
		},
		{
			name: "legacy explicit",
			configure: func(a *Apollo) {
				a.SetTransactionBodySetTagPolicy(
					TransactionBodySetTagPolicyLegacy,
				)
			},
			want: map[uint64]string{
				bodyInputsKey:          inputVector,
				bodyCollateralKey:      tagged(collateralVector),
				bodyRequiredSignersKey: tagged(signerVector),
				bodyReferenceInputsKey: tagged(referenceVector),
			},
		},
		{
			name: "uniform untagged",
			configure: func(a *Apollo) {
				a.SetTransactionBodySetTagPolicy(
					TransactionBodySetTagPolicyUntagged,
				)
			},
			want: map[uint64]string{
				bodyInputsKey:          inputVector,
				bodyCollateralKey:      collateralVector,
				bodyRequiredSignersKey: signerVector,
				bodyReferenceInputsKey: referenceVector,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := New(setupFixedContext())
			test.configure(a)
			a.requiredSigners = []common.Blake2b224{signer}
			a.referenceInputs = []shelley.ShelleyTransactionInput{{
				TxId:        referenceHash,
				OutputIndex: 2,
			}}
			a.collaterals = []common.Utxo{
				makeTestUtxo(t, collateralHash, 1, 5_000_000),
			}

			body, err := a.buildBody(
				[]common.Utxo{
					makeTestUtxo(t, spendingHash, 0, 10_000_000),
				},
				nil,
				1,
			)
			if err != nil {
				t.Fatal(err)
			}
			fields := transactionBodyFields(t, &body)
			for key, want := range test.want {
				raw, ok := fields[key]
				if !ok {
					t.Fatalf("body field %d is absent", key)
				}
				if got := hex.EncodeToString(raw); got != want {
					t.Errorf("body field %d CBOR = %s, want %s", key, got, want)
				}
			}
		})
	}
}

func TestTransactionBodySetTagPolicySurvivesCompleteAndSigning(t *testing.T) {
	wallet, err := NewBursaWallet(signingTestMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	cc := networkTestContext(t, common.AddressNetworkMainnet)
	addTestUtxo(cc, wallet.Address(), 10_000_000, 0x01, 0)
	addTestUtxo(cc, wallet.Address(), 5_000_000, 0x02, 1)
	addTestUtxo(cc, wallet.Address(), 5_000_000, 0x03, 2)
	utxos, err := cc.Utxos(wallet.Address())
	if err != nil {
		t.Fatal(err)
	}
	if len(utxos) != 3 {
		t.Fatalf("fixture has %d UTxOs, want 3", len(utxos))
	}

	payee := syntheticAddress(
		t,
		common.AddressTypeKeyKey,
		common.AddressNetworkMainnet,
	)
	a := New(cc).
		SetTransactionBodySetTagPolicy(
			TransactionBodySetTagPolicyUntagged,
		).
		SetWallet(wallet).
		AddLoadedUTxOs(utxos[0]).
		AddCollateral(utxos[1]).
		AddRequiredSigner(wallet.PubKeyHash()).
		SetTtl(50_000_000).
		PayToAddress(payee, 2_000_000)
	a, err = a.AddReferenceInput(hex.EncodeToString(utxos[2].Id.Id().Bytes()), 2)
	if err != nil {
		t.Fatal(err)
	}
	a, err = a.Complete()
	if err != nil {
		t.Fatal(err)
	}

	unsignedCBOR, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	unsignedElements := txTopLevelElements(t, unsignedCBOR)
	fields := make(map[uint64]cbor.RawMessage)
	if _, err := cbor.Decode(unsignedElements[0], &fields); err != nil {
		t.Fatal(err)
	}
	setTag := []byte{0xd9, 0x01, 0x02}
	for _, key := range []uint64{
		bodyInputsKey,
		bodyCollateralKey,
		bodyRequiredSignersKey,
		bodyReferenceInputsKey,
	} {
		raw, ok := fields[key]
		if !ok {
			t.Fatalf("completed body field %d is absent", key)
		}
		if bytes.HasPrefix(raw, setTag) {
			t.Errorf("completed body field %d unexpectedly uses CBOR tag 258", key)
		}
	}

	a, err = a.Sign()
	if err != nil {
		t.Fatal(err)
	}
	signedCBOR, err := a.GetTxCbor()
	if err != nil {
		t.Fatal(err)
	}
	signedElements := txTopLevelElements(t, signedCBOR)
	if !bytes.Equal(unsignedElements[0], signedElements[0]) {
		t.Fatal("Sign changed the exact transaction-body bytes")
	}
	assertSignsSubmittedBodyHash(t, signedCBOR, wallet.PubKeyHash())

	witnessFields := make(map[uint64]cbor.RawMessage)
	if _, err := cbor.Decode(signedElements[1], &witnessFields); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(witnessFields[0], setTag) {
		t.Fatal("transaction-body policy changed vkey witness-set tagging")
	}

	datumBuilder := New(cc).SetTransactionBodySetTagPolicy(
		TransactionBodySetTagPolicyUntagged,
	)
	datumBuilder.datums = []common.Datum{{}}
	witnessSet := datumBuilder.buildWitnessSet(nil)
	datumCBOR, err := cbor.Encode(&witnessSet.WsPlutusData)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(datumCBOR, setTag) {
		t.Fatal("transaction-body policy changed Plutus-data set tagging")
	}
}

func TestTransactionBodySetTagPolicyCloneAndValidation(t *testing.T) {
	a := New(setupFixedContext()).SetTransactionBodySetTagPolicy(
		TransactionBodySetTagPolicyUntagged,
	)
	if got := a.Clone().bodySetTagPolicy; got != TransactionBodySetTagPolicyUntagged {
		t.Fatalf("cloned policy = %d, want untagged", got)
	}

	invalid := New(setupFixedContext()).SetTransactionBodySetTagPolicy(
		TransactionBodySetTagPolicy(255),
	)
	_, err := invalid.Complete()
	if err == nil || !strings.Contains(err.Error(), "unsupported policy 255") {
		t.Fatalf("invalid policy error = %v", err)
	}
}
