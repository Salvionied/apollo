package TransactionBody_test

import (
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Value"
	"github.com/fxamacker/cbor/v2"
)

var SAMPLE_ADDRESS, _ = Address.DecodeAddress(
	"addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh",
)

var SAMPLE_TX_OUT_1 = TransactionOutput.SimpleTransactionOutput(
	SAMPLE_ADDRESS,
	Value.PureLovelaceValue(1000000),
)

var SAMPLE_TX_IN = TransactionInput.TransactionInput{
	TransactionId: []byte{0x01, 0x02, 0x03},
	Index:         0,
}

func TestTransactionBodyMarshalAndUnmarshal(t *testing.T) {
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		},
		Outputs: []TransactionOutput.TransactionOutput{
			SAMPLE_TX_OUT_1,
		},
		Fee: 1000000,
		Ttl: 1000000,
	}

	marshaled, _ := cbor.Marshal(txBody)
	if hex.EncodeToString(
		marshaled,
	) != "a40081824301020300018182583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a000f4240021a000f4240031a000f4240" {
		t.Error(
			"Invalid marshaling",
			hex.EncodeToString(marshaled),
			"Expected",
			"a40081824301020300018182583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a000f4240021a000f4240031a000f4240",
		)
	}
	txBody2 := TransactionBody.TransactionBody{}
	err := cbor.Unmarshal(marshaled, &txBody2)
	if err != nil {
		t.Error("Unmarshal failed", err)
	}
	if txBody2.Inputs[0].Index != 0 {
		t.Error("Invalid unmarshaling", txBody2.Inputs[0].Index, "Expected", 0)
	}
	if txBody2.Inputs[0].TransactionId[0] != 0x01 {
		t.Error(
			"Invalid unmarshaling",
			txBody2.Inputs[0].TransactionId[0],
			"Expected",
			0x01,
		)
	}
	if txBody2.Outputs[0].GetAddress().String() != SAMPLE_ADDRESS.String() {
		t.Error(
			"Invalid unmarshaling",
			txBody2.Outputs[0].GetAddress().String(),
			"Expected",
			SAMPLE_ADDRESS,
		)
	}
	if txBody2.Outputs[0].GetAmount().
		GetCoin() !=
		SAMPLE_TX_OUT_1.GetAmount().
			GetCoin() {
		t.Error(
			"Invalid unmarshaling",
			txBody2.Outputs[0].GetAmount().GetCoin(),
			"Expected",
			SAMPLE_TX_OUT_1.GetAmount().GetCoin(),
		)
	}
}

func TestEmptyMintOmitted(t *testing.T) {
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		},
		Outputs: []TransactionOutput.TransactionOutput{
			SAMPLE_TX_OUT_1,
		},
		Fee:  1000000,
		Mint: make(MultiAsset.MultiAsset[int64]),
	}
	marshaled, err := cbor.Marshal(txBody)
	if err != nil {
		t.Fatal("Marshal failed:", err)
	}
	var rawMap map[int]cbor.RawMessage
	if err := cbor.Unmarshal(marshaled, &rawMap); err != nil {
		t.Fatal("Unmarshal to raw map failed:", err)
	}
	if _, ok := rawMap[9]; ok {
		t.Error(
			"Empty Mint (key 9) should be omitted",
			hex.EncodeToString(marshaled),
		)
	}
}

func TestNilMintOmitted(t *testing.T) {
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		},
		Outputs: []TransactionOutput.TransactionOutput{
			SAMPLE_TX_OUT_1,
		},
		Fee: 1000000,
	}
	marshaled, err := cbor.Marshal(txBody)
	if err != nil {
		t.Fatal("Marshal failed:", err)
	}
	var rawMap map[int]cbor.RawMessage
	if err := cbor.Unmarshal(marshaled, &rawMap); err != nil {
		t.Fatal("Unmarshal to raw map failed:", err)
	}
	if _, ok := rawMap[9]; ok {
		t.Error(
			"Nil Mint (key 9) should be omitted",
			hex.EncodeToString(marshaled),
		)
	}
}

func TestNonEmptyMintIncluded(t *testing.T) {
	policyId := "a0a0a0a0a0a0a0a0a0a0a0a0a0" +
		"a0a0a0a0a0a0a0a0a0a0a0a0a0a0a0"
	mint := MultiAsset.MultiAsset[int64]{
		Policy.PolicyId{Value: policyId}: Asset.Asset[int64]{
			AssetName.NewAssetNameFromString("token"): 100,
		},
	}
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		},
		Outputs: []TransactionOutput.TransactionOutput{
			SAMPLE_TX_OUT_1,
		},
		Fee:  1000000,
		Mint: mint,
	}
	marshaled, err := cbor.Marshal(txBody)
	if err != nil {
		t.Fatal("Marshal failed:", err)
	}
	var rawMap map[int]cbor.RawMessage
	if err := cbor.Unmarshal(marshaled, &rawMap); err != nil {
		t.Fatal("Unmarshal to raw map failed:", err)
	}
	if _, ok := rawMap[9]; !ok {
		t.Error(
			"Non-empty Mint (key 9) should be present",
			hex.EncodeToString(marshaled),
		)
	}
}

func TestTransactionBodyHash(t *testing.T) {
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		}}
	hash, _ := txBody.Hash()
	if hex.EncodeToString(
		hash,
	) != "49289fa2198208f49f62303aab86d06fb1ff960c812ee98d88c7a5cebb29b615" {
		t.Error(
			"Invalid hash",
			hex.EncodeToString(hash),
			"Expected",
			"49289fa2198208f49f62303aab86d06fb1ff960c812ee98d88c7a5cebb29b615",
		)
	}
}

func TestId(t *testing.T) {
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		}}
	txId, _ := txBody.Id()
	if hex.EncodeToString(
		txId.Payload,
	) != "49289fa2198208f49f62303aab86d06fb1ff960c812ee98d88c7a5cebb29b615" {
		t.Error(
			"Invalid Id",
			hex.EncodeToString(txId.Payload),
			"Expected",
			"49289fa2198208f49f62303aab86d06fb1ff960c812ee98d88c7a5cebb29b615",
		)
	}
}

func TestTreasuryFieldsCBORRoundTrip(t *testing.T) {
	original := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		},
		Outputs: []TransactionOutput.TransactionOutput{
			SAMPLE_TX_OUT_1,
		},
		Fee:                  200_000,
		CurrentTreasuryValue: 1_000_000_000,
		Donation:             5_000_000,
	}

	data, err := cbor.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded TransactionBody.TransactionBody
	if err := cbor.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.CurrentTreasuryValue != 1_000_000_000 {
		t.Fatalf(
			"CurrentTreasuryValue: expected 1000000000, got %d",
			decoded.CurrentTreasuryValue,
		)
	}
	if decoded.Donation != 5_000_000 {
		t.Fatalf(
			"Donation: expected 5000000, got %d",
			decoded.Donation,
		)
	}
	if decoded.Fee != 200_000 {
		t.Fatalf(
			"Fee: expected 200000, got %d",
			decoded.Fee,
		)
	}
}

func TestTreasuryFieldsRejectOverflow(t *testing.T) {
	tests := []struct {
		name      string
		fieldKey  int
		fieldName string
	}{
		{
			name:      "current_treasury_value",
			fieldKey:  21,
			fieldName: "current treasury value",
		},
		{
			name:      "donation",
			fieldKey:  22,
			fieldName: "donation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := cbor.Marshal(map[int]any{
				0: []TransactionInput.TransactionInput{
					SAMPLE_TX_IN,
				},
				1: []TransactionOutput.TransactionOutput{
					SAMPLE_TX_OUT_1,
				},
				2:           int64(200_000),
				tc.fieldKey: uint64(math.MaxInt64) + 1,
			})
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var decoded TransactionBody.TransactionBody
			err = cbor.Unmarshal(data, &decoded)
			if err == nil {
				t.Fatal("expected overflow error")
			}
			if !strings.Contains(err.Error(), tc.fieldName) {
				t.Fatalf(
					"expected error to mention %q, got %v",
					tc.fieldName,
					err,
				)
			}
		})
	}
}
