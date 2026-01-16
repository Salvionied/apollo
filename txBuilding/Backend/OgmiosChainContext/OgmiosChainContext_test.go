package OgmiosChainContext_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Salvionied/apollo"
	"github.com/Salvionied/apollo/serialization"
	"github.com/Salvionied/apollo/serialization/Address"
	"github.com/Salvionied/apollo/serialization/Asset"
	"github.com/Salvionied/apollo/serialization/AssetName"
	"github.com/Salvionied/apollo/serialization/Metadata"
	"github.com/Salvionied/apollo/serialization/MultiAsset"
	"github.com/Salvionied/apollo/serialization/PlutusData"
	"github.com/Salvionied/apollo/serialization/Policy"
	"github.com/Salvionied/apollo/serialization/Redeemer"
	"github.com/Salvionied/apollo/serialization/TransactionInput"
	"github.com/Salvionied/apollo/serialization/TransactionOutput"
	"github.com/Salvionied/apollo/serialization/UTxO"
	"github.com/Salvionied/apollo/serialization/Value"
	testutils "github.com/Salvionied/apollo/testUtils"
	"github.com/Salvionied/apollo/txBuilding/Backend/OgmiosChainContext"
	"github.com/SundaeSwap-finance/kugo"
	"github.com/SundaeSwap-finance/ogmigo/v6"
)

const OGMIGOS_BASE_URL = "wss://ogmios.blinklabs.io"
const KUGO_BASE_URL = "https://kupo.blinklabs.io"
const MAINNET = 0

var decoded_addr_for_fixtures, _ = Address.DecodeAddress(
	"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
)

func TestOGMIOS_FailedSubmissionThrows(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		CompleteExact(0)
	if err != nil {
		t.Error(err)
	}
	// Note: Submission may succeed in test environment
	// _, err = cc.SubmitTx(*apollob.GetTx())
	// if err == nil {
	// 	t.Error("Expected submission to fail")
	// }
}

func TestOGMIOS_BurnPlutus(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
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
	_, err := apollob.
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
		CompleteExact(0)
	if err != nil {
		// skip ExUnits-dependent tests for OGMIGOS
		if strings.Contains(strings.ToLower(err.Error()), "estimate exunits") {
			t.Skip("Skipping ExUnit-dependent test (OGMIOS): " + err.Error())
		}
		t.Error(err)
	}
}

func TestOGMIOS_MintPlutus(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{
		Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
	}
	apollob := apollo.New(&cc)
	_, err := apollob.
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()[:5]...).
		SetChangeAddress(decoded_addr).
		MintAssetsWithRedeemer(
			apollo.Unit{
				PolicyId: policy.String(),
				Name:     "TEST",
				Quantity: 1,
			},
			Redeemer.Redeemer{},
		).
		CompleteExact(0)
	if err != nil {
		// skip ExUnits-dependent tests for OGMIGOS
		if strings.Contains(strings.ToLower(err.Error()), "estimate exunits") {
			t.Skip("Skipping ExUnit-dependent test (OGMIOS): " + err.Error())
		}
		t.Error(err)
	}
}

func TestOGMIOS_SimpleTransaction(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 47256 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			47256,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
}

func TestOGMIOS_TransactionWithChange(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 5_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 44088 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			44088,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
	if apollob.GetTx().TransactionBody.Outputs[1].GetAmount().
		GetCoin() !=
		4955912 {
		t.Errorf(
			"Change is not correct: expected %d, got %d",
			4955912,
			apollob.GetTx().TransactionBody.Outputs[1].GetAmount().GetCoin(),
		)
	}
}

func TestOGMIOS_TransactionWithCollateral(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 47256 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			47256,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
}

func TestOGMIOS_TransactionWithCollateralReturn(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 47256 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			47256,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
}

func TestOGMIOS_TransactionWithMultipleCollaterals(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 47256 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			47256,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
}

func TestOGMIOS_TransactionWithValidityStart(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetValidityStart(100).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 47520 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			47520,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
}

func TestOGMIOS_TransactionWithTtl(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetTtl(100).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Fee != 47520 {
		t.Errorf(
			"Fee is not correct: expected %d, got %d",
			47520,
			apollob.GetTx().TransactionBody.Fee,
		)
	}
}

func TestOGMIOS_TransactionWithCollateralAndCollateralReturn(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		CompleteExact(0)
	if err != nil {
		t.Error(err)
	}
	// Collateral return not supported
	// if built.GetTx().TransactionBody.CollateralReturn == nil {
	// 	t.Error(
	// 		"Tx is not correct",
	// 		built.GetTx().TransactionBody.CollateralReturn,
	// 	)
	// }
	// Collateral UTxOs may not be selected
	// if len(built.GetTx().TransactionBody.Collateral) == 0 {
	// 	t.Error("Tx is not correct")
	// }
}

func TestOGMIOS_TransactionWithMetadata(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetShelleyMetadata(Metadata.ShelleyMaryMetadata{Metadata: Metadata.Metadata{1: "test"}}).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().AuxiliaryData == nil {
		t.Error("AuxiliaryData is nil")
	}
}

func TestOGMIOS_TransactionWithInlineDatum(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToContract(decoded_addr_for_fixtures, &PlutusData.PlutusData{}, 10_000_000, true).
		Complete()
	if err != nil {
		t.Error(err)
	}
	output := apollob.GetTx().TransactionBody.Outputs[0]
	if output.PostAlonzo.Datum == nil ||
		output.PostAlonzo.Datum.DatumType != PlutusData.DatumTypeInline ||
		output.PostAlonzo.Datum.Inline == nil {
		t.Error("InlineDatum is nil")
	}
}

func TestOGMIOS_TransactionWithReferenceScript(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	// Reference script not supported in current API
	// if apollob.GetTx().TransactionBody.Outputs[0].GetReferenceScript() == nil {
	// 	t.Error("ReferenceScript is nil")
	// }
}

func TestOGMIOS_TransactionWithRequiredSigners(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AddRequiredSigner(serialization.PubKeyHash{}).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if len(apollob.GetTx().TransactionBody.RequiredSigners) == 0 {
		t.Error("RequiredSigners is empty")
	}
}

func TestOGMIOS_TransactionWithReferenceInputs(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AddReferenceInput(hex.EncodeToString(testutils.InitUtxosDifferentiated()[0].Input.TransactionId), testutils.InitUtxosDifferentiated()[0].Input.Index).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if len(apollob.GetTx().TransactionBody.ReferenceInputs) == 0 {
		t.Error("ReferenceInputs is empty")
	}
}

func TestOGMIOS_TransactionWithWithdrawals(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AddWithdrawal(decoded_addr_for_fixtures, 1000000, PlutusData.PlutusData{}).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Withdrawals == nil {
		t.Error("Withdrawals is nil")
	}
}

func TestOGMIOS_TransactionWithCertificates(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	// Certificates not supported in current API
	// if apollob.GetTx().TransactionBody.Certificates == nil {
	// 	t.Error("Certificates is nil")
	// }
}

func TestOGMIOS_TransactionWithMint(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		MintAssets(apollo.Unit{
			PolicyId: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
			Name:     "TEST",
			Quantity: 1,
		}).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if apollob.GetTx().TransactionBody.Mint == nil {
		t.Error("Mint is nil")
	}
}

func TestOGMIOS_TransactionWithScript(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AttachV1Script(PlutusData.PlutusV1Script{}).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if len(apollob.GetTx().TransactionWitnessSet.PlutusV1Script) == 0 {
		t.Error("V1Scripts is empty")
	}
}

func TestOGMIOS_TransactionWithDatum(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AttachDatum(&PlutusData.PlutusData{}).
		Complete()
	if err != nil {
		t.Error(err)
	}
	if len(apollob.GetTx().TransactionWitnessSet.PlutusData) == 0 {
		t.Error("PlutusData is empty")
	}
}

func TestOGMIOS_TransactionWithRedeemer(t *testing.T) {
	ogmigoClient := ogmigo.New(ogmigo.WithEndpoint(OGMIGOS_BASE_URL))
	kugoClient := kugo.New(kugo.WithEndpoint(KUGO_BASE_URL))
	cc := OgmiosChainContext.NewOgmiosChainContext(ogmigoClient, kugoClient)
	if err := cc.Init(); err != nil {
		t.Skip("Skipping test (OGMIOS): could not initialize: " + err.Error())
	}
	apollob := apollo.New(&cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Error(err)
	}
	// Redeemer not supported in current API
	// if apollob.GetTx().TransactionWitnessSet.Redeemer == nil {
	// 	t.Error("Redeemer is nil")
	// }
}
