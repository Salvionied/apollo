package TransactionInput_test

import (
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/blinklabs-io/gouroboros/cbor"
)

var SAMPLE_TX_IN = TransactionInput.TransactionInput{
	TransactionId: []byte{0x01, 0x02, 0x03},
	Index:         0,
}

func TestMarshalAndUnmarshal(t *testing.T) {
	txIn := SAMPLE_TX_IN
	marshaled, _ := cbor.Encode(txIn)
	txIn2 := TransactionInput.TransactionInput{}
	_, err := cbor.Decode(marshaled, &txIn2)
	if err != nil {
		t.Error("Unmarshal failed", err)
	}
	if txIn2.Index != 0 {
		t.Error("Invalid unmarshaling", txIn2.Index, "Expected", 0)
	}
	if len(txIn2.TransactionId) == 0 {
		t.Error("TransactionId is empty")
	} else if txIn2.TransactionId[0] != 0x01 {
		t.Error(
			"Invalid unmarshaling",
			txIn2.TransactionId[0],
			"Expected",
			0x01,
		)
	}
}

func TestClone(t *testing.T) {
	txIn := SAMPLE_TX_IN
	txInClone := txIn.Clone()
	if txInClone.Index != txIn.Index || &txInClone == &txIn {
		t.Error("Clone failed")
	}
}

func TestEqualTo(t *testing.T) {
	txIn := SAMPLE_TX_IN
	txInClone := txIn.Clone()
	if !txIn.EqualTo(txInClone) {
		t.Error("EqualTo failed")
	}
}

func TestLessThan(t *testing.T) {
	txIn := SAMPLE_TX_IN
	txInClone := txIn.Clone()
	if txIn.LessThan(txInClone) {
		t.Error("LessThan failed")
	}
	txInClone.Index = 1
	if !txIn.LessThan(txInClone) {
		t.Error("LessThan failed")
	}
}

func TestString(t *testing.T) {
	txIn := SAMPLE_TX_IN
	if txIn.String() != "010203.0" {
		t.Error(txIn.String(), "Expected", "010203.0")
	}
}
