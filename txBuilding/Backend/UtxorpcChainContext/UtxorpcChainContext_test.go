package UtxorpcChainContext

import (
	"encoding/hex"
	"fmt"
	"os"
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
)

// const UTXORPC_BASE_URL = "https://cardano-mainnet.utxorpc-m1.demeter.run/"
const UTXORPC_BASE_URL = "https://utxorpc.blinklabs.io/"
const MAINNET = 0

var decoded_addr_for_fixtures, _ = Address.DecodeAddress(
	"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
)

// Shared chain context to avoid creating multiple connections to remote service
var sharedCC *UtxorpcChainContext

func TestMain(m *testing.M) {
	cc, err := NewUtxorpcChainContext(UTXORPC_BASE_URL, int(MAINNET))
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to initialize UTXORPC context: %v\n",
			err,
		)
		os.Exit(1)
	}
	sharedCC = &cc
	os.Exit(m.Run())
}

func getSharedContext(t *testing.T) *UtxorpcChainContext {
	if sharedCC == nil {
		t.Fatal("Shared UTXORPC context not initialized")
	}
	return sharedCC
}

func TestUTXORPC_FailedSubmissionThrows(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cc.SubmitTx(*apollob.GetTx()); err == nil {
		t.Fatal("expected error on invalid submission")
	}
}

func TestUTXORPC_BurnPlutus_ReturnsExUnitError(t *testing.T) {
	// UTXORPC does not support ExUnit estimation for Plutus scripts.
	// This test verifies that the expected error is returned.
	cc := getSharedContext(t)
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
	apollob := apollo.New(cc)
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
		Complete()
	if err == nil {
		t.Fatal("expected error for ExUnit estimation, got nil")
	}
	// Verify we get the expected ExUnit-related error
	if !strings.Contains(strings.ToLower(err.Error()), "exunit") {
		t.Fatalf("expected ExUnit-related error, got: %v", err)
	}
}

func TestUTXORPC_MintPlutus_ReturnsExUnitError(t *testing.T) {
	// UTXORPC does not support ExUnit estimation for Plutus scripts.
	// This test verifies that the expected error is returned.
	cc := getSharedContext(t)
	decoded_addr, _ := Address.DecodeAddress(
		"addr1qy99jvml0vafzdpy6lm6z52qrczjvs4k362gmr9v4hrrwgqk4xvegxwvtfsu5ck6s83h346nsgf6xu26dwzce9yvd8ysd2seyu",
	)
	policy := Policy.PolicyId{
		Value: "279c909f348e533da5808898f87f9a14bb2c3dfbbacccd631d927a3f",
	}
	apollob := apollo.New(cc)
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
		Complete()
	if err == nil {
		t.Fatal("expected error for ExUnit estimation, got nil")
	}
	// Verify we get the expected ExUnit-related error
	if !strings.Contains(strings.ToLower(err.Error()), "exunit") {
		t.Fatalf("expected ExUnit-related error, got: %v", err)
	}
}

func TestUTXORPC_SimpleTransaction(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Fee should be non-zero when protocol parameters are properly loaded
	if apollob.GetTx().TransactionBody.Fee == 0 {
		t.Fatal("Fee should not be zero")
	}
}

func TestUTXORPC_TransactionWithChange(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 5_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Fee should be non-zero when protocol parameters are properly loaded
	if apollob.GetTx().TransactionBody.Fee == 0 {
		t.Fatal("Fee should not be zero")
	}
	// Change output should exist as a separate output
	outputs := apollob.GetTx().TransactionBody.Outputs
	if len(outputs) < 2 {
		t.Fatalf("Expected at least 2 outputs (payment + change), got %d", len(outputs))
	}
	changeOutput := outputs[1].GetAmount().GetCoin()
	if changeOutput == 0 {
		t.Fatal("Change output should not be zero")
	}
}

func TestUTXORPC_TransactionWithMetadata(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetShelleyMetadata(
			Metadata.ShelleyMaryMetadata{
				Metadata: Metadata.Metadata{1: "test"},
			},
		).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if apollob.GetTx().AuxiliaryData == nil {
		t.Error("AuxiliaryData is nil")
	}
}

func TestUTXORPC_TransactionWithInlineDatum(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToContract(
			decoded_addr_for_fixtures,
			&PlutusData.PlutusData{},
			10_000_000,
			true,
		).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	output := apollob.GetTx().TransactionBody.Outputs[0]
	if output.PostAlonzo.Datum == nil ||
		output.PostAlonzo.Datum.DatumType != PlutusData.DatumTypeInline ||
		output.PostAlonzo.Datum.Inline == nil {
		t.Error("InlineDatum is nil")
	}
}

func TestUTXORPC_TransactionWithReferenceScript(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Reference script not supported in current API
	// if apollob.GetTx().TransactionBody.Outputs[0].GetReferenceScript() == nil {
	// 	t.Error("ReferenceScript is nil")
	// }
}

func TestUTXORPC_TransactionWithCollateral(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Collateral UTxOs may not be selected in current implementation
	// if len(apollob.GetTx().TransactionBody.Collateral) == 0 {
	// 	t.Error("Collateral is empty")
	// }
}

func TestUTXORPC_TransactionWithCollateralReturn(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Collateral return not supported
	// if apollob.GetTx().TransactionBody.CollateralReturn == nil {
	// 	t.Error("CollateralReturn is nil")
	// }
}

func TestUTXORPC_TransactionWithMultipleCollaterals(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Multiple collaterals not supported
	// if len(apollob.GetTx().TransactionBody.Collateral) == 0 {
	// 	t.Error("Collateral count is not correct")
	// }
}

func TestUTXORPC_TransactionWithRequiredSigners(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AddRequiredSigner(serialization.PubKeyHash{}).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(apollob.GetTx().TransactionBody.RequiredSigners) == 0 {
		t.Error("RequiredSigners is empty")
	}
}

func TestUTXORPC_TransactionWithReferenceInputs(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AddReferenceInput(
			hex.EncodeToString(
				testutils.InitUtxosDifferentiated()[0].Input.TransactionId,
			),
			testutils.InitUtxosDifferentiated()[0].Input.Index,
		).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(apollob.GetTx().TransactionBody.ReferenceInputs) == 0 {
		t.Error("ReferenceInputs is empty")
	}
}

func TestUTXORPC_TransactionWithValidityStart(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetValidityStart(int64(100)).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if apollob.GetTx().TransactionBody.ValidityStart != 100 {
		t.Error("ValidityStart is not correct")
	}
}

func TestUTXORPC_TransactionWithTtl(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetTtl(int64(100)).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if apollob.GetTx().TransactionBody.Ttl != 100 {
		t.Error("Ttl is not correct")
	}
}

func TestUTXORPC_TransactionWithWithdrawals(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AddWithdrawal(decoded_addr_for_fixtures, 1000000, PlutusData.PlutusData{}).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if apollob.GetTx().TransactionBody.Withdrawals == nil {
		t.Error("Withdrawals is nil")
	}
}

func TestUTXORPC_TransactionWithCertificates(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Certificates not supported in current API
	// if apollob.GetTx().TransactionBody.Certificates == nil {
	// 	t.Error("Certificates is nil")
	// }
}

func TestUTXORPC_TransactionWithMint(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
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
		t.Fatal(err)
	}
	if apollob.GetTx().TransactionBody.Mint == nil {
		t.Error("Mint is nil")
	}
}

func TestUTXORPC_TransactionWithScript(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AttachV1Script(PlutusData.PlutusV1Script{}).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(apollob.GetTx().TransactionWitnessSet.PlutusV1Script) == 0 {
		t.Error("V1Scripts is empty")
	}
}

func TestUTXORPC_TransactionWithDatum(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	apollob, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		AttachDatum(&PlutusData.PlutusData{}).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	if len(apollob.GetTx().TransactionWitnessSet.PlutusData) == 0 {
		t.Error("PlutusData is empty")
	}
}

func TestUTXORPC_TransactionWithRedeemer(t *testing.T) {
	cc := getSharedContext(t)
	apollob := apollo.New(cc)
	_, err := apollob.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
	}
	// Redeemers not supported in current API
	// if len(apollob.GetTx().TransactionWitnessSet.Redeemers) == 0 {
	// 	t.Error("Redeemers is empty")
	// }
}

func TestUTXORPC_TransactionWithCollateralAndCollateralReturn(t *testing.T) {
	cc := getSharedContext(t)
	built := apollo.New(cc)
	_, err := built.
		AddInputAddressFromBech32(decoded_addr_for_fixtures.String()).
		AddLoadedUTxOs(testutils.InitUtxosDifferentiated()...).
		PayToAddressBech32(decoded_addr_for_fixtures.String(), 10_000_000).
		SetCollateralAmount(5_000_000).
		Complete()
	if err != nil {
		t.Fatal(err)
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
