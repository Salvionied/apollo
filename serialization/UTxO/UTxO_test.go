package UTxO_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2/serialization/Address"
	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/Salvionied/apollo/v2/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/v2/serialization/UTxO"
	"github.com/Salvionied/apollo/v2/serialization/Value"
)

func TestUTxOUTils(t *testing.T) {
	addr, _ := Address.DecodeAddress(
		"addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh",
	)
	decodedTxHash, _ := hex.DecodeString(
		"a357be4527a7afc2cebf37259fd4c5f220540d7dca90721e386cfe4865c107c6",
	)
	utxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: decodedTxHash,
			Index:         0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			addr,
			Value.PureLovelaceValue(1000000),
		),
	}
	if utxo.GetKey() != "a357be4527a7afc2cebf37259fd4c5f220540d7dca90721e386cfe4865c107c6:0" {
		t.Errorf("UTxO key is incorrect")
	}
	utxo2 := utxo.Clone()
	if !utxo.EqualTo(utxo2) {
		t.Errorf("UTxO should be equal to its clone")
	}
}

func TestEqualTo(t *testing.T) {
	addr, _ := Address.DecodeAddress(
		"addr1qxajla3qcrwckzkur8n0lt02rg2sepw3kgkstckmzrz4ccfm3j9pqrqkea3tns46e3qy2w42vl8dvvue8u45amzm3rjqvv2nxh",
	)
	utxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte{0x01, 0x02, 0x03},
			Index:         0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			addr,
			Value.PureLovelaceValue(1000000),
		),
	}
	utxo2 := utxo.Clone()
	if !utxo.EqualTo(utxo2) {
		t.Errorf("UTxO should be equal to its clone")
	}
	utxo3 := utxo.Clone()
	utxo3.Input.Index = 1
	if utxo.EqualTo(utxo3) {
		t.Errorf("UTxO should not be equal to utxo3")
	}

}
