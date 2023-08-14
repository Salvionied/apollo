package transactionoutput_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/cbor/v2"
)

func TestTransactionOutputWithDatumHash(t *testing.T) {
	cborHex := "83583911a65ca58a4e9c755fa830173d2a5caed458ac0c73f97db7faae2e7e3b52563c5410bff6a0d43ccebb7c37e1f69f5eb260552521adff33b9c21a00895440582070c5d760293d3d92bfa7369e472891ab36041cbf81fd5ed103462fb7c03f2a6e"
	cborBytes, _ := hex.DecodeString(cborHex)
	txOut := TransactionOutput.TransactionOutput{}
	err := txOut.UnmarshalCBOR(cborBytes)
	if err != nil {
		t.Errorf("Error while unmarshaling")
	}
	outBytes, err := cbor.Marshal(txOut)
	if err != nil {
		t.Errorf("Error while marshaling")
	}

	if hex.EncodeToString(outBytes) != cborHex {
		t.Errorf("Invalid Reserialization")
	}
}
