package txBuilding_test

import (
	"reflect"
	"testing"

	"github.com/salvionied/apollo/serialization/Address"
	"github.com/salvionied/apollo/serialization/Asset"
	"github.com/salvionied/apollo/serialization/AssetName"
	"github.com/salvionied/apollo/serialization/MultiAsset"
	"github.com/salvionied/apollo/serialization/Policy"
	"github.com/salvionied/apollo/serialization/TransactionOutput"
	"github.com/salvionied/apollo/serialization/Value"
	"github.com/salvionied/apollo/txBuilding/Backend/FixedChainContext"
	"github.com/salvionied/apollo/txBuilding/TxBuilder"
)

var decoded_address, _ = Address.DecodeAddress("addr_test1vrm9x2zsux7va6w892g38tvchnzahvcd9tykqf3ygnmwtaqyfg52x")

func TestTxBuilderSimple(t *testing.T) {
	ctx := FixedChainContext.InitFixedChainContext()
	txBuilder := TxBuilder.InitBuilder(ctx)
	txBuilder.AddInputAddress(decoded_address)

	txBuilder.AddOutput(
		TransactionOutput.SimpleTransactionOutput(
			decoded_address, Value.PureLovelaceValue(500000),
		), nil, false,
	)

	body, err := txBuilder.Build(&decoded_address, false, &decoded_address)

	if err != nil {
		t.Error(err)
	}
	usedHash := []byte{0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02}
	if !reflect.DeepEqual(body.Inputs[0].TransactionId, usedHash) {
		t.Error("Invalid transaction id")
	}
	if body.Inputs[0].Index != 1 {
		t.Error("Invalid transaction index")
	}

	if body.Outputs[0].Lovelace() != 500_000 {
		t.Error("Invalid output value")
	}

	if len(body.Outputs) < 2 {
		t.Error("Invalid output length")
	}

	if !body.Outputs[1].GetValue().HasAssets {
		t.Error("Invalid output value")
	}

	expected_output := MultiAsset.MultiAsset[int64]{Policy.PolicyId{Value: "11111111111111111111111111111111111111111111111111111111", Tp: "string"}: Asset.Asset[int64]{AssetName.NewAssetNameFromString("Token1"): 1, AssetName.NewAssetNameFromString("Token2"): 2}}

	if !body.Outputs[1].GetAmount().GetAssets().Equal(expected_output) {
		t.Error("Invalid output value")
	}
}
