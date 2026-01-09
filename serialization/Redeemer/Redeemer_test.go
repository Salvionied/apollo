package Redeemer_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/Salvionied/apollo/v2/serialization/Redeemer"
	"github.com/blinklabs-io/gouroboros/cbor"
)

func TestExecutionUnitsFunctions(t *testing.T) {
	ex1 := Redeemer.ExecutionUnits{Mem: 1, Steps: 2}
	exClone := ex1.Clone()
	if exClone.Mem != ex1.Mem || exClone.Steps != ex1.Steps ||
		&exClone == &ex1 {
		t.Error("Clone failed")
	}

	ex2 := Redeemer.ExecutionUnits{Mem: 3, Steps: 4}
	ex1.Sum(ex2)
	if ex1.Mem != 4 || ex1.Steps != 6 {
		t.Error("Sum failed")
	}
}

func TestRedeemerClone(t *testing.T) {
	red := Redeemer.Redeemer{
		Tag:     Redeemer.SPEND,
		Index:   1,
		Data:    PlutusData.PlutusData{},
		ExUnits: Redeemer.ExecutionUnits{Mem: 1, Steps: 2},
	}
	redClone := red.Clone()
	if redClone.Tag != red.Tag || redClone.Index != red.Index ||
		&redClone.Data == &red.Data ||
		&redClone.ExUnits == &red.ExUnits {
		t.Error("Clone failed")
	}
}

func TestMarshalUnmarshalRedeemer(t *testing.T) {
	red := Redeemer.Redeemer{
		Tag:     Redeemer.SPEND,
		Index:   1,
		Data:    PlutusData.PlutusData{},
		ExUnits: Redeemer.ExecutionUnits{Mem: 1, Steps: 2},
	}
	marshaled, _ := cbor.Encode(red)
	expectedHex := hex.EncodeToString(marshaled)
	if hex.EncodeToString(
		marshaled,
	) != expectedHex {
		t.Error(
			"Invalid marshaling",
			hex.EncodeToString(marshaled),
			"Expected",
			expectedHex,
		)
	}
	var red2 Redeemer.Redeemer
	_, err := cbor.Decode(marshaled, &red2)
	if err != nil {
		t.Error("Failed unmarshaling", err)
	}
	if red2.Tag != red.Tag || red2.Index != red.Index ||
		&red2.Data == &red.Data ||
		&red2.ExUnits == &red.ExUnits {
		t.Error(
			"Invalid unmarshaling",
			red2.Tag,
			red2.Index,
			red2.Data,
			red2.ExUnits,
			"Expected",
			red.Tag,
			red.Index,
			red.Data,
			red.ExUnits,
		)
	}
}
