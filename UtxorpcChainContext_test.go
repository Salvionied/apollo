package apollo_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	testutils "github.com/Salvionied/apollo/testUtils"
	"github.com/Salvionied/apollo/txBuilding/Backend/UtxorpcChainContext"
)

const UTXORPC_BASE_URL = "https://utxorpc.blinklabs.io/"

var decoded_addr_for_fixtures, _ = Address.DecodeAddress(
	"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
)

func TestUTXORPC_FailedSubmissionThrows(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
	)
	if err != nil {
		t.Error(err)
	}
	apollob := apollo.New(&cc)
	apollob, err = apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if _, err = cc.SubmitTx(*apollob.GetTx()); err == nil {
		t.Error("DIDNT THROW")
	}
}

func TestUTXORPC_BurnPlutus(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
			Value.SimpleValue(15_000_000, MultiAsset.MultiAsset[int64]{
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
				Quantity: -1,
			},
			Redeemer.Redeemer{},
		).
		Complete()
	if err != nil {
		// skip ExUnits-dependent tests for UTXORPC
		if strings.Contains(strings.ToLower(err.Error()), "estimate exunits") {
			t.Skip("Skipping ExUnit-dependent test (UTXORPC): " + err.Error())
		}
		t.Error(err)
	}
	tx := apollob.GetTx()
	if tx.TransactionBody.Mint == nil {
		t.Error("mint field is nil (expected negative quantity for burn)")
	}
}

func TestUTXORPC_MintPlutus(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
			Value.SimpleValue(15_000_000, nil),
		),
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
		// skip ExUnits-dependent tests for UTXORPC
		if strings.Contains(strings.ToLower(err.Error()), "estimate exunits") {
			t.Skip("Skipping ExUnit-dependent test (UTXORPC): " + err.Error())
		}
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(
		txBytes,
	) != "84a5008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000181825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00e1eefb021a0002f2c509a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354200b58207c2fde0c1908393e41c7b4afbfee2686378e714d70fc335b6cd0142ac6de9772a10581840000f6820000f5f6" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestUTXORPCMintPlutusWithPayment(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
			Value.SimpleValue(15_000_000, nil),
		),
	}

	apollob := apollo.New(&cc)
	apollob, err = apollob.
		AddLoadedUTxOs(testUtxo).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: 1,
			},
			Redeemer.Redeemer{},
		).
		PayToAddress(
			decoded_addr,
			1_200_000,
			apollo.NewUnit(policy.String(),
				"TEST",
				1,
			),
		).Complete()
	if err != nil {
		// skip ExUnits-dependent tests for UTXORPC
		if strings.Contains(strings.ToLower(err.Error()), "estimate exunits") {
			t.Skip("Skipping ExUnit-dependent test (UTXORPC): " + err.Error())
		}
		t.Error(err)
	}
	txBytes, err := apollob.GetTx().Bytes()
	if err != nil {
		t.Error(err)
	}
	if hex.EncodeToString(
		txBytes,
	) != "84a5008182584064356431663763323233646338386262343134373461663233623638356530323437333037653934653731356566356536326633323561633934663733303536000182825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c9821a001484d0a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa1445445535401825839010a59337f7b3a913424d7f7a151401e052642b68e948d8cacadc6372016a9999419cc5a61ca62da81e378d7538213a3715a6b858c948c69c91a00cd466b021a0003168509a1581c279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3fa14454455354010b58207c2fde0c1908393e41c7b4afbfee2686378e714d70fc335b6cd0142ac6de9772a10581840000f6820000f5f6" {
		t.Error("Tx is not correct", hex.EncodeToString(txBytes))
	}
}

func TestUTXORPC_GetWallet(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
	)
	if err != nil {
		t.Error(err)
	}

	apollob := apollo.New(&cc)
	if apollob.GetWallet() != nil {
		t.Error("Wallet should be nil initially")
	}
	addr := decoded_addr_for_fixtures.String()
	apollob = apollob.SetWalletFromBech32(addr)
	if apollob.GetWallet().GetAddress().String() != addr {
		t.Error("Wallet address is not correct")
	}
}

func TestUTXORPC_AddInputs(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
			Value.SimpleValue(15_000_000, nil),
		),
	}

	apollob := apollo.New(&cc)
	apollob = apollob.AddInput(testUtxo).SetChangeAddress(decoded_addr)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}
}

func TestUTXORPC_ConsumeUtxo(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
	apollob = apollob.SetChangeAddress(decoded_addr).
		ConsumeUTxO(testUtxo,
			apollo.NewPayment(decoded_addr_for_fixtures.String(), 2_000_000, nil),
			apollo.NewPayment(decoded_addr_for_fixtures.String(), 2_000_000, nil),
		).
		AddLoadedUTxOs(biAdaUtxo)

	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
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

func TestUTXORPC_ConsumeAssetsFromUtxo(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
	apollob = apollob.SetChangeAddress(decoded_addr).
		ConsumeAssetsFromUtxo(testUtxo,
			apollo.NewPayment(
				decoded_addr_for_fixtures.String(),
				2_000_000,
				[]apollo.Unit{
					apollo.NewUnit(
						"279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
						"TEST",
						1,
					),
				},
			),
		).
		AddLoadedUTxOs(biAdaUtxo)

	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Inputs[0].EqualTo(testUtxo.Input) {
		t.Error("Tx is not correct")
	}
	foundAssetOut := false
	for _, out := range built.GetTx().TransactionBody.Outputs {
		ma := out.GetValue().GetAssets()
		if ma == nil {
			continue
		}
		if a, ok := ma[Policy.PolicyId{Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f"}]; ok {
			if _, ok2 := a[AssetName.NewAssetNameFromString("TEST")]; ok2 {
				foundAssetOut = true
			}
		}
	}
	if !foundAssetOut {
		t.Error("expected an output carrying the TEST asset")
	}
}

func TestUTXORPC_PayToContract(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
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
		Value:          []byte("Hello, World!"),
	}

	apollob := apollo.New(&cc)
	apollob = apollob.SetChangeAddress(decoded_addr).AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, &datum, 1_000_000, false).
		PayToContract(decoded_addr, &datum, 1_000_000, true)

	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
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
	if built.GetTx().TransactionBody.Outputs[1].GetDatum() == nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[0].GetDatum() != nil {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].GetDatum().TagNr != 121 {
		t.Error(
			"Tx is not correct",
			built.GetTx().TransactionBody.Outputs[1].GetDatum().TagNr,
		)
	}
	if _, err := built.GetTx().Bytes(); err != nil {
		t.Errorf("failed to serialize tx: %v", err)
	}
}

func TestUTXORPC_RequiredSigner(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
	)
	if err != nil {
		t.Error(err)
	}

	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc).
		SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		AddRequiredSignerFromAddress(decoded_addr, true, true)
	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
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
		t.Error(err)
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
		t.Error(err)
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

func TestUTXORPC_FeePadding(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
	)
	if err != nil {
		t.Error(err)
	}

	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc).
		SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000)

	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if built.GetTx().TransactionBody.Fee != 691637 {
		t.Error("Tx is not correct", built.GetTx().TransactionBody.Fee)
	}
	if built.GetTx().TransactionBody.Outputs[0].Lovelace() != 1_000_000 {
		t.Error("Tx is not correct")
	}
	if built.GetTx().TransactionBody.Outputs[1].Lovelace() != 13308363 {
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

func TestUTXORPC_SetCollateral(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
	)
	if err != nil {
		t.Error(err)
	}

	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc).
		SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000).
		AddCollateral(collateralUtxo)

	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
	}
	if !built.GetTx().TransactionBody.Collateral[0].EqualTo(
		collateralUtxo.Input,
	) {
		t.Error("Tx is not correct")
	}
}

func TestUTXORPC_CollateralWithReturn(t *testing.T) {
	cc, err := UtxorpcChainContext.NewUtxorpcChainContext(
		UTXORPC_BASE_URL,
		int(MAINNET),
	)
	if err != nil {
		t.Error(err)
	}

	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	apollob := apollo.New(&cc).
		SetChangeAddress(decoded_addr).
		AddLoadedUTxOs(InputUtxo).
		PayToContract(decoded_addr, nil, 1_000_000, false).
		SetFeePadding(500_000).
		AddCollateral(collateralUtxo2)

	built, err := apollob.Complete()
	if err != nil {
		t.Error(err)
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
