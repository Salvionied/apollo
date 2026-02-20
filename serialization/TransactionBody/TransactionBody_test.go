package TransactionBody_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Address"
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
	) != "a50081824301020300018182583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a000f4240021a000f4240031a000f424009a0" {
		t.Error(
			"Invalid marshaling",
			hex.EncodeToString(marshaled),
			"Expected",
			"a50081824301020300018182583901bb2ff620c0dd8b0adc19e6ffadea1a150c85d1b22d05e2db10c55c613b8c8a100c16cf62b9c2bacc40453aaa67ced633993f2b4eec5b88e41a000f4240021a000f4240031a000f424009a0",
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

func TestTransactionBodyHash(t *testing.T) {
	txBody := TransactionBody.TransactionBody{
		Inputs: []TransactionInput.TransactionInput{
			SAMPLE_TX_IN,
		}}
	hash, _ := txBody.Hash()
	if hex.EncodeToString(
		hash,
	) != "2d1312c2950d08c5fe35b8d1f293d13e0cf85e51a1c1779ee05b89838cf4e771" {
		t.Error(
			"Invalid hash",
			hex.EncodeToString(hash),
			"Expected",
			"2d1312c2950d08c5fe35b8d1f293d13e0cf85e51a1c1779ee05b89838cf4e771",
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
	) != "2d1312c2950d08c5fe35b8d1f293d13e0cf85e51a1c1779ee05b89838cf4e771" {
		t.Error(
			"Invalid Id",
			hex.EncodeToString(txId.Payload),
			"Expected",
			"2d1312c2950d08c5fe35b8d1f293d13e0cf85e51a1c1779ee05b89838cf4e771",
		)
	}
}
