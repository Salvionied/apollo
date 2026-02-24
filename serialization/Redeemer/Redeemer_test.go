package Redeemer_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/fxamacker/cbor/v2"
)

func TestExecutionUnitsFunctions(t *testing.T) {
	ex1 := Redeemer.ExecutionUnits{
		Mem:   1,
		Steps: 2,
	}
	exClone := ex1.Clone()
	if exClone.Mem != ex1.Mem || exClone.Steps != ex1.Steps ||
		&exClone == &ex1 {
		t.Error("Clone failed")
	}

	ex2 := Redeemer.ExecutionUnits{
		Mem:   3,
		Steps: 4,
	}
	ex1.Sum(ex2)
	if ex1.Mem != 4 || ex1.Steps != 6 {
		t.Error("Sum failed")
	}
}

func TestRedeemerClone(t *testing.T) {
	red := Redeemer.Redeemer{
		Tag:   Redeemer.SPEND,
		Index: 1,
		Data:  PlutusData.PlutusData{},
		ExUnits: Redeemer.ExecutionUnits{
			Mem:   1,
			Steps: 2,
		},
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
		Tag:   Redeemer.SPEND,
		Index: 1,
		Data:  PlutusData.PlutusData{},
		ExUnits: Redeemer.ExecutionUnits{
			Mem:   1,
			Steps: 2,
		},
	}
	marshaled, _ := cbor.Marshal(red)
	if hex.EncodeToString(marshaled) != "840001f6820102" {
		t.Error(
			"Invalid marshaling",
			hex.EncodeToString(marshaled),
			"Expected",
			"840001f6820102",
		)
	}
	var red2 Redeemer.Redeemer
	err := cbor.Unmarshal(marshaled, &red2)
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

func TestRedeemersMapMarshal(t *testing.T) {
	r := Redeemer.Redeemers{
		Redeemers: []Redeemer.Redeemer{
			{
				Tag:   Redeemer.MINT,
				Index: 0,
				Data:  PlutusData.PlutusData{},
				ExUnits: Redeemer.ExecutionUnits{
					Mem: 100, Steps: 200,
				},
			},
			{
				Tag:   Redeemer.SPEND,
				Index: 0,
				Data:  PlutusData.PlutusData{},
				ExUnits: Redeemer.ExecutionUnits{
					Mem: 50, Steps: 100,
				},
			},
		},
	}
	data, err := r.MarshalCBOR()
	if err != nil {
		t.Fatal("MarshalCBOR failed:", err)
	}
	// Map header 0xa2 = 2 entries
	if data[0] != 0xa2 {
		t.Errorf(
			"expected map header 0xa2, got 0x%02x",
			data[0],
		)
	}
	// SPEND (tag=0) should come before MINT (tag=1)
	// due to canonical sorting by (Tag, Index).
	// First key: [0, 0] = SPEND:0
	// Second key: [1, 0] = MINT:0
	firstKey := hex.EncodeToString(data[1:4])
	if firstKey != "820000" {
		t.Errorf(
			"expected first key 820000 (SPEND:0), got %s",
			firstKey,
		)
	}
}

func TestRedeemersLargeMapHeader(t *testing.T) {
	redeemers := make([]Redeemer.Redeemer, 25)
	for i := range redeemers {
		redeemers[i] = Redeemer.Redeemer{
			Tag:   Redeemer.SPEND,
			Index: i,
			Data:  PlutusData.PlutusData{},
			ExUnits: Redeemer.ExecutionUnits{
				Mem: 1, Steps: 1,
			},
		}
	}
	r := Redeemer.Redeemers{Redeemers: redeemers}
	data, err := r.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	// 0xb8 = map with 1-byte length, 25 entries
	if data[0] != 0xb8 || data[1] != 25 {
		t.Errorf(
			"expected header [b8 19], got [%02x %02x]",
			data[0], data[1],
		)
	}
}

func TestRedeemersEmptyMarshal(t *testing.T) {
	r := Redeemer.Redeemers{
		Redeemers: []Redeemer.Redeemer{},
	}
	data, err := r.MarshalCBOR()
	if err != nil {
		t.Fatal("MarshalCBOR failed:", err)
	}
	// Empty map = 0xa0
	if len(data) != 1 || data[0] != 0xa0 {
		t.Errorf(
			"expected empty map [a0], got %s",
			hex.EncodeToString(data),
		)
	}
}
