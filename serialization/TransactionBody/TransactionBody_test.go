package TransactionBody_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/Address"
	"github.com/Salvionied/apollo/v2/serialization/TransactionBody"
	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/Salvionied/apollo/v2/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/v2/serialization/Value"
	"github.com/blinklabs-io/gouroboros/cbor"
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

	marshaled, _ := cbor.Encode(txBody)
	txBody2 := TransactionBody.TransactionBody{}
	_, err := cbor.Decode(marshaled, &txBody2)
	if err != nil {
		t.Error("Unmarshal failed", err)
	}
	if len(txBody2.Inputs) == 0 {
		t.Error("No inputs")
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
