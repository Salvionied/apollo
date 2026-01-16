package BlockFrostChainContext_test

import (
	"encoding/hex"
	"testing"

	"github.com/Salvionied/apollo/v2"
	"github.com/Salvionied/apollo/v2/serialization"
	"github.com/Salvionied/apollo/v2/serialization/Address"
	"github.com/Salvionied/apollo/v2/serialization/Asset"
	"github.com/Salvionied/apollo/v2/serialization/AssetName"
	"github.com/Salvionied/apollo/v2/serialization/MultiAsset"
	"github.com/Salvionied/apollo/v2/serialization/PlutusData"
	"github.com/Salvionied/apollo/v2/serialization/Policy"
	"github.com/Salvionied/apollo/v2/serialization/Redeemer"
	"github.com/Salvionied/apollo/v2/serialization/TransactionInput"
	"github.com/Salvionied/apollo/v2/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/v2/serialization/UTxO"
	"github.com/Salvionied/apollo/v2/serialization/Value"
	testutils "github.com/Salvionied/apollo/v2/testUtils"
	"github.com/Salvionied/apollo/v2/txBuilding/Backend/BlockFrostChainContext"
)

type Network int

const (
	MAINNET Network = iota
	TESTNET
	PREVIEW
	PREPROD
)

const BLOCKFROST_BASE_URL_MAINNET = "https://cardano-mainnet.blockfrost.io/api"

const BLOCKFROST_API_KEY = "mainnet88ZdHRG3UHXf2IEIT098i53GWWpbZWlU"

var decoded_addr, _ = Address.DecodeAddress(
	"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
)

var InputUtxo = UTxO.UTxO{
	Input: TransactionInput.TransactionInput{
		TransactionId: []byte(
			"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
		),
		Index: 1,
	},
	Output: TransactionOutput.SimpleTransactionOutput(
		decoded_addr,
		Value.SimpleValue(15_000_000, nil)),
}

var collateralUtxo = UTxO.UTxO{
	Input: TransactionInput.TransactionInput{
		TransactionId: []byte(
			"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
		),
		Index: 1,
	},
	Output: TransactionOutput.SimpleTransactionOutput(
		decoded_addr,
		Value.SimpleValue(15_000_000, nil)),
}

var collateralUtxo2 = UTxO.UTxO{
	Input: TransactionInput.TransactionInput{
		TransactionId: []byte(
			"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
		),
		Index: 1,
	},
	Output: TransactionOutput.SimpleTransactionOutput(
		decoded_addr,
		Value.SimpleValue(5_000_000, nil)),
}

func TestFailedSubmissionThrows(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	apollob := apollo.New(&cc)
	apollob, err = apollob.
		AddInputAddressFromBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu").
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 10_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	_, err = cc.SubmitTx(*apollob.GetTx())
	if err == nil {
		t.Error("DIDNT THROW")
	}
}

func TestBurnPlutus(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}

	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{
		Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
	}
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, MultiAsset.MultiAsset[int64]{
				policy: Asset.Asset[int64]{
					AssetName.NewAssetNameFromString("TEST"): 1,
				},
			})),
	}

	apollob := apollo.New(&cc)
	apollob, err = apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: int(-1),
			},
			Redeemer.Redeemer{},
		).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if apollob == nil {
		t.Fatal("Complete returned nil")
	}
	txBytes, err := apollob.GetTx().Bytes()
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(
		txBytes,
	) != "84a6008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000181825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00e1ddcb021a000303f5075820d36a2619a672494604e11bb447cbcf5231e9f2ba25c2169177edc941bd50ad6c09a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354200b5820ef7bdf2d3a3e3e7ce671c79236885a10d52d97ce3d66f7e9f1ca61fe6c8fe5d4a1058184000080820000f5a0" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestMintPlutus(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{
		Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
	}
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, nil)),
	}

	apollob := apollo.New(&cc)
	apollob, err = apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: int(1),
			},
			Redeemer.Redeemer{},
		).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if apollob == nil {
		t.Fatal("Complete returned nil")
	}
	txBytes, err := apollob.GetTx().Bytes()
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(
		txBytes,
	) != "84a6008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000181825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c9821a00e1d063a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa1445445535401021a0003115d075820d36a2619a672494604e11bb447cbcf5231e9f2ba25c2169177edc941bd50ad6c09a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354010b5820ef7bdf2d3a3e3e7ce671c79236885a10d52d97ce3d66f7e9f1ca61fe6c8fe5d4a1058184000080820000f5a0" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestMintPlutusWithPayment(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{
		Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
	}
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, nil)),
	}
	apollob := apollo.New(&cc)
	apollob, err = apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: int(1),
			},
			Redeemer.Redeemer{},
		).PayToAddress(
		decoded_addr,
		1200000,
		apollo.NewUnit(
			policy.String(),
			"TEST",
			1,
		),
	).Complete()

	if err != nil {
		t.Fatal(err)
	}
	if apollob == nil {
		t.Fatal("Complete returned nil")
	}
	txBytes, err := apollob.GetTx().Bytes()
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(
		txBytes,
	) != "84a6008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000182825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c9821a00144178a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa1445445535401825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00cd7893021a000327b5075820d36a2619a672494604e11bb447cbcf5231e9f2ba25c2169177edc941bd50ad6c09a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354010b5820ef7bdf2d3a3e3e7ce671c79236885a10d52d97ce3d66f7e9f1ca61fe6c8fe5d4a1058184000080820000f5a0" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestGetWallet(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	apollob := apollo.New(&cc)
	wall := apollob.GetWallet()
	if wall != nil {
		t.Error("Wallet should be nil")
	}
	apollob = apollob.SetWalletFromBech32(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	wallet := apollob.GetWallet()
	if wallet.GetAddress().
		String() !=
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu" {
		t.Error("Wallet address is not correct")
	}
}

func TestAddInputs(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15000000, nil)),
	}
	apollob := apollo.New(&cc)
	apollob = apollob.AddInput(testUtxo).SetChangeAddress(decoded_addr)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 0 {
		t.Error("Tx is not correct")
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 0 {
		t.Error("Tx is not correct")
	}

}

func TestConsumeUtxo(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, nil)),
	}
	biAdaUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 1,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, nil)),
	}

	apollob := apollo.New(&cc)
	apollob, err = apollob.SetChangeAddress(decoded_addr).
		ConsumeUTxO(testUtxo,
			apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 2_000_000, nil),
			apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 2_000_000, nil),
		)
	if err != nil {
		t.Fatal(err)
	}
	apollob = apollob.AddLoadedUTxOs(biAdaUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}
	if len(built.GetTx().TransactionBody.Outputs) != 4 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 2_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 2_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[2].Lovelace() != 11_000_000 {
		t.Error("Tx is not correct")
	}
}

func TestConsumeAssetsFromUtxo(t *testing.T) {

	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	testUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 0,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, MultiAsset.MultiAsset[int64]{
				Policy.PolicyId{Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f"}: Asset.Asset[int64]{
					AssetName.NewAssetNameFromString("TEST"): 1,
				},
			})),
	}
	biAdaUtxo := UTxO.UTxO{
		Input: TransactionInput.TransactionInput{
			TransactionId: []byte(
				"d5d1f7c223dc88bb41474af23b685e0247307e94e715ef5e62f325ac94f73056",
			),
			Index: 1,
		},
		Output: TransactionOutput.SimpleTransactionOutput(
			decoded_addr,
			Value.SimpleValue(15_000_000, nil)),
	}

	apollob := apollo.New(&cc)
	apollob, err = apollob.SetChangeAddress(decoded_addr).
		ConsumeAssetsFromUtxo(testUtxo,
			apollo.NewPayment("addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu", 2_000_000, []apollo.Unit{apollo.NewUnit("279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f", "TEST", 1)}),
		)
	if err != nil {
		t.Fatal(err)
	}
	apollob = apollob.AddLoadedUTxOs(biAdaUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}
	if len(built.GetTx().TransactionBody.Outputs) != 3 {
		t.Error("Tx is not correct")
	}

	if len(
		built.GetTx().TransactionBody.Outputs[0].GetValue().GetAssets(),
	) != 1 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 15_000_000 {
		t.Error("Tx is not correct")
	}
	if len(
		built.GetTx().TransactionBody.Outputs[1].GetValue().GetAssets(),
	) != 0 {
		t.Error("Tx is not correct")
	}
}

func TestPayToContract(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	datum := PlutusData.PlutusData{
		TagNr:          121,
		PlutusDataType: PlutusData.PlutusBytes,
		Value:          []byte("Hello, World!")}

	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, &datum, 1_000_000, false).
		PayToContract(decoded_addr, &datum, 1_000_000, true)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if len(built.GetTx().TransactionBody.Outputs) != 3 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].IsPostAlonzo {
		t.Error("Tx is not correct")
	}
	if !built.GetTx().TransactionBody.Outputs[1].IsPostAlonzo {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].GetDatumHash() == nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].GetDatumHash() != nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionWitnessSet.PlutusData == nil {
		t.Error("Tx is not correct")
	}
	datum1 := built.GetTx().TransactionBody.Outputs[1].GetDatum()
	if datum1 == nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].GetDatum() != nil {
		t.Error("Tx is not correct")
	}
	if datum1 != nil && datum1.TagNr != 121 {
		t.Error(
			"Tx is not correct",
			datum1.TagNr,
		)
	}
	txBytes, _ := built.GetTx().Bytes()
	if hex.EncodeToString(
		txBytes,
	) != "84a5008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536010183835839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a000f4240582037ead362f4ab7844a8416b045caa46a91066d391c16ae4d4a81557f14f7a0984845839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a000f42408201d81850d8794d48656c6c6f2c20576f726c6421f6825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00c32353021a000339ed075820d36a2619a672494604e11bb447cbcf5231e9f2ba25c2169177edc941bd50ad6c0b5820a1f321beb89d87d81b931988fc9561df62357c5b845ff48fdb926265a43110e2a1049fd8794d48656c6c6f2c20576f726c6421fff5a0" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}

}

func TestRequiredSigner(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		AddRequiredSignerFromAddress(decoded_addr, true, true)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 2 {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[0][:],
	) != hex.EncodeToString(
		decoded_addr.PaymentPart,
	) {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[1][:],
	) != hex.EncodeToString(
		decoded_addr.StakingPart,
	) {
		t.Error("Tx is not correct")
	}
	apollob = apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		AddRequiredSignerFromBech32(decoded_addr.String(), true, true)
	built, err = apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 2 {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[0][:],
	) != hex.EncodeToString(
		decoded_addr.PaymentPart,
	) {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[1][:],
	) != hex.EncodeToString(
		decoded_addr.StakingPart,
	) {
		t.Error("Tx is not correct")
	}

	apollob = apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		AddRequiredSigner(serialization.PubKeyHash(decoded_addr.PaymentPart))
	built, err = apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if len(built.GetTx().TransactionBody.RequiredSigners) != 1 {
		t.Error("Tx is not correct")
	}
	if hex.EncodeToString(
		built.GetTx().TransactionBody.RequiredSigners[0][:],
	) != hex.EncodeToString(
		decoded_addr.PaymentPart,
	) {
		t.Error("Tx is not correct")
	}
}

func TestFeePadding(t *testing.T) {
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if built.GetTx().TransactionBody.Fee != 696037 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Fee)
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 13303963 {
		t.Error(
			"Tx is not correct",
			built.GetTx().TransactionBody.Outputs[1].Lovelace(),
		)
	}
	if built.GetTx().TransactionBody.Outputs[0].IsPostAlonzo &&
		built.GetTx().TransactionBody.Outputs[0].GetDatumHash() != nil {
		t.Error(
			"Tx is not correct",
			built.GetTx().TransactionBody.Outputs[0].GetDatumHash(),
			built.GetTx().TransactionBody.Outputs[0].IsPostAlonzo,
		)
	}
}

func TestSetCollateral(t *testing.T) {
	// full 5 ada collateral
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000).
		AddCollateral(collateralUtxo)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if !built.GetTx().TransactionBody.Collateral[0].EqualTo(
		collateralUtxo.Input,
	) {
		t.Error("Tx is not correct")
	}
}

func TestCollateralwithReturn(t *testing.T) {
	// full 5 ada collateral
	cc, err := BlockFrostChainContext.NewBlockfrostChainContext(
		BLOCKFROST_BASE_URL_MAINNET,
		int(MAINNET),
		BLOCKFROST_API_KEY,
	)
	if err != nil {
		t.Error(err)
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000).
		AddCollateral(collateralUtxo2)
	built, err := apollob.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if built == nil {
		t.Fatal("Complete returned nil")
	}
	if built.GetTx().TransactionBody.TotalCollateral != 5_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.CollateralReturn.Lovelace() != 5_000_000 {
		t.Error(
			"Tx is not correct",
			built.GetTx().TransactionBody.CollateralReturn,
		)
	}
	if !built.GetTx().TransactionBody.Collateral[0].EqualTo(
		collateralUtxo2.Input,
	) {
		t.Error("Tx is not correct")
	}
}
