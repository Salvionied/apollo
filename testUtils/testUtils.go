package testutils

import (
	"fmt"

	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
)

var TESTADDRESS = "addr_test1vrm9x2zsux7va6w892g38tvchnzahvcd9tykqf3ygnmwtaqyfg52x"

func InitUtxos() []UTxO.UTxO {
	utxos := make([]UTxO.UTxO, 0)
	for i := 0; i < 10; i++ {
		tx_in := TransactionInput.TransactionInput{
			TransactionId: make([]byte, 32),
			Index:         i,
		}

		Addr, _ := Address.DecodeAddress(TESTADDRESS)
		policy := Policy.PolicyId{Value: "00000000000000000000000000000000000000000000000000000000"}
		asset_name := AssetName.NewAssetNameFromString(fmt.Sprintf("token%d", i))
		Asset := Asset.Asset[int64]{
			asset_name: int64((i + 1) * 100)}
		assets := MultiAsset.MultiAsset[int64]{policy: Asset}
		value := Value.SimpleValue(int64((i+1)*1000000),
			assets)
		tx_out := TransactionOutput.SimpleTransactionOutput(
			Addr, value)
		utxos = append(utxos, UTxO.UTxO{Input: tx_in, Output: tx_out})
	}
	return utxos
}
func InitUtxosDifferentiated() []UTxO.UTxO {
	utxos := make([]UTxO.UTxO, 0)
	for i := 0; i < 10; i++ {
		tx_in := TransactionInput.TransactionInput{
			TransactionId: make([]byte, 32),
			Index:         i,
		}

		Addr, _ := Address.DecodeAddress(TESTADDRESS)
		policy := Policy.PolicyId{Value: "00000000000000000000000000000000000000000000000000000000"}
		singleasset := Asset.Asset[int64]{}
		for j := 0; j < i; j++ {
			asset_name := AssetName.NewAssetNameFromString(fmt.Sprintf("token%d", j))
			singleasset[asset_name] = int64((i + 1) * 100)
		}

		assets := MultiAsset.MultiAsset[int64]{policy: singleasset}
		value := Value.SimpleValue(int64((i+1)*1000000),
			assets)
		tx_out := TransactionOutput.SimpleTransactionOutput(
			Addr, value)
		utxos = append(utxos, UTxO.UTxO{Input: tx_in, Output: tx_out})
	}
	return utxos
}
