package TransactionWitnessSet_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/Salvionied/apollo/v2/serialization/TransactionWitnessSet"
	"github.com/blinklabs-io/gouroboros/cbor"
)

func TestMarshalAndUnmarshalNoScripts(t *testing.T) {
	tws := TransactionWitnessSet.TransactionWitnessSet{}
	twsBytes, err := cbor.Encode(tws)
	if err != nil {
		t.Errorf("Error marshaling TransactionWitnessSet: %v", err)
	}
	if hex.EncodeToString(twsBytes) != "a0" {
		t.Error(
			"TransactionWitnessSet marshaled incorrectly",
			hex.EncodeToString(twsBytes),
		)
	}
}

var pd = PlutusData.PlutusData{
	PlutusDataType: PlutusData.PlutusBytes,
	TagNr:          0,
	Value:          []byte{0x01, 0x02, 0x03},
}

func TestMarshalBasicPlutus(t *testing.T) {
	tws := TransactionWitnessSet.TransactionWitnessSet{
		PlutusData: &PlutusData.PlutusIndefArray{pd},
	}
	twsBytes, err := cbor.Encode(tws)
	if err != nil {
		t.Errorf("Error marshaling TransactionWitnessSet: %v", err)
	}
	if hex.EncodeToString(twsBytes) != "a1049f43010203ff" {
		t.Error(
			"TransactionWitnessSet marshaled incorrectly",
			hex.EncodeToString(twsBytes),
		)
	}
}
