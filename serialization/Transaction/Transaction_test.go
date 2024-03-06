package Transaction_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/Transaction"
	"github.com/Salvionied/apollo/serialization/TransactionBody"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionWitnessSet"
	"github.com/Salvionied/cbor/v2"
)

func TestMarshalAndUnmarshal(t *testing.T) {
	tx := Transaction.Transaction{
		TransactionBody: TransactionBody.TransactionBody{
			Inputs: []TransactionInput.TransactionInput{
				{
					TransactionId: []byte{0x01, 0x02, 0x03},
					Index:         0,
				},
			},
		},
		TransactionWitnessSet: TransactionWitnessSet.TransactionWitnessSet{},
		Valid:                 false,
		AuxiliaryData:         nil,
	}

	marshaled, _ := tx.Bytes()
	if hex.EncodeToString(marshaled) != "84a3008182430102030001f60200a1049ffff4f6" {
		t.Error("Invalid marshaling", hex.EncodeToString(marshaled), "Expected", "84a3008182430102030001f60200a1049ffff4f6")
	}
	tx2 := Transaction.Transaction{}
	err := cbor.Unmarshal(marshaled, &tx2)
	if err != nil {
		t.Error("Unmarshal failed", err)
	}
	if tx2.TransactionBody.Inputs[0].Index != 0 {
		t.Error("Invalid unmarshaling", tx2.TransactionBody.Inputs[0].Index, "Expected", 0)
	}
	if tx2.TransactionBody.Inputs[0].TransactionId[0] != 0x01 {
		t.Error("Invalid unmarshaling", tx2.TransactionBody.Inputs[0].TransactionId[0], "Expected", 0x01)
	}
}

func TestId(t *testing.T) {
	tx := Transaction.Transaction{
		TransactionBody: TransactionBody.TransactionBody{
			Inputs: []TransactionInput.TransactionInput{
				{
					TransactionId: []byte{0x01, 0x02, 0x03},
					Index:         0,
				},
			},
		},
		TransactionWitnessSet: TransactionWitnessSet.TransactionWitnessSet{},
		Valid:                 false,
		AuxiliaryData:         nil,
	}
	txId := tx.Id()
	if hex.EncodeToString(txId.Payload) != "49289fa2198208f49f62303aab86d06fb1ff960c812ee98d88c7a5cebb29b615" {
		t.Error("Invalid transaction ID", hex.EncodeToString(txId.Payload), "Expected", "49289fa2198208f49f62303aab86d06fb1ff960c812ee98d88c7a5cebb29b615")
	}
}
