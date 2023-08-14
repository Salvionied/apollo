package transactionoutput_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/Value"
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

func TestPostAlonzo(t *testing.T) {
	txO := TransactionOutput.TransactionOutput{}
	cborHex := "d8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd87a80d8799fd8799f581c25f0fc240e91bd95dcdaebd2ba7713fc5168ac77234a3d79449fc20c47534f4349455459ff1b00002cc16be02b37ff1a001e84801a001e8480ff"
	decoded_cbor, _ := hex.DecodeString(cborHex)
	var pd PlutusData.PlutusData
	cbor.Unmarshal(decoded_cbor, &pd)
	txO.IsPostAlonzo = true
	decoded_address, _ := Address.DecodeAddress("addr1wynp362vmvr8jtc946d3a3utqgclfdl5y9d3kn849e359hsskr20n")
	txO.PostAlonzo = TransactionOutput.TransactionOutputAlonzo{}
	txO.PostAlonzo.Address = decoded_address
	txO.PostAlonzo.Amount = Value.PureLovelaceValue(1000000).ToAlonzoValue()
	txO.PostAlonzo.Datum = &pd
	resultHex := "a300581d712618e94cdb06792f05ae9b1ec78b0231f4b7f4215b1b4cf52e6342de01821a000f4240a002d8799fd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd8799fd8799f581c37dce7298152979f0d0ff71fb2d0c759b298ac6fa7bc56b928ffc1bcffd8799fd8799fd8799f581cf68864a338ae8ed81f61114d857cb6a215c8e685aa5c43bc1f879cceffffffffd87a80d8799fd8799f581c25f0fc240e91bd95dcdaebd2ba7713fc5168ac77234a3d79449fc20c47534f4349455459ff1b00002cc16be02b37ff1a001e84801a001e8480ff"
	cborred, _ := cbor.Marshal(txO)
	if hex.EncodeToString(cborred) != resultHex {
		fmt.Println(hex.EncodeToString(cborred))
		t.Errorf("Invalid marshaling")
	}

}
